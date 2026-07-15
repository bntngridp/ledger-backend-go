package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (uc *authUsecase) LoginWithGoogle(profile *domain.GoogleUserProfile, jwtSecret string, expiryHours int) (*domain.LoginResponse, error) {
	// 1. Check if user already exists by GoogleID
	user, err := uc.userRepo.GetUserByGoogleID(profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check google id: %w", err)
	}

	if user != nil {
		// User exists, update avatar just in case it changed
		user.AvatarURL = &profile.Picture
		if err := uc.userRepo.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to update user avatar: %w", err)
		}
		return uc.generateJWTResponse(user, jwtSecret, expiryHours)
	}

	// 2. GoogleID not found, check if email is registered
	user, err = uc.userRepo.GetUserByEmail(profile.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	if user != nil {
		// Link Google account to existing user account
		user.GoogleID = &profile.ID
		user.AvatarURL = &profile.Picture
		if err := uc.userRepo.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to link google account: %w", err)
		}
		return uc.generateJWTResponse(user, jwtSecret, expiryHours)
	}

	// 3. User is brand new, register new user with wallet
	// Generate unique username from name or email prefix
	usernameBase := profile.Name
	if usernameBase == "" {
		parts := strings.Split(profile.Email, "@")
		usernameBase = parts[0]
	}

	// Sanitize base username
	usernameBase = strings.ReplaceAll(strings.ToLower(usernameBase), " ", "")
	username := usernameBase

	// Check for collision
	exists, err := uc.userRepo.CheckUsernameExists(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username collision: %w", err)
	}

	suffix := 1
	for exists {
		username = fmt.Sprintf("%s%d", usernameBase, suffix)
		exists, err = uc.userRepo.CheckUsernameExists(username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username collision: %w", err)
		}
		suffix++
	}

	userID := uuid.New()
	walletID := uuid.New()

	newUser := &domain.User{
		UserID:    userID,
		Username:  username,
		Email:     profile.Email,
		Password:  nil, // No password for Google users
		GoogleID:  &profile.ID,
		AvatarURL: &profile.Picture,
		IsActive:  true,
	}

	newWallet := &domain.Wallet{
		WalletID: walletID,
		UserID:   userID,
	}

	if err := uc.userRepo.CreateUserWithWallet(newUser, newWallet); err != nil {
		return nil, fmt.Errorf("failed to create user with google: %w", err)
	}

	return uc.generateJWTResponse(newUser, jwtSecret, expiryHours)
}

func (uc *authUsecase) generateJWTResponse(user *domain.User, jwtSecret string, expiryHours int) (*domain.LoginResponse, error) {
	expiry := time.Now().Add(time.Duration(expiryHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": user.UserID.String(),
		"email":   user.Email,
		"exp":     expiry.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &domain.LoginResponse{
		Token:     tokenStr,
		ExpiresIn: expiryHours,
	}, nil
}
