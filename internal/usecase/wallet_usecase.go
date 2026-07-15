package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/bntngridp/ledger-backend/pkg/midtrans"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletUsecase interface {
	TopUp(userID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*domain.TopUpResponse, error)
	GetTransactionHistory(userID uuid.UUID) ([]domain.TransactionHistoryItem, error)
	GetDashboard(userID uuid.UUID) (*domain.DashboardResponse, error)
}

type walletUsecase struct {
	walletRepo     domain.WalletRepository
	txRepo         domain.TransactionRepository
	midtransClient midtrans.Client
}

func NewWalletUsecase(walletRepo domain.WalletRepository, txRepo domain.TransactionRepository, midtransClient midtrans.Client) WalletUsecase {
	return &walletUsecase{
		walletRepo:     walletRepo,
		txRepo:         txRepo,
		midtransClient: midtransClient,
	}
}

func (uc *walletUsecase) TopUp(userID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*domain.TopUpResponse, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be greater than 0")
	}

	wallet, err := uc.walletRepo.GetWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	// Generate a unique Midtrans Order ID
	orderID := fmt.Sprintf("TOPUP-IDR-%s-%d", wallet.WalletID.String()[:8], time.Now().UnixNano())

	// Create pending transaction in database
	txRecord, err := uc.txRepo.CreatePendingTopUpTx(wallet.WalletID, amount, assetSymbol, orderID, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to record pending transaction: %w", err)
	}

	// Fetch user details for Midtrans
	email := "user@example.com"
	name := "User"
	if wallet.User != nil {
		email = wallet.User.Email
		name = wallet.User.Username
	}

	// Initiate Snap Transaction
	snapResp, err := uc.midtransClient.CreateSnapTransaction(orderID, amount, email, name)
	if err != nil {
		// Update transaction status to failed
		_ = uc.txRepo.UpdateTransactionStatus(txRecord.TransactionID, "failed", "Midtrans charge failed: "+err.Error())
		return nil, fmt.Errorf("failed to initiate Midtrans Snap: %w", err)
	}

	return &domain.TopUpResponse{
		TransactionID: txRecord.TransactionID.String(),
		WalletID:      wallet.WalletID.String(),
		AssetSymbol:   assetSymbol,
		Amount:        amount,
		SnapToken:     snapResp.Token,
		RedirectURL:   snapResp.RedirectURL,
	}, nil
}

func (uc *walletUsecase) GetTransactionHistory(userID uuid.UUID) ([]domain.TransactionHistoryItem, error) {
	wallet, err := uc.walletRepo.GetWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	transactions, err := uc.txRepo.GetTransactionsByWalletID(wallet.WalletID)
	if err != nil {
		return nil, err
	}

	var result []domain.TransactionHistoryItem
	for _, t := range transactions {
		item := domain.TransactionHistoryItem{
			TransactionID:    t.TransactionID.String(),
			AssetSymbol:      t.AssetSymbol,
			Amount:           t.Amount,
			Type:             t.Type,
			Status:           t.Status,
			TransactionNotes: t.TransactionNotes,
			TxHash:           t.TxHash,
			MidtransOrderID:  t.MidtransOrderID,
			CreatedAt:        t.CreatedAt,
		}
		if t.SourceWalletID != nil {
			s := t.SourceWalletID.String()
			item.SourceWalletID = &s
		}
		if t.DestinationWalletID != nil {
			d := t.DestinationWalletID.String()
			item.DestinationWalletID = &d
		}
		result = append(result, item)
	}

	return result, nil
}

func (uc *walletUsecase) GetDashboard(userID uuid.UUID) (*domain.DashboardResponse, error) {
	wallet, err := uc.walletRepo.GetWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	var balancesDTO []domain.WalletBalanceDTO
	estimatedTotalIDR := decimal.Zero

	// Hardcoded fallback rates before Binance API implementation
	rates := map[string]decimal.Decimal{
		"IDR":  decimal.NewFromInt(1),
		"USDT": decimal.NewFromInt(16200),
		"USDC": decimal.NewFromInt(16180),
	}

	for _, b := range wallet.Balances {
		balancesDTO = append(balancesDTO, domain.WalletBalanceDTO{
			AssetSymbol: b.AssetSymbol,
			Balance:     b.Balance,
		})

		rate, exists := rates[b.AssetSymbol]
		if !exists {
			rate = decimal.Zero
		}
		estimatedTotalIDR = estimatedTotalIDR.Add(b.Balance.Mul(rate))
	}

	return &domain.DashboardResponse{
		WalletID:          wallet.WalletID.String(),
		Balances:          balancesDTO,
		EstimatedTotalIDR: estimatedTotalIDR,
	}, nil
}
