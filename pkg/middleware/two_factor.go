package middleware

import (
	"net/http"

	"github.com/bntngridp/ledger-backend/internal/domain"
	"github.com/bntngridp/ledger-backend/internal/usecase"
	"github.com/bntngridp/ledger-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Require2FAIfEnabled enforces a TOTP verification check if the authenticated user has enabled 2FA.
func Require2FAIfEnabled(authUC usecase.AuthUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		code := c.GetHeader("X-2FA-Code")

		if err := authUC.Verify2FACode(userID, code); err != nil {
			response.HandleError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}
