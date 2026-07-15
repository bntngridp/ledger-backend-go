package domain

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	WalletID    uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"wallet_id"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Balances        []WalletBalance `gorm:"foreignKey:WalletID" json:"balances,omitempty"`
	CryptoAddresses []CryptoAddress `gorm:"foreignKey:WalletID" json:"crypto_addresses,omitempty"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
}
