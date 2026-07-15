package delivery

import (
	"net/http"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/bntngridp/ledger-backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUC      usecase.AuthUsecase
	jwtSecret   string
	expiryHours int
}

// NewAuthHandler godoc
// @Description Constructs AuthHandler with usecase and JWT config.
func NewAuthHandler(authUC usecase.AuthUsecase, jwtSecret string, expiryHours int) *AuthHandler {
	return &AuthHandler{
		authUC:      authUC,
		jwtSecret:   jwtSecret,
		expiryHours: expiryHours,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account and an associated wallet with balance 0 in a single atomic transaction.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.RegisterRequest true "Registration payload"
// @Success      201 {object} domain.SuccessResponse{data=domain.RegisterResponse} "User registered successfully"
// @Failure      400 {object} domain.ErrorResponse "Invalid request body"
// @Failure      409 {object} domain.ErrorResponse "Email or username already taken"
// @Failure      500 {object} domain.ErrorResponse "Internal server error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.authUC.Register(req.Username, req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "email already registered", "username already taken":
			c.JSON(http.StatusConflict, domain.ErrorResponse{
				Status:  http.StatusConflict,
				Message: err.Error(),
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
				Status:  http.StatusInternalServerError,
				Message: "internal server error",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, domain.SuccessResponse{
		Status:  http.StatusCreated,
		Message: "user registered successfully",
		Data:    resp,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticates a user and returns a JWT token valid for 24 hours.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.LoginRequest true "Login payload"
// @Success      200 {object} domain.SuccessResponse{data=domain.LoginResponse} "Login successful"
// @Failure      400 {object} domain.ErrorResponse "Invalid request body"
// @Failure      401 {object} domain.ErrorResponse "Invalid email or password"
// @Failure      500 {object} domain.ErrorResponse "Internal server error"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.authUC.Login(req.Email, req.Password, h.jwtSecret, h.expiryHours)
	if err != nil {
		if err.Error() == "invalid email or password" {
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Status:  http.StatusUnauthorized,
				Message: "invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Status:  http.StatusInternalServerError,
			Message: "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, domain.SuccessResponse{
		Status:  http.StatusOK,
		Message: "login successful",
		Data:    resp,
	})
}
