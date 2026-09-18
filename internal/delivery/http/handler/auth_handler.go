package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/internal/events"
	internalmetrics "github.com/mrhumster/identity-service/internal/metrics"
	"github.com/mrhumster/identity-service/internal/service"
)

type AuthHandler struct {
	UserService  *service.UserService
	TokenService *service.TokenService
	JwtSecret    string
	Domain       string
	Events       *events.Recorder
	RefreshStore *service.RefreshTokenStore
}

func NewAuthHandler(userService *service.UserService, tokenService *service.TokenService, jwtSecret, domain string) *AuthHandler {
	return &AuthHandler{
		UserService:  userService,
		TokenService: tokenService,
		JwtSecret:    jwtSecret,
		Domain:       domain,
	}
}

// WithEvents attaches the activity-event recorder (best-effort, may be nil).
func (a *AuthHandler) WithEvents(r *events.Recorder) *AuthHandler {
	a.Events = r
	return a
}

// WithRefreshStore enables single-use refresh tokens (rotation + reuse
// detection). When nil, refresh behaves as before (stateless).
func (a *AuthHandler) WithRefreshStore(r *service.RefreshTokenStore) *AuthHandler {
	a.RefreshStore = r
	return a
}

func (a *AuthHandler) Login(c *gin.Context) {
	var (
		req request.LoginRequest
		u   *models.User
		err error
	)
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if u, err = a.UserService.ValidateUser(c, req.Email, req.Password); err != nil {
		slog.Debug("login failed", "email", req.Email, "error", err)
		internalmetrics.LoginAttempts.WithLabelValues("invalid").Inc()
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("invalid email or password"))
		return
	}

	internalmetrics.LoginAttempts.WithLabelValues("success").Inc()

	if a.Events != nil {
		if err := a.Events.RecordActivity(c, u.ID, "user.login", nil, map[string]any{"email": req.Email}); err != nil {
			slog.Warn("record user.login event failed", "user_id", u.ID.String(), "error", err)
		}
	}

	tokenPair, err := a.TokenService.GenerateToken(u)
	if err != nil {
		slog.Error("generate token failed", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.ErrorResponse("internal server error"))
		return
	}

	if a.RefreshStore != nil {
		if err := a.RefreshStore.Allow(c, tokenPair.RefreshToken, u.ID.String()); err != nil {
			// Best-effort: without the allow-entry the first refresh looks
			// like a replay, so log it loudly.
			slog.Error("allow refresh token on login", "user_id", u.ID.String(), "error", err)
		}
	}

	c.SetSameSite(http.SameSiteLaxMode)

	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(a.TokenService.GetRefreshExpiry().Seconds()),
		"/",
		a.Domain,
		true,
		true,
	)

	var user response.UserResponse
	user.FillInTheModel(u)
	c.JSON(http.StatusOK, response.LoginResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   tokenPair.ExpiresIn,
		TokenType:   tokenPair.TokenType,
	})
}

func (a *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("refresh token required"))
		return
	}

	claims, err := a.TokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("invalid refresh token"))
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("invalid user id in claim"))
		return
	}
	u, err := a.UserService.ReadUser(c, userID)
	if err != nil || u.TokenVersion != claims.TokenVersion {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("token revoke"))
		return
	}

	revokeSession := func(reason string) {
		if bumpErr := a.UserService.UpdateTokenVersion(c, &userID, generateNewTokenVersion()); bumpErr != nil {
			slog.Error("revoke session on refresh anomaly", "user_id", userID.String(), "reason", reason, "error", bumpErr)
		}
	}

	if a.RefreshStore != nil {
		grantedUserID, gErr := a.RefreshStore.Grant(c, refreshToken)
		if gErr != nil {
			if errors.Is(gErr, service.ErrRefreshRevoked) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("refresh token revoked"))
				return
			}
			// Reuse of a consumed token (or lost Redis state): fail closed and
			// invalidate every session of the user.
			slog.Warn("refresh reuse detected", "user_id", userID.String(), "error", gErr)
			revokeSession("reuse")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("refresh token reuse detected"))
			return
		}
		if grantedUserID != u.ID.String() {
			slog.Warn("refresh token user mismatch", "user_id", userID.String())
			revokeSession("mismatch")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("refresh token reuse detected"))
			return
		}
	}

	tokenPair, err := a.TokenService.GenerateToken(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("failed to generate token"))
		return
	}
	if a.RefreshStore != nil {
		if err := a.RefreshStore.Allow(c, tokenPair.RefreshToken, u.ID.String()); err != nil {
			slog.Error("allow refresh token on rotation", "user_id", u.ID.String(), "error", err)
		}
	}
	internalmetrics.TokensRefreshed.Inc()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"refresh_token",
		tokenPair.RefreshToken,
		int(a.TokenService.GetRefreshExpiry().Seconds()),
		"/",
		a.Domain,
		true,
		true,
	)

	var user response.UserResponse
	user.FillInTheModel(u)
	c.JSON(http.StatusOK, response.LoginResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   tokenPair.ExpiresIn,
		TokenType:   tokenPair.TokenType,
	})
}

func (a *AuthHandler) Logout(c *gin.Context) {
	if a.RefreshStore != nil {
		if refreshToken, err := c.Cookie("refresh_token"); err == nil && refreshToken != "" {
			if err := a.RefreshStore.Deny(c, refreshToken); err != nil {
				slog.Warn("deny refresh token on logout", "error", err)
			}
		}
	}
	c.SetCookie("refresh_token", "", -1, "/", a.Domain, true, true)
	c.JSON(http.StatusOK, response.SuccessResponse("Logged out successfully"))
}

func (a *AuthHandler) LogoutAll(c *gin.Context) {
	userUUID := c.MustGet("user").(uuid.UUID)
	if err := a.UserService.UpdateTokenVersion(c, &userUUID, generateNewTokenVersion()); err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorResponse("failed to logout"))
		return
	}

	c.SetCookie("refresh_token", "", -1, "/", a.Domain, true, true)

	c.JSON(http.StatusOK, response.ErrorResponse("logged out from all devices"))
}

func generateNewTokenVersion() string {
	return "v" + time.Now().Format("20060102150405")
}
