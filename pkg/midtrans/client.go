package midtrans

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/shopspring/decimal"
)

type Client interface {
	CreateSnapTransaction(orderID string, amount decimal.Decimal, email string, name string) (*snap.Response, error)
	VerifySignature(orderID, statusCode, grossAmount, receivedSignature string) bool
}

type clientImpl struct {
	serverKey  string
	snapClient snap.Client
}

func NewMidtransClient(serverKey string, isProduction bool) Client {
	var env midtrans.EnvironmentType
	if isProduction {
		env = midtrans.Production
	} else {
		env = midtrans.Sandbox
	}

	snapClient := snap.Client{}
	snapClient.New(serverKey, env)

	return &clientImpl{
		serverKey:  serverKey,
		snapClient: snapClient,
	}
}

func (c *clientImpl) CreateSnapTransaction(orderID string, amount decimal.Decimal, email string, name string) (*snap.Response, error) {
	// Round to integer since Midtrans only supports whole numbers for IDR
	amountInt := amount.IntPart()

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: amountInt,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: name,
			Email: email,
		},
	}

	log.Printf("creating midtrans snap transaction for order: %s, amount: %d", orderID, amountInt)
	resp, err := c.snapClient.CreateTransaction(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans error: %w", err)
	}

	return resp, nil
}

func (c *clientImpl) VerifySignature(orderID, statusCode, grossAmount, receivedSignature string) bool {
	// Formula: SHA512(order_id + status_code + gross_amount + server_key)
	raw := orderID + statusCode + grossAmount + c.serverKey
	hash := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(hash[:])
	return expected == receivedSignature
}
