package postgres

import (
	"context"
	"errors"
	"fmt"

	"go-project/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrWalletNotFound    = errors.New("wallet not found")
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Transfer(ctx context.Context, req domain.TransferRequest) (domain.TransferResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.TransferResult{}, err
	}
	defer tx.Rollback(ctx)

	ordered := []int64{req.FromUserID, req.ToUserID}
	if ordered[0] > ordered[1] {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}

	q := "SELECT user_id, balance FROM wallets WHERE user_id = ANY($1) ORDER BY user_id FOR UPDATE"
	rows, err := tx.Query(ctx, q, ordered)
	if err != nil {
		return domain.TransferResult{}, err
	}
	defer rows.Close()

	balances := map[int64]int64{}
	for rows.Next() {
		var userID int64
		var balance int64
		if err = rows.Scan(&userID, &balance); err != nil {
			return domain.TransferResult{}, err
		}
		balances[userID] = balance
	}
	if rows.Err() != nil {
		return domain.TransferResult{}, rows.Err()
	}
	if _, ok := balances[req.FromUserID]; !ok {
		return domain.TransferResult{}, ErrWalletNotFound
	}
	if _, ok := balances[req.ToUserID]; !ok {
		return domain.TransferResult{}, ErrWalletNotFound
	}
	if balances[req.FromUserID] < req.Amount {
		return domain.TransferResult{}, ErrInsufficientFunds
	}

	if _, err = tx.Exec(ctx, "UPDATE wallets SET balance = balance - $1 WHERE user_id = $2", req.Amount, req.FromUserID); err != nil {
		return domain.TransferResult{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE wallets SET balance = balance + $1 WHERE user_id = $2", req.Amount, req.ToUserID); err != nil {
		return domain.TransferResult{}, err
	}

	var result domain.TransferResult
	err = tx.QueryRow(
		ctx,
		`INSERT INTO transactions (from_user_id, to_user_id, amount, description, status)
		 VALUES ($1, $2, $3, $4, 'success')
		 RETURNING id, from_user_id, to_user_id, amount, status, created_at`,
		req.FromUserID, req.ToUserID, req.Amount, req.Description,
	).Scan(
		&result.TransactionID,
		&result.FromUserID,
		&result.ToUserID,
		&result.Amount,
		&result.Status,
		&result.CreatedAt,
	)
	if err != nil {
		return domain.TransferResult{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return domain.TransferResult{}, err
	}

	return result, nil
}

func (s *Store) GetWallet(ctx context.Context, userID int64) (domain.WalletBalance, error) {
	var out domain.WalletBalance
	err := s.db.QueryRow(ctx, "SELECT user_id, balance FROM wallets WHERE user_id = $1", userID).Scan(&out.UserID, &out.Balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WalletBalance{}, fmt.Errorf("%w: user_id=%d", ErrWalletNotFound, userID)
		}
		return domain.WalletBalance{}, err
	}
	return out, nil
}
