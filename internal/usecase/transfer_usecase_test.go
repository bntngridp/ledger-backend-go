package usecase

import (
	"errors"
	"testing"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/google/uuid"
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
		Balance:  100000,
	}
	recipientWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   recipientID,
		Balance:  50000,
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(recipientWallet, nil)
	mockTxRepo.On("ExecuteTransferTx", senderWallet, recipientWallet, int64(50000), "test transfer").Return(nil)

	err := uc.Transfer(senderID, recipientID, 50000, "test transfer")

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
		Balance:  10000,
	}
	recipientWallet := &domain.Wallet{
		WalletID: uuid.New(),
		UserID:   recipientID,
		Balance:  50000,
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(recipientWallet, nil)
	mockTxRepo.On("ExecuteTransferTx", senderWallet, recipientWallet, int64(50000), "too much").
		Return(errors.New("insufficient balance: have 10000, need 50000"))

	err := uc.Transfer(senderID, recipientID, 50000, "too much")

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
		Balance:  100000,
	}

	mockWalletRepo.On("GetWalletByUserID", senderID).Return(senderWallet, nil)
	mockWalletRepo.On("GetWalletByUserID", recipientID).Return(nil, nil)

	err := uc.Transfer(senderID, recipientID, 50000, "test")

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

	err := uc.Transfer(senderID, recipientID, 50000, "test")

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

	err := uc.Transfer(senderID, recipientID, 0, "test")

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

	err := uc.Transfer(senderID, recipientID, -1000, "test")

	assert.Error(t, err)
	assert.Equal(t, "amount must be greater than 0", err.Error())
}

func TestTransfer_SelfTransfer(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewTransferUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	err := uc.Transfer(userID, userID, 50000, "test")

	assert.Error(t, err)
	assert.Equal(t, "cannot transfer to yourself", err.Error())
	mockWalletRepo.AssertNotCalled(t, "GetWalletByUserID")
	mockTxRepo.AssertNotCalled(t, "ExecuteTransferTx")
}
