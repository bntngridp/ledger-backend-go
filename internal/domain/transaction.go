package domain

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	TransactionID       uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"transaction_id"`
	SourceWalletID      *uuid.UUID `gorm:"type:uuid;index" json:"source_wallet_id"`
	SourceWallet        *Wallet    `gorm:"foreignKey:SourceWalletID" json:"source_wallet,omitempty"`
	DestinationWalletID *uuid.UUID `gorm:"type:uuid;index" json:"destination_wallet_id"`
	DestinationWallet   *Wallet    `gorm:"foreignKey:DestinationWalletID" json:"destination_wallet,omitempty"`
	Amount              int64      `gorm:"type:bigint;not null;check:amount > 0" json:"amount"`
	Type                string     `gorm:"type:varchar(20);not null" json:"type"`
	Status              string     `gorm:"type:varchar(20);not null" json:"status"`
	TransactionNotes    string     `gorm:"type:text" json:"transaction_notes"`
	CreatedAt           time.Time  `gorm:"autoCreateTime" json:"created_at"`
}
