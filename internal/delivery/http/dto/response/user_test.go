package response

import (
	"testing"

	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestUserResponse_Fill(t *testing.T) {
	user := models.User{
		PasswordHash: "***123***",
		Email:        "billy@test.local",
	}
	res := UserResponse{}
	res.FillInTheModel(&user)
	assert.Equal(t, user.Email, res.Email)
}
