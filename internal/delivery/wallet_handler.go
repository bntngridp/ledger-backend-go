package delivery

import (
	"net/http"
	"strings"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/bntngridp/ledger-backend/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WalletHandler struct {
	walletUC usecase.WalletUsecase
}

// NewWalletHandler godoc
// @Description Constructs WalletHandler with wallet usecase.
func NewWalletHandler(walletUC usecase.WalletUsecase) *WalletHandler {
	return &WalletHandler{walletUC: walletUC}
}

// TopUp godoc
// @Summary      Top-up wallet balance
// @Description  Adds the specified amount to the authenticated user's wallet balance. Records a transaction with type 'topup'.
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.TopUpRequest true "Top-up payload"
// @Success      200 {object} domain.SuccessResponse{data=domain.TopUpResponse} "Top-up successful"
// @Failure      400 {object} domain.ErrorResponse "Invalid request, amount must be greater than 0, or wallet not found"
// @Failure      401 {object} domain.ErrorResponse "Unauthorized"
// @Failure      500 {object} domain.ErrorResponse "Internal server error"
// @Router       /topup [post]
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

	// For fiat top-up endpoint, asset symbol is IDR
	resp, err := h.walletUC.TopUp(userID, req.Amount, "IDR", req.Notes)
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

// GetTransactionHistory godoc
// @Summary      Get transaction history
// @Description  Returns the authenticated user's transaction history (incoming and outgoing), ordered by most recent first.
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.SuccessResponse{data=[]domain.TransactionHistoryItem} "Transaction history retrieved"
// @Failure      401 {object} domain.ErrorResponse "Unauthorized"
// @Failure      500 {object} domain.ErrorResponse "Internal server error"
// @Router       /transactions [get]
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

// GetDashboard godoc
// @Summary      Get wallet dashboard
// @Description  Returns the authenticated user's wallet balances (IDR, USDT, USDC) and estimated total IDR value.
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} domain.SuccessResponse{data=domain.DashboardResponse} "Dashboard data retrieved"
// @Failure      401 {object} domain.ErrorResponse "Unauthorized"
// @Failure      500 {object} domain.ErrorResponse "Internal server error"
// @Router       /wallet/dashboard [get]
func (h *WalletHandler) GetDashboard(c *gin.Context) {
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

	dashboard, err := h.walletUC.GetDashboard(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Status:  http.StatusInternalServerError,
			Message: "failed to retrieve dashboard: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, domain.SuccessResponse{
		Status:  http.StatusOK,
		Message: "dashboard retrieved successfully",
		Data:    dashboard,
	})
}
