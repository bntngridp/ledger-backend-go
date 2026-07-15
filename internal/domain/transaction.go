package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	TransactionID       uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"transaction_id"`
	SourceWalletID      *uuid.UUID       `gorm:"type:uuid;index" json:"source_wallet_id"`
	SourceWallet        *Wallet          `gorm:"foreignKey:SourceWalletID" json:"source_wallet,omitempty"`
	DestinationWalletID *uuid.UUID       `gorm:"type:uuid;index" json:"destination_wallet_id"`
	DestinationWallet   *Wallet          `gorm:"foreignKey:DestinationWalletID" json:"destination_wallet,omitempty"`
	AssetSymbol         string           `gorm:"type:varchar(10);not null;default:'IDR'" json:"asset_symbol"`
	Amount              decimal.Decimal  `gorm:"type:decimal(36,18);not null" json:"amount"`
	Type                string           `gorm:"type:varchar(30);not null" json:"type"`
	Status              string           `gorm:"type:varchar(20);not null" json:"status"`
	TransactionNotes    string           `gorm:"type:text" json:"transaction_notes"`
	TxHash              *string          `gorm:"type:varchar(100);uniqueIndex;default:null" json:"tx_hash,omitempty"`
	MidtransOrderID     *string          `gorm:"type:varchar(100);uniqueIndex;default:null" json:"midtrans_order_id,omitempty"`
	RateUsed            *decimal.Decimal `gorm:"type:decimal(20,8);default:null" json:"rate_used,omitempty"`
	FeeCharged          *decimal.Decimal `gorm:"type:decimal(36,18);default:null" json:"fee_charged,omitempty"`
	CreatedAt           time.Time        `gorm:"autoCreateTime" json:"created_at"`
}
