package repository

import (
	"errors"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cryptoAddressRepository struct {
	db *gorm.DB
}

func NewCryptoAddressRepository(db *gorm.DB) domain.CryptoAddressRepository {
	return &cryptoAddressRepository{db: db}
}

func (r *cryptoAddressRepository) GetAddressByWalletID(walletID uuid.UUID, network, assetSymbol string) (*domain.CryptoAddress, error) {
	var cryptoAddr domain.CryptoAddress
	if err := r.db.Where("wallet_id = ? AND network = ? AND asset_symbol = ?", walletID, network, assetSymbol).First(&cryptoAddr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cryptoAddr, nil
}

func (r *cryptoAddressRepository) GetAddressByValue(address string) (*domain.CryptoAddress, error) {
	var cryptoAddr domain.CryptoAddress
	if err := r.db.Where("address = ?", address).First(&cryptoAddr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cryptoAddr, nil
}

func (r *cryptoAddressRepository) CreateAddress(cryptoAddr *domain.CryptoAddress) error {
	return r.db.Create(cryptoAddr).Error
}
