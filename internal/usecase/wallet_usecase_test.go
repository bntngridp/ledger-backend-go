package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	}

	expectedTxID := uuid.New()
	expectedTx := &domain.Transaction{
		TransactionID:       expectedTxID,
		DestinationWalletID: &walletID,
		Amount:              decimal.NewFromInt(100000),
		Type:                "topup",
		Status:              "success",
	}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("ExecuteTopUpTx", walletID, decimal.NewFromInt(100000), "IDR", "topup awal").
		Return(expectedTx, decimal.NewFromInt(150000), nil)

	resp, err := uc.TopUp(userID, decimal.NewFromInt(100000), "IDR", "topup awal")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedTxID.String(), resp.TransactionID)
	assert.Equal(t, walletID.String(), resp.WalletID)
	assert.True(t, decimal.NewFromInt(100000).Equal(resp.Amount))
	assert.True(t, decimal.NewFromInt(150000).Equal(resp.NewBalance))
	mockWalletRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestTopUp_ZeroAmount(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()

	resp, err := uc.TopUp(userID, decimal.Zero, "IDR", "test")

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

	resp, err := uc.TopUp(userID, decimal.NewFromInt(-1000), "IDR", "test")

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

	resp, err := uc.TopUp(userID, decimal.NewFromInt(100000), "IDR", "test")

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

	resp, err := uc.TopUp(userID, decimal.NewFromInt(100000), "IDR", "test")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get wallet")
}

func TestTopUp_ExecuteTxError(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	wallet := &domain.Wallet{WalletID: uuid.New(), UserID: userID}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("ExecuteTopUpTx", wallet.WalletID, decimal.NewFromInt(100000), "IDR", "test").
		Return(nil, decimal.Zero, errors.New("failed to lock wallet"))

	resp, err := uc.TopUp(userID, decimal.NewFromInt(100000), "IDR", "test")

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
	wallet := &domain.Wallet{WalletID: walletID, UserID: userID}

	otherWalletID := uuid.New()
	txTopUp := domain.Transaction{
		TransactionID:       uuid.New(),
		SourceWalletID:      nil,
		DestinationWalletID: &walletID,
		Amount:              decimal.NewFromInt(100000),
		Type:                "topup",
		Status:              "success",
		TransactionNotes:    "topup awal",
		CreatedAt:           time.Now(),
	}
	txTransferIn := domain.Transaction{
		TransactionID:       uuid.New(),
		SourceWalletID:      &otherWalletID,
		DestinationWalletID: &walletID,
		Amount:              decimal.NewFromInt(25000),
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
	wallet := &domain.Wallet{WalletID: walletID, UserID: userID}

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
	wallet := &domain.Wallet{WalletID: uuid.New(), UserID: userID}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)
	mockTxRepo.On("GetTransactionsByWalletID", wallet.WalletID).
		Return(nil, errors.New("query timeout"))

	history, err := uc.GetTransactionHistory(userID)

	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Equal(t, "query timeout", err.Error())
}

func TestGetDashboard_Success(t *testing.T) {
	mockWalletRepo := new(MockWalletRepository)
	mockTxRepo := new(MockTransactionRepository)
	uc := NewWalletUsecase(mockWalletRepo, mockTxRepo)

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &domain.Wallet{
		WalletID: walletID,
		UserID:   userID,
		Balances: []domain.WalletBalance{
			{AssetSymbol: "IDR", Balance: decimal.NewFromInt(50000)},
			{AssetSymbol: "USDT", Balance: decimal.NewFromInt(10)},
			{AssetSymbol: "USDC", Balance: decimal.NewFromInt(5)},
		},
	}

	mockWalletRepo.On("GetWalletByUserID", userID).Return(wallet, nil)

	dashboard, err := uc.GetDashboard(userID)

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Equal(t, walletID.String(), dashboard.WalletID)
	assert.Len(t, dashboard.Balances, 3)

	// 50000*1 + 10*16200 + 5*16180 = 50000 + 162000 + 80900 = 292900
	expectedTotal := decimal.NewFromInt(292900)
	assert.True(t, expectedTotal.Equal(dashboard.EstimatedTotalIDR))
	mockWalletRepo.AssertExpectations(t)
}
