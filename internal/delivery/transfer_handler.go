package delivery

import (
	"net/http"
	"strings"

	"github.com/bntngridp/ledger-backend-go/internal/domain"
	"github.com/bntngridp/ledger-backend-go/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransferHandler struct {
	transferUC usecase.TransferUsecase
}

func NewTransferHandler(transferUC usecase.TransferUsecase) *TransferHandler {
	return &TransferHandler{transferUC: transferUC}
}

func (h *TransferHandler) Transfer(c *gin.Context) {
	var req domain.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	senderStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized",
		})
		return
	}

	senderID, err := uuid.Parse(senderStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: "invalid user id in token",
		})
		return
	}

	destID, err := uuid.Parse(req.DestinationUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid destination_user_id",
		})
		return
	}

	if err := h.transferUC.Transfer(senderID, destID, req.Amount, req.Notes); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "insufficient balance") || strings.Contains(msg, "amount must be greater than 0") || strings.Contains(msg, "cannot transfer to yourself") {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{
				Status:  http.StatusBadRequest,
				Message: msg,
			})
			return
		}
		if strings.Contains(msg, "recipient wallet not found") || strings.Contains(msg, "sender wallet not found") {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{
				Status:  http.StatusNotFound,
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
		Message: "transfer successful",
	})
}
