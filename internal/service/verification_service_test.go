package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	repomock "github.com/mrhumster/identity-service/internal/repository/mock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerificationService_CreateAndVerifyTokenFull(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := repomock.NewMockUserRepository(ctrl)
	svc := NewVerificationService(client, repo, time.Hour*2)

	ctx := context.Background()
	userID := uuid.New()

	repo.EXPECT().UpdateEmailVerified(gomock.Any(), userID, true).Return(nil)

	token, err := svc.CreateToken(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	verifiedID, err := svc.VerifyToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, userID, *verifiedID)

	// токен одноразовый
	_, err = svc.VerifyToken(ctx, token)
	assert.ErrorIs(t, err, ErrInvalidVerificationToken)
}

func TestVerificationService_VerifyToken_Invalid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := repomock.NewMockUserRepository(ctrl)
	svc := NewVerificationService(client, repo, time.Hour*2)

	_, err := svc.VerifyToken(context.Background(), "")
	assert.ErrorIs(t, err, ErrInvalidVerificationToken)

	_, err = svc.VerifyToken(context.Background(), "nosuchtoken")
	assert.ErrorIs(t, err, ErrInvalidVerificationToken)
}

func TestVerificationService_CreateToken_RevokesPrevious(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	repo := repomock.NewMockUserRepository(ctrl)
	svc := NewVerificationService(client, repo, time.Hour*2)

	ctx := context.Background()
	userID := uuid.New()

	first, err := svc.CreateToken(ctx, userID)
	require.NoError(t, err)

	// регенерация отзывает первый токен
	second, err := svc.CreateToken(ctx, userID)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)

	_, err = svc.VerifyToken(ctx, first)
	assert.ErrorIs(t, err, ErrInvalidVerificationToken)

	repo.EXPECT().UpdateEmailVerified(gomock.Any(), userID, true).Return(nil)
	_, err = svc.VerifyToken(ctx, second)
	assert.NoError(t, err)
}