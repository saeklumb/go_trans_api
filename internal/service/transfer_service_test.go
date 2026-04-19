package service

import (
	"context"
	"testing"
	"time"

	"go-project/internal/domain"
)

// Простые моки для интерфейсов
type mockWalletRepo struct{}
func (m *mockWalletRepo) Transfer(ctx context.Context, req domain.TransferRequest) (domain.TransferResult, error) {
	return domain.TransferResult{TransactionID: 1, FromUserID: req.FromUserID, ToUserID: req.ToUserID, Amount: req.Amount, Status: "success"}, nil
}
func (m *mockWalletRepo) GetWallet(ctx context.Context, userID int64) (domain.WalletBalance, error) {
	return domain.WalletBalance{UserID: userID, Balance: 1000}, nil
}

type mockIdempotencyRepo struct{}
func (m *mockIdempotencyRepo) Get(ctx context.Context, key string) (string, bool, error) { return "", false, nil }
func (m *mockIdempotencyRepo) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) { return true, nil }
func (m *mockIdempotencyRepo) Unlock(ctx context.Context, key string) error { return nil }
func (m *mockIdempotencyRepo) Save(ctx context.Context, key, payload string, ttl time.Duration) error { return nil }

type mockPublisher struct{}
func (m *mockPublisher) PublishTransaction(ctx context.Context, event domain.TransactionEvent) error { return nil }

func TestTransferService_Transfer_Success(t *testing.T) {
	svc := NewTransferService(
		&mockWalletRepo{},
		&mockIdempotencyRepo{},
		&mockPublisher{},
		time.Minute,
		time.Second,
	)

	req := domain.TransferRequest{
		FromUserID: 1,
		ToUserID:   2,
		Amount:     500,
	}

	result, err := svc.Transfer(context.Background(), req, "unique-test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TransactionID != 1 {
		t.Errorf("expected transaction ID 1, got %d", result.TransactionID)
	}
	if result.Amount != 500 {
		t.Errorf("expected amount 500, got %d", result.Amount)
	}
}

func TestTransferService_Transfer_InvalidRequest(t *testing.T) {
	svc := NewTransferService(&mockWalletRepo{}, &mockIdempotencyRepo{}, &mockPublisher{}, time.Minute, time.Second)
	
	req := domain.TransferRequest{FromUserID: 1, ToUserID: 1, Amount: 500}
	
	_, err := svc.Transfer(context.Background(), req, "key")
	if err == nil {
		t.Error("expected error for transferring to self, got nil")
	}
}