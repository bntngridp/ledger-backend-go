package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTopUp_Success(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &domain.Wallet{
		WalletID: walletID,
		UserID:   userID,
		Balance:  50000,
	}

	expectedTxID := uuid.New()
	expectedTx := &domain.Transaction{
		TransactionID:       expectedTxID,
		DestinationWalletID: &walletID,
		Amount:              100000,
		Type:                "topup",
		Status:              "success",
	}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("ExecuteTopUpTx", wallet, int64(100000), "topup awal").
		Return(expectedTx, int64(150000), nil)

	resp, err := uc.TopUp(userID, 100000, "topup awal")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedTxID.String(), resp.TransactionID)
	assert.Equal(t, walletID.String(), resp.WalletID)
	assert.Equal(t, int64(100000), resp.Amount)
	assert.Equal(t, int64(150000), resp.NewBalance)
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestTopUp_ZeroAmount(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	resp, err := uc.TopUp(userID, 0, "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "amount must be greater than 0", err.Error())
	mockWalletRepo.AssertNotCalled(t, "GetWalletByUserID")
	mockTxRepo.AssertNotCalled(t, "ExecuteTopUpTx")
}

func TestTopUp_NegativeAmount(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	resp, err := uc.TopUp(userID, -1000, "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "amount must be greater than 0", err.Error())
}

func TestTopUp_WalletNotFound(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	mockWalletRepo.On("GetWalletByUserID", userID).Return(nil, nil)

	resp, err := uc.TopUp(userID, 100000, "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "wallet not found", err.Error())
	mockTxRepo.AssertNotCalled(t, "ExecuteTopUpTx")
}

func TestTopUp_GetWalletError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	mockWalletRepo.On("GetWalletByUserID", userID).
		Return(nil, errors.New("db error"))

	resp, err := uc.TopUp(userID, 100000, "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get wallet")
}

func TestTopUp_ExecuteTxError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	wallet := &domain.Wallet{WalletID: uuid.New(), UserID: userID, Balance: 0}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("ExecuteTopUpTx", wallet, int64(100000), "test").
		Return(nil, int64(0), errors.New("failed to lock wallet"))

	resp, err := uc.TopUp(userID, 100000, "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to execute top-up")
}

func TestGetTransactionHistory_Success(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &domain.Wallet{WalletID: walletID, UserID: userID, Balance: 100000}

	otherWalletID := uuid.New()
	txTopUp := domain.Transaction{
		TransactionID:       uuid.New(),
		SourceWalletID:      nil,
		DestinationWalletID: &walletID,
		Amount:              100000,
		Type:                "topup",
		Status:              "success",
		TransactionNotes:    "topup awal",
		CreatedAt:           time.Now(),
	}
	txTransferIn := domain.Transaction{
		TransactionID:       uuid.New(),
		SourceWalletID:      &otherWalletID,
		DestinationWalletID: &walletID,
		Amount:              25000,
		Type:                "transfer",
		Status:              "success",
		TransactionNotes:    "from other user",
		CreatedAt:           time.Now().Add(-1 * time.Hour),
	}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("GetTransactionsByWalletID", walletID).
		Return([]domain.Transaction{txTopUp, txTransferIn}, nil)

	history, err := uc.GetTransactionHistory(userID)

	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "topup", history[0].Type)
	assert.Equal(t, "transfer", history[1].Type)
	assert.Nil(t, history[0].SourceWalletID, "topup source_wallet_id should be nil")
	assert.NotNil(t, history[1].SourceWalletID)
	assert.Equal(t, otherWalletID.String(), *history[1].SourceWalletID)
	assert.NotNil(t, history[0].DestinationWalletID)
	assert.Equal(t, walletID.String(), *history[0].DestinationWalletID)
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestGetTransactionHistory_EmptyList(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &domain.Wallet{WalletID: walletID, UserID: userID, Balance: 0}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("GetTransactionsByWalletID", walletID).
		Return([]domain.Transaction{}, nil)

	history, err := uc.GetTransactionHistory(userID)

	assert.NoError(t, err)
	assert.Len(t, history, 0)
}

func TestGetTransactionHistory_WalletNotFound(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	mockWalletRepo.On("GetWalletByUserID", userID).Return(nil, nil)

	history, err := uc.GetTransactionHistory(userID)

	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Equal(t, "wallet not found", err.Error())
	mockTxRepo.AssertNotCalled(t, "GetTransactionsByWalletID")
}

func TestGetTransactionHistory_GetWalletError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	mockWalletRepo.On("GetWalletByUserID", userID).
		Return(nil, errors.New("db connection lost"))

	history, err := uc.GetTransactionHistory(userID)

	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "failed to get wallet")
}

func TestGetTransactionHistory_RepoError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	wallet := &domain.Wallet{WalletID: uuid.New(), UserID: userID, Balance: 100}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("GetTransactionsByWalletID", wallet.WalletID).
		Return(nil, errors.New("query timeout"))

	history, err := uc.GetTransactionHistory(userID)

	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Equal(t, "query timeout", err.Error())
}
