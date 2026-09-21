package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/events"
	"github.com/mrhumster/identity-service/internal/service"
)

type VerificationHandler struct {
	service *service.VerificationService

	// Session dependencies (optional): when wired, a successful verification
	// issues a fresh token pair (access_token + refresh cookie) so the client
	// does not need to sign in again to pick up the email_verified claim.
	UserService  *service.UserService
	TokenService *service.TokenService
	Domain       string
	RefreshStore *service.RefreshTokenStore

	Events *events.Recorder
}

func NewVerificationHandler(service *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: service}
}

// WithEvents attaches the activity-event recorder (best-effort, may be nil).
func (h *VerificationHandler) WithEvents(r *events.Recorder) *VerificationHandler {
	h.Events = r
	return h
}

// WithSession enables issuing a token pair on successful verification.
func (h *VerificationHandler) WithSession(userService *service.UserService, tokenService *service.TokenService, domain string) *VerificationHandler {
	h.UserService = userService
	h.TokenService = tokenService
	h.Domain = domain
	return h
}

// WithRefreshStore enables single-use refresh tokens (rotation + reuse
// detection), matching the login/refresh flow.
func (h *VerificationHandler) WithRefreshStore(r *service.RefreshTokenStore) *VerificationHandler {
	h.RefreshStore = r
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

	if userID != nil && h.UserService != nil && h.TokenService != nil {
		h.issueSession(c, *userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": true})
}

// issueSession выдаёт свежую токен-пару для верифицированного пользователя и
// ставит refresh_token cookie — тот же флоу, что Login/Refresh.
func (h *VerificationHandler) issueSession(c *gin.Context, userID uuid.UUID) {
	u, err := h.UserService.ReadUser(c, userID)
	if err != nil {
		slog.Error("verify: read user failed", "user_id", userID.String(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	tokenPair, err := h.TokenService.GenerateToken(u)
	if err != nil {
		slog.Error("verify: generate token failed", "user_id", userID.String(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if h.RefreshStore != nil {
		if err := h.RefreshStore.Allow(c, tokenPair.RefreshToken, u.ID.String()); err != nil {
			slog.Error("allow refresh token on verify", "user_id", u.ID.String(), "error", err)
		}
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(h.TokenService.GetRefreshExpiry().Seconds()),
		"/",
		h.Domain,
		true,
		true,
	)

	c.JSON(http.StatusOK, response.VerifyResponse{
		Verified:    true,
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   tokenPair.ExpiresIn,
		TokenType:   tokenPair.TokenType,
	})
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