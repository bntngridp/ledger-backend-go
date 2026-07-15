package usecase

import (
	"errors"

	"github.com/bntngridp/ledger-backend/internal/domain"
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

	if user.Password == nil {
		return nil, errors.New("invalid email or password") // or "please login with google" but standard security is to not leak auth type
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	return uc.generateJWTResponse(user, jwtSecret, expiryHours)
}
