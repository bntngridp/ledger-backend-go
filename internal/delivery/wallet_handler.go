package delivery

import (
	"net/http"
	"strings"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/bntngridp/ledger-backend-go/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WalletHandler struct {
	walletUC usecase.WalletUsecase
}

func NewWalletHandler(walletUC usecase.WalletUsecase) *WalletHandler {
	return &WalletHandler{walletUC: walletUC}
}

func (h *WalletHandler) TopUp(c *gin.Context) {
	var req domain.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "invalid user id in token",
		})
		return
	}

	resp, err := h.walletUC.TopUp(userID, req.Amount, req.Notes)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "amount must be greater than 0") || strings.Contains(msg, "wallet not found") {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{
				Status:  http.StatusBadRequest,
				Message: msg,
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
		Message: "top-up successful",
		Data:    resp,
	})
}

func (h *WalletHandler) GetTransactionHistory(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "invalid user id in token",
		})
		return
	}

	history, err := h.walletUC.GetTransactionHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Status:  http.StatusInternalServerError,
			Message: "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, domain.SuccessResponse{
		Status:  http.StatusOK,
		Message: "transaction history retrieved",
		Data:    history,
	})
}
