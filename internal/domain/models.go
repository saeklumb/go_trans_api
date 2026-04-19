package domain

import "time"

type TransferRequest struct {
	FromUserID  int64  `json:"from_user_id"`
	ToUserID    int64  `json:"to_user_id"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type TransferResult struct {
	TransactionID int64     `json:"transaction_id"`
	FromUserID    int64     `json:"from_user_id"`
	ToUserID      int64     `json:"to_user_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type WalletBalance struct {
	UserID  int64 `json:"user_id"`
	Balance int64 `json:"balance"`
}

type TransactionEvent struct {
	TransactionID int64     `json:"transaction_id"`
	FromUserID    int64     `json:"from_user_id"`
	ToUserID      int64     `json:"to_user_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
