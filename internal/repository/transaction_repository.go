package repository

import (
	"fmt"
	"time"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) ExecuteTransferTx(sender, recipient *domain.Wallet, amount int64, notes string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var senderBalance int64
		var recipientBalance int64

		row := tx.Raw(`SELECT balance FROM wallets WHERE wallet_id = ? FOR UPDATE`, sender.WalletID).Row()
		if err := row.Scan(&senderBalance); err != nil {
			return fmt.Errorf("failed to lock sender wallet: %w", err)
		}

		row2 := tx.Raw(`SELECT balance FROM wallets WHERE wallet_id = ? FOR UPDATE`, recipient.WalletID).Row()
		if err := row2.Scan(&recipientBalance); err != nil {
			return fmt.Errorf("failed to lock recipient wallet: %w", err)
		}

		if senderBalance < amount {
			return fmt.Errorf("insufficient balance: have %d, need %d", senderBalance, amount)
		}

		newSenderBalance := senderBalance - amount
		newRecipientBalance := recipientBalance + amount

		if err := tx.Model(&domain.Wallet{}).Where("wallet_id = ?", sender.WalletID).
			Updates(map[string]interface{}{
				"balance":      newSenderBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		if err := tx.Model(&domain.Wallet{}).Where("wallet_id = ?", recipient.WalletID).
			Updates(map[string]interface{}{
				"balance":      newRecipientBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to credit recipient: %w", err)
		}

		transaction := domain.Transaction{
			TransactionID:       uuid.New(),
			SourceWalletID:      &sender.WalletID,
			DestinationWalletID: &recipient.WalletID,
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

func (r *transactionRepository) ExecuteTopUpTx(wallet *domain.Wallet, amount int64, notes string) (*domain.Transaction, int64, error) {
	var newBalance int64
	var transaction domain.Transaction

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var currentBalance int64
		row := tx.Raw(`SELECT balance FROM wallets WHERE wallet_id = ? FOR UPDATE`, wallet.WalletID).Row()
		if err := row.Scan(&currentBalance); err != nil {
			return fmt.Errorf("failed to lock wallet: %w", err)
		}

		newBalance = currentBalance + amount

		if err := tx.Model(&domain.Wallet{}).Where("wallet_id = ?", wallet.WalletID).
			Updates(map[string]interface{}{
				"balance":      newBalance,
				"last_updated": time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update wallet balance: %w", err)
		}

		transaction = domain.Transaction{
			TransactionID:       uuid.New(),
			SourceWalletID:      nil,
			DestinationWalletID: &wallet.WalletID,
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
		return nil, 0, err
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
