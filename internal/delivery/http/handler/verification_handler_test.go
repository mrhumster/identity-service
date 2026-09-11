package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/internal/repository"
	"github.com/mrhumster/identity-service/tests/testutils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setVerificationToken кладёт токен напрямую в Redis, как это сделал бы
// VerificationService.CreateToken (ключи с verify: индекс для пользователя).
func setVerificationToken(t *testing.T, cfg *config.Config, token string, userID uuid.UUID) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "verify:"+token, userID.String(), cfg.Server.VerifyTokenTTL).Err())
	require.NoError(t, client.Set(ctx, "verify_user:"+userID.String(), token, cfg.Server.VerifyTokenTTL).Err())
}

func TestVerifyEmail_Success(t *testing.T) {
	router, db := setupTest()
	defer testutils.CleanTestDatabase()
	cfg, err := config.TestConfig()
	require.NoError(t, err)

	email := "verify@test.local"
	resp, err := createUserRequest(router, "password", email)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.Code)

	user := readUserByEmail(db, email)
	require.NotNil(t, user)
	require.False(t, user.EmailVerified)

	token := uuid.New().String()
	setVerificationToken(t, cfg, token, user.ID)

	body := request.VerifyRequest{Token: token}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/verify", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	verified := readUserByEmail(db, email)
	require.NotNil(t, verified)
	assert.True(t, verified.EmailVerified)

	// токен одноразовый
	req, _ = http.NewRequest("POST", "/api/verify", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()

	body := request.VerifyRequest{Token: "bogus-token"}
	bodyJSON, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/verify", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	req, _ = http.NewRequest("POST", "/api/verify", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestResendVerification_RequiresAuth(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()

	req, _ := http.NewRequest("POST", "/api/resend", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestResendVerification_Ok(t *testing.T) {
	router, _ := setupTest()
	defer testutils.CleanTestDatabase()

	_, err := createUserRequest(router, "password", "resend@test.local")
	require.NoError(t, err)
	token := LoginAndGetToken(router, "resend@test.local", "password")
	require.NotEmpty(t, token)

	req, _ := http.NewRequest("POST", "/api/resend", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func readUserByEmail(db *gorm.DB, email string) *models.User {
	repo := repository.NewGormUserRepository(db)
	user, err := repo.ReadUserByEmail(context.Background(), email)
	if err != nil {
		return nil
	}
	return user
}