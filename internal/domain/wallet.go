package domain

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	WalletID    uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"wallet_id"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Balance     int64     `gorm:"type:bigint;default:0;not null" json:"balance"`
	LastUpdated time.Time `gorm:"autoUpdateTime" json:"last_updated"`
}
