package dto

import "github.com/golang-jwt/jwt/v5"

func ErrorResponse(message string) map[string]any {
	return map[string]any{
		"error": message,
	}
}

type AccessClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID       string `json:"user_id"`
	TokenVersion string `json:"token_version"`
	jwt.RegisteredClaims
}
