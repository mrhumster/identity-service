package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRefreshStore(t *testing.T) (*RefreshTokenStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRefreshTokenStore(client, time.Hour), mr
}

func TestRefreshTokenStore_SingleUse(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()
	tok := "r." + uuid.NewString()
	userID := uuid.NewString()

	require.NoError(t, store.Allow(ctx, tok, userID))

	uid, err := store.Grant(ctx, tok)
	require.NoError(t, err)
	assert.Equal(t, userID, uid)

	// второй раз тот же токен → reuse
	_, err = store.Grant(ctx, tok)
	assert.ErrorIs(t, err, ErrRefreshReplay)
}

func TestRefreshTokenStore_UnknownTokenIsReplay(t *testing.T) {
	store, _ := newRefreshStore(t)
	_, err := store.Grant(context.Background(), "never-issued")
	assert.ErrorIs(t, err, ErrRefreshReplay)
}

func TestRefreshTokenStore_RevokedByLogout(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()
	tok := "r." + uuid.NewString()

	require.NoError(t, store.Allow(ctx, tok, uuid.NewString()))
	require.NoError(t, store.Deny(ctx, tok))

	_, err := store.Grant(ctx, tok)
	assert.ErrorIs(t, err, ErrRefreshRevoked)
}

func TestRefreshTokenStore_ExpiredEntryIsReplay(t *testing.T) {
	store, mr := newRefreshStore(t)
	ctx := context.Background()
	tok := "r." + uuid.NewString()

	require.NoError(t, store.Allow(ctx, tok, uuid.NewString()))
	mr.FastForward(2 * time.Hour)

	_, err := store.Grant(ctx, tok)
	assert.ErrorIs(t, err, ErrRefreshReplay)
}

func TestRefreshTokenStore_AllowAfterDeny(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()
	// logout denies tokenA; a fresh login always issues a NEW token (new hash)
	deniedTok := "r." + uuid.NewString()
	freshTok := "r." + uuid.NewString()

	require.NoError(t, store.Deny(ctx, deniedTok))
	require.NoError(t, store.Allow(ctx, freshTok, uuid.NewString()))

	uid, err := store.Grant(ctx, freshTok)
	require.NoError(t, err)
	assert.NotEmpty(t, uid)

	// старый токен остаётся невалидным
	_, err = store.Grant(ctx, deniedTok)
	assert.ErrorIs(t, err, ErrRefreshRevoked)
}
