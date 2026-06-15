package usecase

import (
	"errors"
	"time"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func (uc *authUsecase) Login(email, password, jwtSecret string, expiryHours int) (*domain.LoginResponse, error) {
	user, err := uc.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

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
		return nil, err
	}

	return &domain.LoginResponse{
		Token:     tokenStr,
		ExpiresIn: expiryHours,
	}, nil
}
