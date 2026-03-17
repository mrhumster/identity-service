package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/delivery/http/dto/request"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/sony/gobreaker"
	"gorm.io/gorm"
	"log"
	"time"
)

var dbCircuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "Database",
	MaxRequests: 3,
	Interval:    30 * time.Second,
	Timeout:     60 * time.Second,
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures > 3
	},
	OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
		log.Printf("Circuit Breaker %s: %s -> %s", name, from, to)
	},
})

func WithCircuitBreaker(fn func() error) error {
	_, err := dbCircuitBreaker.Execute(func() (any, error) {
		return nil, fn()
	})
	return err
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) CreateUser(ctx context.Context, user models.User) (*uuid.UUID, error) {
	result := r.db.WithContext(ctx).Create(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user.ID, nil
}

func (r *GormUserRepository) ReadUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user *models.User
	result := r.db.WithContext(ctx).First(&user, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func (r *GormUserRepository) UpdateUser(ctx context.Context, id uuid.UUID, user request.UpdateUserRequest) (*uuid.UUID, error) {
	var userForUpdate *models.User
	result := r.db.WithContext(ctx).First(&userForUpdate, id)
	if result.Error != nil {
		return nil, result.Error
	}
	userForUpdate.FillInTheUpdateRequest(user)
	result = r.db.WithContext(ctx).Save(&userForUpdate)
	if result.Error != nil {
		return nil, result.Error
	}
	return &userForUpdate.ID, nil
}

func (r *GormUserRepository) DeleteUserByID(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	return result.Error
}

func (r *GormUserRepository) ReadUserList(ctx context.Context, l, page int64) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	limit := int(l)
	offset := int((page - 1) * l)
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	result := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users)
	if result.Error != nil {
		return []models.User{}, int64(0), result.Error
	}
	return users, total, nil
}

func (r *GormUserRepository) ReadUserByEmail(ctx context.Context, value string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Model(&models.User{}).First(&user, "email = ?", value).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Exists(ctx context.Context, id uuid.UUID) bool {
	u, _ := r.ReadUserByID(ctx, id)
	return u != nil
}

func (r *GormUserRepository) UpdateTokenVersion(ctx context.Context, userID *uuid.UUID, version string) error {
	var userForUpdate *models.User
	result := r.db.WithContext(ctx).First(&userForUpdate, userID)
	if result.Error != nil {
		return result.Error
	}
	userForUpdate.TokenVersion = version
	result = r.db.WithContext(ctx).Save(&userForUpdate)
	return result.Error
}
