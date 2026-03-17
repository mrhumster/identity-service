package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/response"
	"github.com/mrhumster/identity-service/internal/delivery/http/routes"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/mrhumster/identity-service/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTest() (*gin.Engine, *gorm.DB) {
	db := testutils.GetTestDB()
	cfg, err := config.TestConfig()
	if err != nil {
		panic("⚠️ SetupTest config error")
	}
	permGRPCClient, err := auth.NewPermissionClient(cfg.Server.AuthServiceAddr)
	if err != nil {
		panic("⚠️ SetupTest gRPC client error")
	}
	router := routes.SetupRoutes(db, "test", permGRPCClient)
	return router, db
}

func createUserRequest(router *gin.Engine, password, email string) (*httptest.ResponseRecorder, error) {
	user := request.UserRequest{
		Password: password,
		Email:    email,
	}

	userJSON, _ := json.Marshal(user)
	req, err := http.NewRequest("POST", "/api/users", bytes.NewBuffer(userJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var body map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func LoginAndGetToken(router *gin.Engine, email, password string) string {
	body := request.LoginRequest{Email: email, Password: password}
	bodyJson, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(bodyJson))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var response response.LoginResponse
	json.Unmarshal(resp.Body.Bytes(), &response)
	return response.AccessToken
}

func TestUserHandler_Success(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()

	user := request.UserRequest{
		Password: "testuser",
		Email:    "testuser9@test.local",
	}

	userJSON, _ := json.Marshal(user)

	req, _ := http.NewRequest("POST", "/api/users", bytes.NewBuffer(userJSON))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusCreated, resp.Code)

	var response map[string]any

	json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response, "id")
	assert.NotZero(t, response["id"])
}

func TestUserHandler_DiplucateLogin(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp1, err := createUserRequest(router, "password1", "testuser@test.local")
	if err != nil {
		t.Errorf("⚠️ %v", err)
	}
	assert.Equal(t, http.StatusCreated, resp1.Code)

	resp2, err := createUserRequest(router, "password2", "testuser@test.local")
	log.Printf("🐞 resp = %v, err = %v", resp2, err)
	assert.Equal(t, http.StatusConflict, resp2.Code)

	var response map[string]any
	json.Unmarshal(resp2.Body.Bytes(), &response)

}

func TestUserHandler_EmptyPassword(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp1, _ := createUserRequest(router, "", "testuser@test.local")

	assert.Equal(t, http.StatusBadRequest, resp1.Code)

	var response map[string]any
	json.Unmarshal(resp1.Body.Bytes(), &response)
	assert.Contains(t, response, "errors")
	assert.Equal(t, response["errors"], map[string]interface{}{"Password": "This field is required"})
}

func TestUserHandler_InvalidDate(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	userJSON := `{"login": 123456, "password": 123456}`
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewBuffer([]byte(userJSON)))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	var response map[string]any
	json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	log.Printf("Invalid Data error: %s", response["error"])
}

