package usecase

import (
	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) GetWalletByUserID(userID uuid.UUID) (*domain.Wallet, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletBalance(walletID uuid.UUID, assetSymbol string) (*domain.WalletBalance, error) {
	args := m.Called(walletID, assetSymbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WalletBalance), args.Error(1)
}

func (m *MockWalletRepository) GetBalancesByWalletID(walletID uuid.UUID) ([]domain.WalletBalance, error) {
	args := m.Called(walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WalletBalance), args.Error(1)
}

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) ExecuteTransferTx(senderWalletID, recipientWalletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) error {
	args := m.Called(senderWalletID, recipientWalletID, amount, assetSymbol, notes)
	return args.Error(0)
}

func (m *MockTransactionRepository) ExecuteTopUpTx(walletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*domain.Transaction, decimal.Decimal, error) {
	args := m.Called(walletID, amount, assetSymbol, notes)
	if args.Get(0) == nil {
		return nil, args.Get(1).(decimal.Decimal), args.Error(2)
	}
	return args.Get(0).(*domain.Transaction), args.Get(1).(decimal.Decimal), args.Error(2)
}

func (m *MockTransactionRepository) GetTransactionsByWalletID(walletID uuid.UUID) ([]domain.Transaction, error) {
	args := m.Called(walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUserByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByID(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) CheckEmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) CheckUsernameExists(username string) (bool, error) {
	args := m.Called(username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) CreateUserWithWallet(user *domain.User, wallet *domain.Wallet) error {
	args := m.Called(user, wallet)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByGoogleID(googleID string) (*domain.User, error) {
	args := m.Called(googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

type MockCryptoAddressRepository struct {
	mock.Mock
}

func (m *MockCryptoAddressRepository) GetAddressByWalletID(walletID uuid.UUID, network, assetSymbol string) (*domain.CryptoAddress, error) {
	args := m.Called(walletID, network, assetSymbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CryptoAddress), args.Error(1)
}

func (m *MockCryptoAddressRepository) GetAddressByValue(address string) (*domain.CryptoAddress, error) {
	args := m.Called(address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CryptoAddress), args.Error(1)
}

func (m *MockCryptoAddressRepository) CreateAddress(cryptoAddr *domain.CryptoAddress) error {
	args := m.Called(cryptoAddr)
	return args.Error(0)
}
