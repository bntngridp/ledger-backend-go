package usecase

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bntngridp/ledger-backend/internal/domain"
	pkgcrypto "github.com/bntngridp/ledger-backend/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

func (uc *authUsecase) Generate2FASecret(userID uuid.UUID) (*domain.Enable2FAResponse, error) {
	user, err := uc.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "LedgerHybridWallet",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secret := key.Secret()

	encrypted, err := pkgcrypto.Encrypt([]byte(secret), uc.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}
	encryptedHex := hex.EncodeToString(encrypted)

	if err := uc.userRepo.Update2FA(userID, &encryptedHex, false); err != nil {
		return nil, fmt.Errorf("failed to save 2FA secret: %w", err)
	}

	return &domain.Enable2FAResponse{
		Secret:    secret,
		QRCodeURL: key.URL(),
	}, nil
}

func (uc *authUsecase) Enable2FA(userID uuid.UUID, code string) error {
	user, err := uc.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return domain.ErrNotFound
	}

	if user.TwoFactorSecret == nil {
		return errors.New("2FA secret has not been generated")
	}

	encBytes, err := hex.DecodeString(*user.TwoFactorSecret)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted secret: %w", err)
	}

	decrypted, err := pkgcrypto.Decrypt(encBytes, uc.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret: %w", err)
	}
	secretStr := string(decrypted)

	if !totp.Validate(code, secretStr) {
		return domain.ErrInvalid2FACode
	}

	if err := uc.userRepo.Update2FA(userID, user.TwoFactorSecret, true); err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	return nil
}

func (uc *authUsecase) Disable2FA(userID uuid.UUID, code string) error {
	user, err := uc.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return domain.ErrNotFound
	}

	if !user.TwoFactorEnabled {
		return errors.New("2FA is not enabled")
	}

	if user.TwoFactorSecret == nil {
		return errors.New("2FA secret missing")
	}

	encBytes, err := hex.DecodeString(*user.TwoFactorSecret)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted secret: %w", err)
	}

	decrypted, err := pkgcrypto.Decrypt(encBytes, uc.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret: %w", err)
	}
	secretStr := string(decrypted)

	if !totp.Validate(code, secretStr) {
		return domain.ErrInvalid2FACode
	}

	if err := uc.userRepo.Update2FA(userID, nil, false); err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	return nil
}

func (uc *authUsecase) Verify2FALogin(preAuthToken, code, jwtSecret string, expiryHours int) (*domain.LoginResponse, error) {
	token, err := jwt.Parse(preAuthToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	preAuthClaim, ok := claims["pre_auth"].(bool)
	if !ok || !preAuthClaim {
		return nil, domain.ErrUnauthorized
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	user, err := uc.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}

	if user.TwoFactorSecret == nil {
		return nil, errors.New("2FA secret missing")
	}

	encBytes, err := hex.DecodeString(*user.TwoFactorSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted secret: %w", err)
	}

	decrypted, err := pkgcrypto.Decrypt(encBytes, uc.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}
	secretStr := string(decrypted)

	if !totp.Validate(code, secretStr) {
		return nil, domain.ErrInvalid2FACode
	}

	return uc.generateJWTResponse(user, jwtSecret, expiryHours)
}

func (uc *authUsecase) generatePreAuthToken(user *domain.User, jwtSecret string) (string, error) {
	expiry := time.Now().Add(5 * time.Minute) // 5 minutes challenge timeout

	claims := jwt.MapClaims{
		"user_id":  user.UserID.String(),
		"pre_auth": true,
		"exp":      expiry.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign pre-auth token: %w", err)
	}
	return tokenStr, nil
}

func (uc *authUsecase) Verify2FACode(userID uuid.UUID, code string) error {
	user, err := uc.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return domain.ErrNotFound
	}

	if !user.TwoFactorEnabled {
		return nil
	}

	if code == "" {
		return domain.Err2FARequired
	}

	if user.TwoFactorSecret == nil {
		return errors.New("2FA secret missing")
	}

	encBytes, err := hex.DecodeString(*user.TwoFactorSecret)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted secret: %w", err)
	}

	decrypted, err := pkgcrypto.Decrypt(encBytes, uc.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret: %w", err)
	}
	secretStr := string(decrypted)

	if !totp.Validate(code, secretStr) {
		return domain.ErrInvalid2FACode
	}

	return nil
}
