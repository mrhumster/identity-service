package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/service"
)

type VerificationHandler struct {
	service *service.VerificationService
}

func NewVerificationHandler(service *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: service}
}

func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	var req request.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if _, err := h.service.VerifyToken(c, req.Token); err != nil {
		slog.Info("verify email failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func (h *VerificationHandler) ResendVerification(c *gin.Context) {
	userUUID := c.MustGet("user").(uuid.UUID)

	token, err := h.service.CreateToken(c, userUUID)
	if err != nil {
		slog.Error("resend verification token failed", "user_id", userUUID.String(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// TODO: заменить на доставку через notification-service (SMTP), когда он появится.
	slog.Info("verification token (resend)", "user_id", userUUID.String(), "token", token)
	c.JSON(http.StatusOK, gin.H{"message": "verification sent"})
}