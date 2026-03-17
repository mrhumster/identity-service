package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/internal/service"
)

type AuthHandler struct {
	UserService  *service.UserService
	TokenService *service.TokenService
	JwtSecret    string
	Domain       string
}

func NewAuthHandler(userService *service.UserService, tokenService *service.TokenService, jwtSecret, domain string) *AuthHandler {
	return &AuthHandler{
		UserService:  userService,
		TokenService: tokenService,
		JwtSecret:    jwtSecret,
		Domain:       domain,
	}
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
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse(err.Error()))
		return
	}

	tokenPair, err := a.TokenService.GenerateToken(u)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			response.ErrorResponse(
				fmt.Sprintf("generate token: %v", err.Error()),
			),
		)
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
	}
	u, err := a.UserService.ReadUser(c, userID)
	if err != nil || u.TokenVersion != claims.TokenVersion {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("token revoke"))
		return
	}
	tokenPair, err := a.TokenService.GenerateToken(u)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse("failed to generate token"))
		return
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

func (a *AuthHandler) Logout(c *gin.Context) {
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
