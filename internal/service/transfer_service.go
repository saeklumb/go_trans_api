package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-project/internal/domain"
	"go-project/internal/repository/postgres"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrDuplicateInFlight = errors.New("duplicate request in progress")
)

type WalletRepository interface {
	Transfer(ctx context.Context, req domain.TransferRequest) (domain.TransferResult, error)
	GetWallet(ctx context.Context, userID int64) (domain.WalletBalance, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (string, bool, error)
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
	Save(ctx context.Context, key string, payload string, ttl time.Duration) error
}

type EventPublisher interface {
	PublishTransaction(ctx context.Context, event domain.TransactionEvent) error
}

type TransferService struct {
	walletRepo         WalletRepository
	idempotencyRepo    IdempotencyRepository
	publisher          EventPublisher
	idempotencyTTL     time.Duration
	idempotencyLockTTL time.Duration
}

func NewTransferService(
	walletRepo WalletRepository,
	idempotencyRepo IdempotencyRepository,
	publisher EventPublisher,
	idempotencyTTL time.Duration,
	idempotencyLockTTL time.Duration,
) *TransferService {
	return &TransferService{
		walletRepo:         walletRepo,
		idempotencyRepo:    idempotencyRepo,
		publisher:          publisher,
		idempotencyTTL:     idempotencyTTL,
		idempotencyLockTTL: idempotencyLockTTL,
	}
}

func (s *TransferService) Transfer(ctx context.Context, req domain.TransferRequest, idemKey string) (domain.TransferResult, error) {
	if idemKey == "" {
		return domain.TransferResult{}, fmt.Errorf("%w: empty idempotency key", ErrInvalidRequest)
	}
	if req.FromUserID <= 0 || req.ToUserID <= 0 || req.Amount <= 0 || req.FromUserID == req.ToUserID {
		return domain.TransferResult{}, fmt.Errorf("%w: invalid transfer fields", ErrInvalidRequest)
	}

	cached, found, err := s.idempotencyRepo.Get(ctx, idemKey)
	if err != nil {
		return domain.TransferResult{}, err
	}
	if found {
		var result domain.TransferResult
		if err = json.Unmarshal([]byte(cached), &result); err != nil {
			return domain.TransferResult{}, err
		}
		return result, nil
	}

	locked, err := s.idempotencyRepo.TryLock(ctx, idemKey, s.idempotencyLockTTL)
	if err != nil {
		return domain.TransferResult{}, err
	}
	if !locked {
		return domain.TransferResult{}, ErrDuplicateInFlight
	}
	defer s.idempotencyRepo.Unlock(ctx, idemKey)

	result, err := s.walletRepo.Transfer(ctx, req)
	if err != nil {
		return domain.TransferResult{}, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return domain.TransferResult{}, err
	}
	if err = s.idempotencyRepo.Save(ctx, idemKey, string(payload), s.idempotencyTTL); err != nil {
		return domain.TransferResult{}, err
	}

	event := domain.TransactionEvent{
		TransactionID: result.TransactionID,
		FromUserID:    result.FromUserID,
		ToUserID:      result.ToUserID,
		Amount:        result.Amount,
		Status:        result.Status,
		CreatedAt:     result.CreatedAt,
	}
	if err = s.publisher.PublishTransaction(ctx, event); err != nil {
		return domain.TransferResult{}, err
	}

	return result, nil
}

func (s *TransferService) GetWallet(ctx context.Context, userID int64) (domain.WalletBalance, error) {
	if userID <= 0 {
		return domain.WalletBalance{}, fmt.Errorf("%w: invalid user id", ErrInvalidRequest)
	}
	return s.walletRepo.GetWallet(ctx, userID)
}

func IsBusinessError(err error) bool {
	return errors.Is(err, ErrInvalidRequest) ||
		errors.Is(err, ErrDuplicateInFlight) ||
		errors.Is(err, postgres.ErrInsufficientFunds) ||
		errors.Is(err, postgres.ErrWalletNotFound)
}
