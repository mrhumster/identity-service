package models

import (
	"testing"

	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/stretchr/testify/assert"
)

func TestUser_SetPassword(t *testing.T) {
	user := User{}
	password := "mySecretPassword123"
	err := user.SetPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, user.PasswordHash, password)
}

func TestUser_CheckPassword(t *testing.T) {
	user := User{}
	password := "mySecretPassword123"
	err := user.SetPassword(password)
	assert.NoError(t, err)
	isValid := user.CheckPassword(password)
	assert.True(t, isValid)
	isValid = user.CheckPassword("myWrongPassword")
	assert.False(t, isValid)
}

func TestUser_FillTest(t *testing.T) {
	user := User{}
	req := request.UserRequest{
		Password: "password",
	}
	user.FillInTheRequest(req)
	assert.NotEqual(t, user.PasswordHash, req.Password)
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "me@xomrkob.ru", NormalizeEmail("  ME@Xomrkob.Ru  "))
	assert.Equal(t, "user@example.com", NormalizeEmail("user@example.com"))
	assert.Equal(t, "", NormalizeEmail("   "))
}

func TestUser_FillInTheRequest_NormalizesEmail(t *testing.T) {
	user := User{}
	user.FillInTheRequest(request.UserRequest{Email: "  Foo@Bar.com ", Password: "password"})
	assert.Equal(t, "foo@bar.com", user.Email)
}

func TestUser_FillInTheUpdateRequest_NormalizesEmail(t *testing.T) {
	user := User{}
	user.FillInTheUpdateRequest(request.UpdateUserRequest{Email: " Foo@Bar.com "})
	assert.Equal(t, "foo@bar.com", user.Email)
}
