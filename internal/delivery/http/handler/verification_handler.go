package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/events"
	"github.com/mrhumster/identity-service/internal/service"
)

type VerificationHandler struct {
	service *service.VerificationService
	Events  *events.Recorder
}

func NewVerificationHandler(service *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: service}
}

// WithEvents attaches the activity-event recorder (best-effort, may be nil).
func (h *VerificationHandler) WithEvents(r *events.Recorder) *VerificationHandler {
	h.Events = r
	return h
}

func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	var req request.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := h.service.VerifyToken(c, req.Token)
	if err != nil {
		slog.Info("verify email failed", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification token"})
		return
	}

	if h.Events != nil && userID != nil {
		if err := h.Events.RecordActivity(c, *userID, "user.email.verified", nil, map[string]any{}); err != nil {
			slog.Warn("record user.email.verified event failed", "user_id", userID.String(), "error", err)
		}
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

	// Доставка письма — через mailer-service (asynq), токен логируется как fallback.
	slog.Info("verification token (resend)", "user_id", userUUID.String(), "token", token)
	c.JSON(http.StatusOK, gin.H{"message": "verification sent"})
}