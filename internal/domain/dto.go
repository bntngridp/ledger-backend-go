package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

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

type WalletBalanceDTO struct {
	AssetSymbol string          `json:"asset_symbol"`
	Balance     decimal.Decimal `json:"balance"`
}

type RegisterResponse struct {
	UserID   string             `json:"user_id"`
	Username string             `json:"username"`
	Email    string             `json:"email"`
	WalletID string             `json:"wallet_id"`
	Balances []WalletBalanceDTO `json:"balances"`
}

type TransferRequest struct {
	DestinationUserID string          `json:"destination_user_id" binding:"required"`
	AssetSymbol       string          `json:"asset_symbol" binding:"required"`
	Amount            decimal.Decimal `json:"amount" binding:"required,gt=0"`
	Notes             string          `json:"notes"`
}

type TopUpRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required,gt=0"`
	Notes  string          `json:"notes"`
}

type TopUpResponse struct {
	TransactionID string          `json:"transaction_id"`
	WalletID      string          `json:"wallet_id"`
	AssetSymbol   string          `json:"asset_symbol"`
	Amount        decimal.Decimal `json:"amount"`
	SnapToken     string          `json:"snap_token,omitempty"`
	RedirectURL   string          `json:"redirect_url,omitempty"`
}

type TransactionHistoryItem struct {
	TransactionID       string           `json:"transaction_id"`
	SourceWalletID      *string          `json:"source_wallet_id"`
	DestinationWalletID *string          `json:"destination_wallet_id"`
	AssetSymbol         string           `json:"asset_symbol"`
	Amount              decimal.Decimal  `json:"amount"`
	Type                string           `json:"type"`
	Status              string           `json:"status"`
	TransactionNotes    string           `json:"transaction_notes"`
	TxHash              *string          `json:"tx_hash,omitempty"`
	MidtransOrderID     *string          `json:"midtrans_order_id,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
}

type GoogleUserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type DashboardResponse struct {
	WalletID          string             `json:"wallet_id"`
	Balances          []WalletBalanceDTO `json:"balances"`
	EstimatedTotalIDR decimal.Decimal    `json:"estimated_total_idr"`
}
