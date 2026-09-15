package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestTokenService_GenerateAndValidateToken(t *testing.T) {
	cfg, _ := config.TestConfig()
	service, _ := NewTokenService(&cfg.JWT)

	email := "testuser@test.local"
	role := "member"
	tokenVersion := "1"
	user := &models.User{
		Email:        email,
		Role:         role,
		TokenVersion: tokenVersion,
	}

	token, err := service.GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := service.ValidateAccessToken(token.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s", user.ID), claims.UserID)
	assert.Equal(t, "auth-service", claims.Issuer)
}

func TestTokenService_EmailVerifiedClaim(t *testing.T) {
	cfg, _ := config.TestConfig()
	service, _ := NewTokenService(&cfg.JWT)

	verifiedUser := &models.User{Role: "member", Email: "me@xomrkob.ru", EmailVerified: true}
	token, err := service.GenerateToken(verifiedUser)
	assert.NoError(t, err)
	claims, err := service.ValidateAccessToken(token.AccessToken)
	assert.NoError(t, err)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "me@xomrkob.ru", claims.Email)

	unverifiedUser := &models.User{Role: "member", EmailVerified: false}
	token, err = service.GenerateToken(unverifiedUser)
	assert.NoError(t, err)
	claims, err = service.ValidateAccessToken(token.AccessToken)
	assert.NoError(t, err)
	assert.False(t, claims.EmailVerified)
	assert.Empty(t, claims.Email)
}

func TestTokenService_ValidateToken_Invalid(t *testing.T) {
	cfg, _ := config.TestConfig()
	service, _ := NewTokenService(&cfg.JWT)

	_, err := service.ValidateAccessToken("invalid-token")
	assert.Error(t, err)

	claims := &models.AccessClaims{
		UserID: "123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Issuer:    "test-issuer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	expiredToken, _ := token.SignedString([]byte("test-secret"))

	_, err = service.ValidateAccessToken(expiredToken)
	assert.Error(t, err)
}
