package delivery

import (
	"net/http"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/bntngridp/ledger-backend-go/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUC      usecase.AuthUsecase
	jwtSecret   string
	expiryHours int
}

func NewAuthHandler(authUC usecase.AuthUsecase, jwtSecret string, expiryHours int) *AuthHandler {
	return &AuthHandler{
		authUC:      authUC,
		jwtSecret:   jwtSecret,
		expiryHours: expiryHours,
	}
}

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
