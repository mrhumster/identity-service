//go:generate mockgen -source=user_repository.go -destination=./mock/user_repository_mock.go -package=repomock
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/domain/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user models.User) (*uuid.UUID, error)
	ReadUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, user request.UpdateUserRequest) (*uuid.UUID, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	ReadUserList(ctx context.Context, l, page int64) ([]models.User, int64, error)
	ReadUserByEmail(ctx context.Context, value string) (*models.User, error)
	Exists(ctx context.Context, id uuid.UUID) bool
	UpdateTokenVersion(ctx context.Context, userID *uuid.UUID, version string) error
}
