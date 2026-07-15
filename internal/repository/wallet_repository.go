package repository

import (
	"errors"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) domain.WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) GetWalletByUserID(userID uuid.UUID) (*domain.Wallet, error) {
	var wallet domain.Wallet
	if err := r.db.Preload("User").Preload("Balances").Preload("CryptoAddresses").Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) GetWalletBalance(walletID uuid.UUID, assetSymbol string) (*domain.WalletBalance, error) {
	var balance domain.WalletBalance
	if err := r.db.Where("wallet_id = ? AND asset_symbol = ?", walletID, assetSymbol).First(&balance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &balance, nil
}

func (r *walletRepository) GetBalancesByWalletID(walletID uuid.UUID) ([]domain.WalletBalance, error) {
	var balances []domain.WalletBalance
	if err := r.db.Where("wallet_id = ?", walletID).Find(&balances).Error; err != nil {
		return nil, err
	}
	return balances, nil
}