func TestUserHandler_ReadUser_InvalidID(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp, _ := createUserRequest(router, "password", "testuser@test.local")
	loginResponse, err := AuthByLogin(router, "testuser@test.local", "password")
	if err != nil {
		t.Errorf("⚠️  Login error: %v", err)
	}
	var repsonse1 map[string]any
	json.Unmarshal(resp.Body.Bytes(), &repsonse1)
	userID := repsonse1["id"]
	log.Printf("TestUserHandler_ReadUser_InvalidID: %s", userID)
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/users/%s", userID), nil)
	if err != nil {
		t.Errorf("Request error: %v", err)
	}
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var response map[string]any
	json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response, "id")
	assert.Equal(t, response["id"], userID)
	assert.Contains(t, response, "email")
	assert.NotEmpty(t, response["email"])

	req, _ = http.NewRequest("GET", "/api/users/-1", nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestUserHandler_DeleteUser(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp, _ := createUserRequest(router, "password", "testuser@test.local")
	loginReponse, _ := AuthByLogin(router, "testuser@test.local", "password")
	var response map[string]any
	json.Unmarshal(resp.Body.Bytes(), &response)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/users/%s", response["id"]), nil)
	req.Header.Set("Authorization", loginReponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var body map[string]any
	json.Unmarshal(resp.Body.Bytes(), &body)
	assert.Equal(t, http.StatusNoContent, resp.Code)

	req, _ = http.NewRequest("DELETE", "/api/users/-1", nil)
	req.Header.Add("Authorization", loginReponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestUserHandler_ReadUsers(t *testing.T) {
	page := float64(1)
	limit := float64(5)
	total := 10
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	for i := range total {
		createUserRequest(router, "password", fmt.Sprintf("testuser%d@test.local", i))
	}

	req, _ := http.NewRequest("GET", "/api/users", nil)
	loginResponse, _ := AuthByLogin(router, "testuser1@test.local", "password")
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var respMap1 map[string]any
	json.Unmarshal(resp.Body.Bytes(), &respMap1)
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/users?page=%.0f&limit=%.0f", page, limit), nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var respMap map[string]any
	json.Unmarshal(resp.Body.Bytes(), &respMap)
	assert.Contains(t, respMap, "total")
	assert.Contains(t, respMap, "page")
	assert.Contains(t, respMap, "limit")
	assert.Contains(t, respMap, "users")
	assert.Equal(t, float64(total), respMap["total"])
	assert.Equal(t, page, respMap["page"])
	users, ok := respMap["users"].([]interface{})
	assert.True(t, ok, "users should be an array")
	assert.Equal(t, int(limit), len(users))
	for _, u := range users {

		userMap, ok := u.(map[string]interface{})
		require.True(t, ok, "each user should be an object")

		require.NotEmpty(t, userMap["email"])
	}
}

func TestUserHandler_ReadUsers_QueryValidate(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	createUserRequest(router, "password", "testuser@test.local")
	loginResponse, _ := AuthByLogin(router, "testuser@test.local", "password")
	req, _ := http.NewRequest("GET", "/api/users?page=-1&limit=s", nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	req, _ = http.NewRequest("GET", "/api/users?page=1&limit=-10", nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	req, _ = http.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestUserHandler_UserResponseIncludeEmail(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp, err := createUserRequest(router, "password", "testuser1@test.local")
	if err != nil {
		t.Errorf("Create user request error: %v", err)
	}
	var createUserBody map[string]any

	json.Unmarshal(resp.Body.Bytes(), &createUserBody)
	loginResponse, err := AuthByLogin(router, "testuser1@test.local", "password")
	if err != nil {
		t.Errorf("Create user request error: %v", err)
	}

	var respMap map[string]any
	json.Unmarshal(resp.Body.Bytes(), &respMap)
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/users/%s", respMap["id"]), nil)
	if err != nil {
		t.Errorf("Create user request error: %v", err)
	}

	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code == http.StatusForbidden {
		t.Errorf("🐞 Forbidden: token: %s\n\t user id: %s", loginResponse.GetTokenAsBearerHeader(), createUserBody["userId"])
	}
	json.Unmarshal(resp.Body.Bytes(), &respMap)
	log.Printf("🐞 %v", respMap)
	assert.Contains(t, respMap, "email")
}

func TestUserHandler_EmailIsUniq(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	createUserRequest(router, "password", "testuser1@test.local")
	resp, _ := createUserRequest(router, "password", "testuser1@test.local")
	assert.Equal(t, http.StatusConflict, resp.Code)
	var respMap map[string]any
	json.Unmarshal(resp.Body.Bytes(), &respMap)
	assert.Contains(t, respMap, "error")
	assert.Equal(t, respMap["error"], "user already exists")

}

func TestUserHandler_Response_IncludeAllFields(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()
	resp, _ := createUserRequest(router, "password", "testuser1@test.local")
	loginResponse, _ := AuthByLogin(router, "testuser1@test.local", "password")
	var body map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &body)
	userId := body["id"]
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/users/%s", userId), nil)
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	json.Unmarshal(resp.Body.Bytes(), &body)
}

func TestUserHandler_UpdateUser(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()

	resp, err := createUserRequest(router, "password", "testuser1@test.local")
	if err != nil {
		t.Errorf("UpdateUser error: %v", err)
	}
	loginResponse, err := AuthByLogin(router, "testuser1@test.local", "password")
	if err != nil {
		t.Errorf("UpdateUser error: %v", err)
	}
	var body, updatedBody map[string]any
	json.Unmarshal(resp.Body.Bytes(), &body)

	var userUpdate request.UpdateUserRequest

	userUpdate.Email = "testuser2@test.local"

	userJson, err := json.Marshal(userUpdate)
	if err != nil {
		t.Errorf("UpdateUser error: %v", err)
	}
	req, err := http.NewRequest("PATCH", fmt.Sprintf("/api/users/%s", body["id"]), bytes.NewBuffer(userJson))
	req.Header.Set("Authorization", loginResponse.GetTokenAsBearerHeader())
	if err != nil {
		t.Errorf("UpdateUser error: %v", err)
	}
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	json.Unmarshal(resp.Body.Bytes(), &updatedBody)
	log.Printf("%v", updatedBody)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, updatedBody, "email")
	assert.Equal(t, updatedBody["email"], "testuser2@test.local")
}
