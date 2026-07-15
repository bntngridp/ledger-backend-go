package usecase

import (
	"errors"
	"fmt"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletUsecase interface {
	TopUp(userID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*domain.TopUpResponse, error)
	GetTransactionHistory(userID uuid.UUID) ([]domain.TransactionHistoryItem, error)
	GetDashboard(userID uuid.UUID) (*domain.DashboardResponse, error)
}

type walletUsecase struct {
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
}

func NewWalletUsecase(walletRepo domain.WalletRepository, txRepo domain.TransactionRepository) WalletUsecase {
	return &walletUsecase{
		walletRepo: walletRepo,
		txRepo:     txRepo,
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

	tx, newBalance, err := uc.txRepo.ExecuteTopUpTx(wallet.WalletID, amount, assetSymbol, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to execute top-up: %w", err)
	}

	return &domain.TopUpResponse{
		TransactionID: tx.TransactionID.String(),
		WalletID:      wallet.WalletID.String(),
		AssetSymbol:   assetSymbol,
		Amount:        amount,
		NewBalance:    newBalance,
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
