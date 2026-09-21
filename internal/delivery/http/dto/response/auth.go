package response

import (
	"fmt"
)

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// VerifyResponse возвращается из POST /auth/verify при успешной верификации
// email. Помимо verified он несёт свежую токен-пару (access_token + refresh
// cookie), чтобы клиент не проходил повторный логин ради обновления claim
// email_verified в JWT.
type VerifyResponse struct {
	Verified    bool   `json:"verified"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (l *LoginResponse) GetTokenAsBearerHeader() string {
	return fmt.Sprintf("Bearer %s", l.AccessToken)
}
