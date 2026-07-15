package usecase

import (
	"errors"
	"testing"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransfer_Success(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	senderWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   senderID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(100000)},
		},
	}
	recipientWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   recipientID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(50000)},
		},
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(recipientWallet, nil)
	mockTxRepo.On("ExecuteTransferTx", senderWallet.WalletID, recipientWallet.WalletID, decimal.NewFromInt(50000), "IDR", "test transfer").Return(nil)

	err := uc.Transfer(senderID, recipientID, decimal.NewFromInt(50000), "IDR", "test transfer")

	assert.NoError(t, err)
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	senderWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   senderID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(10000)},
		},
	}
	recipientWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   recipientID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(50000)},
		},
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(recipientWallet, nil)
	mockTxRepo.On("ExecuteTransferTx", senderWallet.WalletID, recipientWallet.WalletID, decimal.NewFromInt(50000), "IDR", "too much").
		Return(errors.New("insufficient balance: have 10000, need 50000"))

	err := uc.Transfer(senderID, recipientID, decimal.NewFromInt(50000), "IDR", "too much")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance")
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestTransfer_RecipientWalletNotFound(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	senderWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   senderID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(100000)},
		},
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(nil, nil)

	err := uc.Transfer(senderID, recipientID, decimal.NewFromInt(50000), "IDR", "test")

	assert.Error(t, err)
	assert.Equal(t, "recipient wallet not found", err.Error())
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertNotCalled(t, "ExecuteTransferTx")
}

func TestTransfer_SenderWalletNotFound(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(nil, nil)

	err := uc.Transfer(senderID, recipientID, decimal.NewFromInt(50000), "IDR", "test")

	assert.Error(t, err)
	assert.Equal(t, "sender wallet not found", err.Error())
	mockTxRepo.AssertNotCalled(t, "ExecuteTransferTx")
}

func TestTransfer_ZeroAmount(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	err := uc.Transfer(senderID, recipientID, decimal.Zero, "IDR", "test")

	assert.Error(t, err)
	assert.Equal(t, "amount must be greater than 0", err.Error())
	mockWalletRepo.AssertNotCalled(t, "GetWalletByUserID")
	mockTxRepo.AssertNotCalled(t, "ExecuteTransferTx")
}

func TestTransfer_NegativeAmount(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	senderID := uuid.New()
	recipientID := uuid.New()

	err := uc.Transfer(senderID, recipientID, decimal.NewFromInt(-1000), "IDR", "test")

	assert.Error(t, err)
	assert.Equal(t, "amount must be greater than 0", err.Error())
}

func TestTransfer_SelfTransfer(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	err := uc.Transfer(userID, userID, decimal.NewFromInt(50000), "IDR", "test")

	assert.Error(t, err)
	assert.Equal(t, "cannot transfer to yourself", err.Error())
	mockWalletRepo.AssertNotCalled(t, "GetWalletByUserID")
	mockTxRepo.AssertNotCalled(t, "ExecuteTransferTx")
}
