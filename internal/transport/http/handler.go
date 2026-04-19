package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-project/internal/domain"
	"go-project/internal/repository/postgres"
	"go-project/internal/service"

	"github.com/go-chi/chi/v5"
)

type TransferService interface {
	Transfer(ctx context.Context, req domain.TransferRequest, idemKey string) (domain.TransferResult, error)
	GetWallet(ctx context.Context, userID int64) (domain.WalletBalance, error)
}

type Handler struct {
	svc *service.TransferService
}

func NewHandler(svc *service.TransferService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Post("/api/v1/transfers", h.createTransfer)
	r.Get("/api/v1/wallets/{userID}", h.getWallet)
	return r
}

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("Idempotency-Key")

	var req domain.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result, err := h.svc.Transfer(r.Context(), req, idemKey)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrDuplicateInFlight):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, postgres.ErrInsufficientFunds):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, postgres.ErrWalletNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getWallet(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	wallet, err := h.svc.GetWallet(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRequest) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, postgres.ErrWalletNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
