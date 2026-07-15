package repository

import (
	"fmt"
	"time"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) ExecuteTransferTx(senderWalletID, recipientWalletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var senderBalance decimal.Decimal
		var recipientBalance decimal.Decimal

		// Fetch and lock sender balance
		row := tx.Raw(`SELECT balance FROM wallet_balances WHERE wallet_id = ? AND asset_symbol = ? FOR UPDATE`, senderWalletID, assetSymbol).Row()
		if err := row.Scan(&senderBalance); err != nil {
			return fmt.Errorf("failed to lock sender balance: %w", err)
		}

		// Fetch and lock recipient balance
		row2 := tx.Raw(`SELECT balance FROM wallet_balances WHERE wallet_id = ? AND asset_symbol = ? FOR UPDATE`, recipientWalletID, assetSymbol).Row()
		if err := row2.Scan(&recipientBalance); err != nil {
			return fmt.Errorf("failed to lock recipient balance: %w", err)
		}

		if senderBalance.LessThan(amount) {
			return fmt.Errorf("insufficient balance: have %s, need %s", senderBalance.String(), amount.String())
		}

		newSenderBalance := senderBalance.Sub(amount)
		newRecipientBalance := recipientBalance.Add(amount)

		// Update sender balance
		if err := tx.Model(&domain.WalletBalance{}).Where("wallet_id = ? AND asset_symbol = ?", senderWalletID, assetSymbol).
			Updates(map[string]interface{}{
				"balance":      newSenderBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		// Update recipient balance
		if err := tx.Model(&domain.WalletBalance{}).Where("wallet_id = ? AND asset_symbol = ?", recipientWalletID, assetSymbol).
			Updates(map[string]interface{}{
				"balance":      newRecipientBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to credit recipient: %w", err)
		}

		transaction := domain.Transaction{
			TransactionID:       uuid.New(),
			SourceWalletID:      &senderWalletID,
			DestinationWalletID: &recipientWalletID,
			AssetSymbol:         assetSymbol,
			Amount:              amount,
			Type:                "transfer",
			Status:              "success",
			TransactionNotes:    notes,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			return fmt.Errorf("failed to record transaction: %w", err)
		}

		return nil
	})
}

func (r *transactionRepository) ExecuteTopUpTx(walletID uuid.UUID, amount decimal.Decimal, assetSymbol string, notes string) (*domain.Transaction, decimal.Decimal, error) {
	var newBalance decimal.Decimal
	var transaction domain.Transaction

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var currentBalance decimal.Decimal
		row := tx.Raw(`SELECT balance FROM wallet_balances WHERE wallet_id = ? AND asset_symbol = ? FOR UPDATE`, walletID, assetSymbol).Row()
		if err := row.Scan(&currentBalance); err != nil {
			return fmt.Errorf("failed to lock wallet balance: %w", err)
		}

		newBalance = currentBalance.Add(amount)

		if err := tx.Model(&domain.WalletBalance{}).Where("wallet_id = ? AND asset_symbol = ?", walletID, assetSymbol).
			Updates(map[string]interface{}{
				"balance":      newBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update wallet balance: %w", err)
		}

		transaction = domain.Transaction{
			TransactionID:       uuid.New(),
			SourceWalletID:      nil,
			DestinationWalletID: &walletID,
			AssetSymbol:         assetSymbol,
			Amount:              amount,
			Type:                "topup",
			Status:              "success",
			TransactionNotes:    notes,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			return fmt.Errorf("failed to record transaction: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, decimal.Zero, err
	}

	return &transaction, newBalance, nil
}

func (r *transactionRepository) GetTransactionsByWalletID(walletID uuid.UUID) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	err := r.db.Where("source_wallet_id = ? OR destination_wallet_id = ?", walletID, walletID).
		Order("created_at DESC").
		Find(&transactions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	return transactions, nil
}
