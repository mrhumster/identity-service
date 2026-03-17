package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/mrhumster/identity-service/pkg/dto"
)

type TokenServiceIFace interface {
	ValidateAccessToken(tokenString string) (*dto.AccessClaims, error)
}

func Authorize(client auth.PermissionClient, obj, act string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
		}
		userUUID := c.MustGet("user").(uuid.UUID)
		resourceID := c.Param("id")

		fullResource := obj
		if resourceID != "" {
			fullResource = fmt.Sprintf("%s/%s", obj, resourceID)
		}

		ok, err := client.CheckPermission(c.Request.Context(), userUUID.String(), fullResource, act)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse("⚠️ Authorize middleware error"))
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse("Access denied"))
		}
		c.Next()
	}
}

func AuthMiddleware(tokenService TokenServiceIFace) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c.Request)
		claims, err := tokenService.ValidateAccessToken(token)
		if err != nil {
			errMsg := fmt.Sprintf("invalid token claims: %s", err.Error())
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse(errMsg))
			c.Abort()
			return
		}
		userUUID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse("error parse user id in auth middleware"))
		}
		c.Set("user", userUUID)
		c.Set("claims", claims)
		c.Next()
	}
}

func OptionalAuthMiddleware(TokenService TokenServiceIFace) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", uuid.Nil)
		token := extractToken(c.Request)
		if token == "" {
			slog.Debug("The auth token has not been transferred")
			c.Next()
			return
		}
		claims, err := TokenService.ValidateAccessToken(token)
		if err != nil {
			slog.Info("Invalid token claims: ", "error", err.Error())
			c.Next()
			return
		}

		userUUID, err := uuid.Parse(claims.UserID)
		if err != nil {
			slog.Error("Error parse User ID", "error", err.Error())
			c.Next()
			return
		}
		c.Set("user", userUUID)
		c.Set("claims", claims)
		c.Next()
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
