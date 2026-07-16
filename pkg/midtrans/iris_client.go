// Package midtrans provides a client for the Midtrans BI-SNAP Disbursement API (Sandbox).
package midtrans

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// BIConfig holds credentials and endpoints for BI-SNAP.
type BIConfig struct {
	ClientID       string
	ClientSecret   string
	PartnerID      string
	PrivateKeyPath string
	BaseURL        string
}

// IrisClient wraps the Midtrans BI-SNAP Sandbox API for fiat disbursements.
type IrisClient struct {
	cfg        BIConfig
	httpClient *http.Client

	// Token cache
	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// NewIrisClient creates a new IrisClient using BI-SNAP configuration.
func NewIrisClient(cfg BIConfig) *IrisClient {
	return &IrisClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// IrisBeneficiary represents the recipient of a payout.
type IrisBeneficiary struct {
	Name          string `json:"name"`
	AccountNumber string `json:"account"`
	BankCode      string `json:"bank_account_type"` // e.g., "bca", "mandiri"
	AliasName     string `json:"alias_name"`
	Email         string `json:"email,omitempty"`
}

// IrisPayoutRequest represents a Midtrans Iris payout request body.
type IrisPayoutRequest struct {
	Payouts []IrisPayoutItem `json:"payouts"`
}

// IrisPayoutItem is a single payout entry within a batch.
type IrisPayoutItem struct {
	BeneficiaryName          string `json:"beneficiary_name"`
	BeneficiaryAccountNumber string `json:"beneficiary_account"`
	BeneficiaryBankCode      string `json:"beneficiary_bank"`
	BeneficiaryEmail         string `json:"beneficiary_email,omitempty"`
	Amount                   string `json:"amount"` // in Rupiah, as string
	Notes                    string `json:"notes,omitempty"`
}

// IrisPayoutResponse is the API response for a create payout request.
type IrisPayoutResponse struct {
	Payouts []struct {
		Status          string `json:"status"`
		ReferenceNo     string `json:"reference_no"`
		Amount          string `json:"amount"`
		BeneficiaryName string `json:"beneficiary_name"`
	} `json:"payouts"`
}

// CreatePayout submits a single disbursement request using Midtrans BI-SNAP Transfer API.
func (c *IrisClient) CreatePayout(req IrisPayoutItem) (*IrisPayoutResponse, error) {
	// 1. Get access token
	token, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get BI-SNAP access token: %w", err)
	}

	// 2. Prepare request parameters
	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")

	valDec, err := decimal.NewFromString(req.Amount)
	if err != nil {
		valDec = decimal.NewFromInt(0)
	}
	formattedAmount := valDec.StringFixed(2)

	partnerRef := req.Notes
	if partnerRef == "" {
		partnerRef = "REF-" + time.Now().Format("20060102150405")
	}

	// BI-SNAP standard payload for transfer/disbursement
	snapReqBody, err := json.Marshal(map[string]interface{}{
		"partnerReferenceNo":   partnerRef,
		"beneficiaryAccountNo": req.BeneficiaryAccountNumber,
		"beneficiaryBankCode":  req.BeneficiaryBankCode,
		"beneficiaryName":      req.BeneficiaryName,
		"amount": map[string]string{
			"value":    formattedAmount,
			"currency": "IDR",
		},
		"remark": req.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BI-SNAP request body: %w", err)
	}

	// 3. Generate signature
	bodyHash := sha256Hex(snapReqBody)
	endpoint := "/v1.0/transfer-b2b"
	stringToSign := fmt.Sprintf("POST:%s:%s:%s:%s", endpoint, token, bodyHash, timestamp)
	signature := signHMAC512(c.cfg.ClientSecret, stringToSign)

	// 4. Send request
	url := c.cfg.BaseURL + endpoint
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(snapReqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create BI-SNAP payout request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", signature)
	httpReq.Header.Set("X-PARTNER-ID", c.cfg.PartnerID)
	httpReq.Header.Set("X-EXTERNAL-ID", partnerRef)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("BI-SNAP payout request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read BI-SNAP response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("BI-SNAP payout returned error status %d: %s", resp.StatusCode, string(respBody))
	}

	// 5. Parse response
	var snapResp struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
		ReferenceNo     string `json:"referenceNo"`
	}
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return nil, fmt.Errorf("failed to parse BI-SNAP response: %w", err)
	}

	// 6. Map to Legacy response struct to prevent breaking other layers
	status := "pending"
	// SNAP-BI Success codes: "2001100" (Transfer Success) or "2000000" (Success)
	if snapResp.ResponseCode == "2001100" || snapResp.ResponseCode == "2000000" {
		status = "success"
	}

	payoutResp := &IrisPayoutResponse{
		Payouts: []struct {
			Status          string `json:"status"`
			ReferenceNo     string `json:"reference_no"`
			Amount          string `json:"amount"`
			BeneficiaryName string `json:"beneficiary_name"`
		}{
			{
				Status:          status,
				ReferenceNo:     snapResp.ReferenceNo,
				Amount:          req.Amount,
				BeneficiaryName: req.BeneficiaryName,
			},
		},
	}

	return payoutResp, nil
}

// getAccessToken returns the cached B2B access token or fetches a new one.
func (c *IrisClient) getAccessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.tokenExpiry) {
		return c.token, nil
	}

	privateKey, err := loadPrivateKey(c.cfg.PrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to load private key: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	stringToSign := fmt.Sprintf("%s|%s", c.cfg.ClientID, timestamp)

	signature, err := signRSA(privateKey, stringToSign)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token request: %w", err)
	}

	reqBody, _ := json.Marshal(map[string]string{
		"grantType": "client_credentials",
	})

	baseURL := c.cfg.BaseURL
	if strings.Contains(baseURL, "merchants.sbx.midtrans.com") {
		baseURL = strings.Replace(baseURL, "merchants.sbx.midtrans.com", "merchants-app.sbx.midtrans.com", 1)
	}
	url := baseURL + "/v1.0/access-token/b2b"

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create access token request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-CLIENT-KEY", c.cfg.ClientID)
	httpReq.Header.Set("X-TIMESTAMP", timestamp)
	httpReq.Header.Set("X-SIGNATURE", signature)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("access token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read access token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get access token, status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   string `json:"expiresIn"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse access token response: %w", err)
	}

	c.token = tokenResp.AccessToken
	expiresInSec := 900
	if tokenResp.ExpiresIn != "" {
		fmt.Sscanf(tokenResp.ExpiresIn, "%d", &expiresInSec)
	}
	c.tokenExpiry = time.Now().Add(time.Duration(expiresInSec) * time.Second)

	return c.token, nil
}

// Helper: loadPrivateKey loads a PEM-encoded RSA Private Key from file.
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	// Try PKCS#8
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}

	// Try PKCS#1
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key: unsupported format")
}

// Helper: signRSA signs a stringToSign using RSA-SHA256.
func signRSA(privateKey *rsa.PrivateKey, stringToSign string) (string, error) {
	hashed := sha256.Sum256([]byte(stringToSign))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign with RSA: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Helper: signHMAC512 signs a stringToSign using HMAC-SHA512 with a secret key.
func signHMAC512(secret string, stringToSign string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Helper: sha256Hex hashes data using SHA-256 and returns a lowercase hex string.
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
