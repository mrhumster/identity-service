package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/notifier"
	"github.com/mrhumster/identity-service/internal/repository"
	"github.com/redis/go-redis/v9"
)

var ErrInvalidVerificationToken = errors.New("invalid or expired verification token")

const (
	verifyKeyPrefix     = "verify:"
	verifyUserKeyPrefix = "verify_user:"
)

// VerificationService генерирует и валидирует одноразовые токены подтверждения
// email. Токены хранятся в Redis с TTL и связываются с userID. Повторный
// CreateToken отзывает предыдущий токен пользователя (защита от утечки старых
// ссылок: verify и так одноразовый, а revoke делает его недействительным сразу
// при регенерации).
type VerificationService struct {
	redis    *redis.Client
	repo     repository.UserRepository
	ttl      time.Duration
	notifier *notifier.Notifier
}

func NewVerificationService(redis *redis.Client, repo repository.UserRepository, ttl time.Duration) *VerificationService {
	return &VerificationService{redis: redis, repo: repo, ttl: ttl}
}

// WithNotifier attaches the email notifier used to deliver verification links.
// When nil (default) the service only stores the token and logs it.
func (s *VerificationService) WithNotifier(n *notifier.Notifier) *VerificationService {
	s.notifier = n
	return s
}

// CreateToken генерирует новый токен для userID и отзывает предыдущий.
func (s *VerificationService) CreateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if old, err := s.redis.Get(ctx, verifyUserKeyPrefix+userID.String()).Result(); err == nil && old != "" {
		_ = s.redis.Del(ctx, verifyKeyPrefix+old)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}
	token := hex.EncodeToString(b)

	if err := s.redis.Set(ctx, verifyKeyPrefix+token, userID.String(), s.ttl).Err(); err != nil {
		return "", fmt.Errorf("store verification token: %w", err)
	}
	if err := s.redis.Set(ctx, verifyUserKeyPrefix+userID.String(), token, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("store verification token index: %w", err)
	}

	s.enqueueNotification(ctx, userID, token)
	return token, nil
}

// enqueueNotification ставит в asynq-очередь отправку письма верификации.
// Best-effort: при любой ошибке только warn-лог, реестрация/ресенд не падают.
func (s *VerificationService) enqueueNotification(ctx context.Context, userID uuid.UUID, token string) {
	if s.notifier == nil {
		return
	}
	user, err := s.repo.ReadUserByID(ctx, userID)
	if err != nil || user == nil || user.Email == "" {
		slog.Warn("verification email: cannot resolve user",
			"user_id", userID.String(),
			"error", err,
		)
		return
	}
	if err := s.notifier.EnqueueVerification(ctx, userID, user.Email, token); err != nil {
		slog.Warn("verification email: enqueue failed",
			"user_id", userID.String(),
			"error", err,
		)
	}
}

// VerifyToken проверяет токен, помечает пользователя verified и делает токен
// одноразовым (удаляет пару ключей). Возвращает userID после успеха.
func (s *VerificationService) VerifyToken(ctx context.Context, token string) (*uuid.UUID, error) {
	if token == "" {
		return nil, ErrInvalidVerificationToken
	}
	userIDStr, err := s.redis.Get(ctx, verifyKeyPrefix+token).Result()
	if err != nil || userIDStr == "" {
		return nil, ErrInvalidVerificationToken
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}

	_ = s.redis.Del(ctx, verifyKeyPrefix+token)
	_ = s.redis.Del(ctx, verifyUserKeyPrefix+userID.String())

	if err := s.repo.UpdateEmailVerified(ctx, userID, true); err != nil {
		return nil, fmt.Errorf("mark email verified: %w", err)
	}
	return &userID, nil
}