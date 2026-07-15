package usecase

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/bntngridp/ledger-backend/pkg/midtrans"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WebhookUsecase interface {
	ProcessMidtransNotification(payload map[string]interface{}) error
	ProcessIrisNotification(payload []domain.IrisCallbackItem) error
}

type webhookUsecase struct {
	txRepo         domain.TransactionRepository
	midtransClient midtrans.Client
}

func NewWebhookUsecase(txRepo domain.TransactionRepository, midtransClient midtrans.Client) WebhookUsecase {
	return &webhookUsecase{
		txRepo:         txRepo,
		midtransClient: midtransClient,
	}
}

func (uc *webhookUsecase) ProcessMidtransNotification(payload map[string]interface{}) error {
	orderID, ok := payload["order_id"].(string)
	if !ok || orderID == "" {
		return errors.New("missing or invalid order_id")
	}

	statusCode, ok := payload["status_code"].(string)
	if !ok || statusCode == "" {
		return errors.New("missing or invalid status_code")
	}

	grossAmount, ok := payload["gross_amount"].(string)
	if !ok || grossAmount == "" {
		return errors.New("missing or invalid gross_amount")
	}

	signatureKey, ok := payload["signature_key"].(string)
	if !ok || signatureKey == "" {
		return errors.New("missing or invalid signature_key")
	}

	if !uc.midtransClient.VerifySignature(orderID, statusCode, grossAmount, signatureKey) {
		slog.Warn("invalid midtrans signature received", "order_id", orderID)
		return errors.New("invalid signature key")
	}

	txRecord, err := uc.txRepo.GetTransactionByOrderID(orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch transaction: %w", err)
	}
	if txRecord == nil {
		return fmt.Errorf("transaction not found for order: %s", orderID)
	}

	if txRecord.Status != "pending" {
		slog.Info("transaction already processed", "tx_id", txRecord.TransactionID, "status", txRecord.Status)
		return nil
	}

	amt, err := decimal.NewFromString(grossAmount)
	if err != nil {
		return fmt.Errorf("failed to parse gross_amount: %w", err)
	}

	txStatus, _ := payload["transaction_status"].(string)
	fraudStatus, _ := payload["fraud_status"].(string)

	slog.Info("processing midtrans transaction status", "order_id", orderID, "tx_status", txStatus, "fraud_status", fraudStatus)

	switch txStatus {
	case "capture":
		if fraudStatus == "accept" {
			return uc.settleTransaction(txRecord, amt)
		}
		return uc.failTransaction(txRecord, "Captured but fraud status was "+fraudStatus)
	case "settlement":
		return uc.settleTransaction(txRecord, amt)
	case "deny", "cancel", "expire":
		return uc.failTransaction(txRecord, "Payment "+txStatus)
	default:
		slog.Warn("unhandled transaction status", "tx_status", txStatus)
		return nil
	}
}

func (uc *webhookUsecase) settleTransaction(txRecord *domain.Transaction, amount decimal.Decimal) error {
	if txRecord.DestinationWalletID == nil {
		return errors.New("destination wallet is nil")
	}
	slog.Info("settling transaction", "tx_id", txRecord.TransactionID, "wallet_id", txRecord.DestinationWalletID, "amount", amount.String())
	return uc.txRepo.SettleTopUpTx(txRecord.TransactionID, *txRecord.DestinationWalletID, amount)
}

func (uc *webhookUsecase) failTransaction(txRecord *domain.Transaction, reason string) error {
	slog.Info("failing transaction", "tx_id", txRecord.TransactionID, "reason", reason)
	return uc.txRepo.UpdateTransactionStatus(txRecord.TransactionID, "failed", reason)
}

func (uc *webhookUsecase) ProcessIrisNotification(payload []domain.IrisCallbackItem) error {
	for _, item := range payload {
		txUUID, err := uuid.Parse(item.ReferenceNo)
		if err != nil {
			slog.Warn("invalid reference_no UUID in Iris callback", "reference_no", item.ReferenceNo)
			continue
		}

		status := strings.ToLower(item.Status)
		slog.Info("processing iris callback item", "reference_no", item.ReferenceNo, "status", status)

		if status == "completed" {
			err = uc.txRepo.UpdateTransactionStatus(txUUID, "success", "")
		} else if status == "failed" {
			errMsg := "Iris callback failure"
			if item.ErrorMessage != nil {
				errMsg = *item.ErrorMessage
			}
			err = uc.txRepo.RejectWithdrawFiatTx(txUUID, errMsg)
		} else {
			slog.Warn("unhandled status in Iris callback", "status", item.Status, "reference_no", item.ReferenceNo)
			continue
		}

		if err != nil {
			slog.Error("failed to process Iris callback item", "reference_no", item.ReferenceNo, "error", err)
			return err
		}
	}
	return nil
}
