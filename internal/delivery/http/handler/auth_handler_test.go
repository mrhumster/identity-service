package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/tests/testutils"
	"github.com/stretchr/testify/assert"
)

func AuthByLogin(router *gin.Engine, email, password string) (*response.LoginResponse, error) {
	var (
		req *http.Request
		err error
	)
	loginRequest := request.LoginRequest{Email: email, Password: password}
	loginRequestJson, _ := json.Marshal(loginRequest)
	if req, err = http.NewRequest("POST", "/api/login", bytes.NewBuffer(loginRequestJson)); err != nil {
		return nil, err
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var body *response.LoginResponse
	json.Unmarshal(resp.Body.Bytes(), &body)
	return body, nil

}

func TestAuthHandler_Login_Success(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	login := fmt.Sprintf("testuser-%s", uuid.New().String())
	password := uuid.New().String()
	email := fmt.Sprintf("%s@test.local", login)
	resp1, _ := createUserRequest(router, password, email)
	assert.Equal(t, http.StatusCreated, resp1.Code)

	loginReq := request.LoginRequest{
		Email:    email,
		Password: password,
	}
	response, err := AuthByLogin(router, loginReq.Email, loginReq.Password)
	if err != nil {
		t.Errorf("⚠️ Login error %v", err)
	}
	assert.NotEmpty(t, response.AccessToken)
}
