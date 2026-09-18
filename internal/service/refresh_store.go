package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrRefreshReplay marks a refresh token that was already consumed,
	// unknown, or whose allow-entry is gone (including lost Redis state).
	// The caller should revoke the whole session.
	ErrRefreshReplay = errors.New("refresh token reuse detected")
	// ErrRefreshRevoked marks a refresh token explicitly invalidated by
	// logout (deny-list hit). No further action is needed.
	ErrRefreshRevoked = errors.New("refresh token revoked")
	// ErrRefreshUnavailable means the store could not answer; callers decide
	// their own policy (fail closed is preferred for refresh).
	ErrRefreshUnavailable = errors.New("refresh store unavailable")
)

var grantScript = redis.NewScript(`
local v = redis.call('GET', KEYS[1])
if v then
  redis.call('DEL', KEYS[1])
end
return v
`)

// RefreshTokenStore turns long-lived refresh tokens into single-use credentials:
// a freshly issued token gets an allow-entry, rotation atomically consumes it,
// and logout writes a deny-entry that rejects later replays.
type RefreshTokenStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRefreshTokenStore(client *redis.Client, ttl time.Duration) *RefreshTokenStore {
	return &RefreshTokenStore{client: client, ttl: ttl}
}

func allowKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "refresh_allow:" + hex.EncodeToString(sum[:])
}

func denyKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "refresh_deny:" + hex.EncodeToString(sum[:])
}

// Allow records a freshly issued refresh token as valid for a single refresh.
func (s *RefreshTokenStore) Allow(ctx context.Context, token, userID string) error {
	return s.client.Set(ctx, allowKey(token), userID, s.ttl).Err()
}

// Deny invalidates a specific refresh token (logout). The deny-entry lives as
// long as the allow-entry so replays after logout get a clean rejection.
func (s *RefreshTokenStore) Deny(ctx context.Context, token string) error {
	return s.client.Set(ctx, denyKey(token), "1", s.ttl).Err()
}

// Grant atomically consumes a token for rotation and returns the owning user
// id. A token explicitly revoked via Deny yields ErrRefreshRevoked (checked
// first so logout stays effective even while the allow-entry exists);
// replaying a consumed/unknown token yields ErrRefreshReplay.
func (s *RefreshTokenStore) Grant(ctx context.Context, token string) (string, error) {
	denied, err := s.client.Exists(ctx, denyKey(token)).Result()
	if err != nil {
		return "", ErrRefreshUnavailable
	}
	if denied > 0 {
		return "", ErrRefreshRevoked
	}

	userID, err := grantScript.Run(ctx, s.client, []string{allowKey(token)}).Text()
	if err != nil {
		if err == redis.Nil {
			return "", ErrRefreshReplay
		}
		return "", ErrRefreshUnavailable
	}
	return userID, nil
}
