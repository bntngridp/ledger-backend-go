package domain

import (
	"time"

	"github.com/google/uuid"
)

type CryptoAddress struct {
	AddressID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"address_id"`
	WalletID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_wallet_network_asset" json:"wallet_id"`
	Wallet        *Wallet   `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
	Network       string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_wallet_network_asset" json:"network"` // 'polygon_amoy', 'sepolia'
	AssetSymbol   string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_wallet_network_asset" json:"asset_symbol"` // 'USDT', 'USDC'
	Address       string    `gorm:"type:varchar(42);not null" json:"address"` // EVM address: 0x...
	EncPrivateKey string    `gorm:"type:text;not null" json:"-"` // AES-256-GCM encrypted private key
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}
