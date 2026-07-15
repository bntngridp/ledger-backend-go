package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletBalance struct {
	BalanceID   uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"balance_id"`
	WalletID    uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_wallet_asset" json:"wallet_id"`
	Wallet      *Wallet         `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
	AssetSymbol string          `gorm:"type:varchar(10);not null;uniqueIndex:idx_wallet_asset" json:"asset_symbol"` // 'IDR', 'USDT', 'USDC'
	Balance     decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"balance"`
	LastUpdated time.Time       `gorm:"autoUpdateTime" json:"last_updated"`
}
