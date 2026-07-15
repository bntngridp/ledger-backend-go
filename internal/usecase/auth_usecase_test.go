package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_Success(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(false, nil)
	mockUserRepo.On("CheckUsernameExists", "budi").Return(false, nil)
	mockUserRepo.On("CreateUserWithWallet", mock.Anything, mock.Anything).Return(nil)

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "budi", resp.Username)
	assert.Equal(t, "budi@mail.com", resp.Email)
	assert.Equal(t, int64(0), resp.Balance)
	assert.NotEmpty(t, resp.UserID)
	assert.NotEmpty(t, resp.WalletID)
	mockUserRepo.AssertExpectations(t)
}

func TestRegister_EmailAlreadyRegistered(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(true, nil)

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "email already registered", err.Error())
	mockUserRepo.AssertNotCalled(t, "CheckUsernameExists")
	mockUserRepo.AssertNotCalled(t, "CreateUserWithWallet")
}

func TestRegister_UsernameAlreadyTaken(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(false, nil)
	mockUserRepo.On("CheckUsernameExists", "budi").Return(true, nil)

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "username already taken", err.Error())
	mockUserRepo.AssertNotCalled(t, "CreateUserWithWallet")
}

func TestRegister_CheckEmailError(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").
		Return(false, errors.New("db connection lost"))

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to check email")
}

func TestRegister_CheckUsernameError(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(false, nil)
	mockUserRepo.On("CheckUsernameExists", "budi").
		Return(false, errors.New("db timeout"))

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to check username")
}

func TestRegister_CreateUserWithWalletError(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(false, nil)
	mockUserRepo.On("CheckUsernameExists", "budi").Return(false, nil)
	mockUserRepo.On("CreateUserWithWallet", mock.Anything, mock.Anything).
		Return(errors.New("unique constraint violation"))

	resp, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create user with wallet")
}

func TestRegister_PasswordIsHashed(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	var capturedUser *domain.User
	mockUserRepo.On("CheckEmailExists", "budi@mail.com").Return(false, nil)
	mockUserRepo.On("CheckUsernameExists", "budi").Return(false, nil)
	mockUserRepo.On("CreateUserWithWallet", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { capturedUser = args.Get(0).(*domain.User) }).
		Return(nil)

	_, err := uc.Register("budi", "budi@mail.com", "secret123")

	assert.NoError(t, err)
	assert.NotNil(t, capturedUser.Password)
	assert.NotEqual(t, "secret123", *capturedUser.Password, "password must be hashed, not plaintext")
	err = bcrypt.CompareHashAndPassword([]byte(*capturedUser.Password), []byte("secret123"))
	assert.NoError(t, err, "hashed password must match the original")
}

func TestLogin_Success(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	userID := uuid.New()
	user := &domain.User{
		UserID:   userID,
		Email:    "budi@mail.com",
		Password: &hashedStr,
	}

	mockUserRepo.On("GetUserByEmail", "budi@mail.com").Return(user, nil)

	resp, err := uc.Login("budi@mail.com", "secret123", "test-secret", 24)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, 24, resp.ExpiresIn)

	parsed, parseErr := jwt.Parse(resp.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	assert.NoError(t, parseErr)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), claims["user_id"])
	assert.Equal(t, "budi@mail.com", claims["email"])
}

func TestLogin_InvalidEmail(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("GetUserByEmail", "nonexistent@mail.com").Return(nil, nil)

	resp, err := uc.Login("nonexistent@mail.com", "any", "secret", 24)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "invalid email or password", err.Error())
}

func TestLogin_WrongPassword(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	user := &domain.User{
		UserID:   uuid.New(),
		Email:    "budi@mail.com",
		Password: &hashedStr,
	}

	mockUserRepo.On("GetUserByEmail", "budi@mail.com").Return(user, nil)

	resp, err := uc.Login("budi@mail.com", "wrongpassword", "secret", 24)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "invalid email or password", err.Error())
}

func TestLogin_GetUserByEmailError(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	mockUserRepo.On("GetUserByEmail", "budi@mail.com").
		Return(nil, errors.New("db error"))

	resp, err := uc.Login("budi@mail.com", "any", "secret", 24)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "db error", err.Error())
}

func TestLogin_TokenExpiryMatchesConfig(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	hashedStr := string(hashed)
	user := &domain.User{
		UserID:   uuid.New(),
		Email:    "budi@mail.com",
		Password: &hashedStr,
	}

	mockUserRepo.On("GetUserByEmail", "budi@mail.com").Return(user, nil)

	beforeLogin := time.Now()
	resp, err := uc.Login("budi@mail.com", "secret123", "secret", 1)
	afterLogin := time.Now()

	assert.NoError(t, err)

	parsed, _ := jwt.Parse(resp.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)
	exp := int64(claims["exp"].(float64))

	expectedMin := beforeLogin.Add(1 * time.Hour).Unix()
	expectedMax := afterLogin.Add(1 * time.Hour).Unix()

	assert.GreaterOrEqual(t, exp, expectedMin)
	assert.LessOrEqual(t, exp, expectedMax)
}

func TestLoginWithGoogle_NewUser(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	profile := &domain.GoogleUserProfile{
		ID:      "google-id-123",
		Email:   "google@mail.com",
		Name:    "Google User",
		Picture: "https://avatar.url",
	}

	mockUserRepo.On("GetUserByGoogleID", "google-id-123").Return(nil, nil)
	mockUserRepo.On("GetUserByEmail", "google@mail.com").Return(nil, nil)
	mockUserRepo.On("CheckUsernameExists", "googleuser").Return(false, nil)
	mockUserRepo.On("CreateUserWithWallet", mock.Anything, mock.Anything).Return(nil)

	resp, err := uc.LoginWithGoogle(profile, "test-secret", 24)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	mockUserRepo.AssertExpectations(t)
}

func TestLoginWithGoogle_ExistingUser(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	uc := NewAuthUsecase(mockUserRepo, mockWalletRepo)

	profile := &domain.GoogleUserProfile{
		ID:      "google-id-123",
		Email:   "google@mail.com",
		Name:    "Google User",
		Picture: "https://avatar.url",
	}

	existingUser := &domain.User{
		UserID: uuid.New(),
		Email:  "google@mail.com",
	}

	mockUserRepo.On("GetUserByGoogleID", "google-id-123").Return(nil, nil)
	mockUserRepo.On("GetUserByEmail", "google@mail.com").Return(existingUser, nil)
	mockUserRepo.On("UpdateUser", mock.Anything).Return(nil)

	resp, err := uc.LoginWithGoogle(profile, "test-secret", 24)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	mockUserRepo.AssertExpectations(t)
}
