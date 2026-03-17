package response

import (
	"fmt"
)

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (l *LoginResponse) GetTokenAsBearerHeader() string {
	return fmt.Sprintf("Bearer %s", l.AccessToken)
}
