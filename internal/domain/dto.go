package domain

import "time"

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

type TransferRequest struct {
	DestinationUserID string `json:"destination_user_id" binding:"required"`
	Amount            int64  `json:"amount" binding:"required"`
	Notes             string `json:"notes"`
}

type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	WalletID string `json:"wallet_id"`
	Balance  int64  `json:"balance"`
}

type TopUpRequest struct {
	Amount int64  `json:"amount" binding:"required,gt=0"`
	Notes  string `json:"notes"`
}

type TopUpResponse struct {
	TransactionID string `json:"transaction_id"`
	WalletID      string `json:"wallet_id"`
	Amount        int64  `json:"amount"`
	NewBalance    int64  `json:"new_balance"`
}

type TransactionHistoryItem struct {
	TransactionID       string    `json:"transaction_id"`
	SourceWalletID      *string   `json:"source_wallet_id"`
	DestinationWalletID *string   `json:"destination_wallet_id"`
	Amount              int64     `json:"amount"`
	Type                string    `json:"type"`
	Status              string    `json:"status"`
	TransactionNotes    string    `json:"transaction_notes"`
	CreatedAt           time.Time `json:"created_at"`
}
