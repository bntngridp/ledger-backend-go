package usecase

import (
	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/google/uuid"
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

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) ExecuteTransferTx(sender, recipient *domain.Wallet, amount int64, notes string) error {
	args := m.Called(sender, recipient, amount, notes)
	return args.Error(0)
}

func (m *MockTransactionRepository) ExecuteTopUpTx(wallet *domain.Wallet, amount int64, notes string) (*domain.Transaction, int64, error) {
	args := m.Called(wallet, amount, notes)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).(*domain.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) GetTransactionsByWalletID(walletID uuid.UUID) ([]domain.Transaction, error) {
	args := m.Called(walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}
