package usecase

import (
	"errors"
	"fmt"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/google/uuid"
)

type WalletUsecase interface {
	TopUp(userID uuid.UUID, amount int64, notes string) (*domain.TopUpResponse, error)
	GetTransactionHistory(userID uuid.UUID) ([]domain.TransactionHistoryItem, error)
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

func (uc *walletUsecase) TopUp(userID uuid.UUID, amount int64, notes string) (*domain.TopUpResponse, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	wallet, err := uc.walletRepo.GetWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	tx, newBalance, err := uc.txRepo.ExecuteTopUpTx(wallet, amount, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to execute top-up: %w", err)
	}

	return &domain.TopUpResponse{
		TransactionID: tx.TransactionID.String(),
		WalletID:      wallet.WalletID.String(),
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
			Amount:           t.Amount,
			Type:             t.Type,
			Status:           t.Status,
			TransactionNotes: t.TransactionNotes,
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
