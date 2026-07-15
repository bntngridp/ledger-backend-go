package repository

import (
	"errors"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByGoogleID(googleID string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("google_id = ?", googleID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("user_id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CheckEmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepository) CheckUsernameExists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepository) CreateUserWithWallet(user *domain.User, wallet *domain.Wallet) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		wallet.UserID = user.UserID
		if err := tx.Create(wallet).Error; err != nil {
			return err
		}

		// Initialize default balances (IDR, USDT, USDC)
		defaultAssets := []string{"IDR", "USDT", "USDC"}
		for _, asset := range defaultAssets {
			balance := &domain.WalletBalance{
				BalanceID:   uuid.New(),
				WalletID:    wallet.WalletID,
				AssetSymbol: asset,
				Balance:     decimal.Zero,
			}
			if err := tx.Create(balance).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *userRepository) UpdateUser(user *domain.User) error {
	return r.db.Save(user).Error
}
