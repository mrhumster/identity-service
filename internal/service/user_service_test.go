package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/domain/models"
	repomock "github.com/mrhumster/identity-service/internal/repository/mock"
	authmock "github.com/mrhumster/identity-service/pkg/auth/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUserService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomock.NewMockUserRepository(ctrl)
	permissionClient := authmock.NewMockPermissionClient(ctrl)

	expectedUser := models.User{
		Email: "testuser123@domain.com",
	}
	userID := uuid.New()
	expectedUser.ID = userID
	expectedUser.SetPassword("******")

	repo.EXPECT().
		CreateUser(
			gomock.Any(),
			gomock.AssignableToTypeOf(models.User{})).
		Return(&userID, nil).
		Times(1)

	permissionClient.EXPECT().
		AddPolicy(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
		Return(true, nil).
		AnyTimes()

	permissionClient.EXPECT().
		AddRoleForUser(
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
		Return(true, nil).
		AnyTimes()

	service := NewUserService(repo, permissionClient)

	ctx := context.Background()
	id, err := service.CreateUser(ctx, expectedUser)

	assert.NoError(t, err)
	require.Equal(t, userID, *id)
}

func TestUserService_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomock.NewMockUserRepository(ctrl)
	permissionClient := authmock.NewMockPermissionClient(ctrl)

	service := NewUserService(repo, permissionClient)

	expectedUser := models.User{
		Email: "testuser123@domain.com",
	}
	userID := uuid.New()
	expectedUser.ID = userID
	expectedUser.SetPassword("******")

	repo.EXPECT().
		CreateUser(
			gomock.Any(),
			gomock.AssignableToTypeOf(models.User{})).
		Return(&userID, nil).
		Times(1)

	repo.EXPECT().
		ReadUserByID(
			gomock.Any(),
			userID).
		Return(&expectedUser, nil)

	repo.EXPECT().
		ReadUserByEmail(
			gomock.Any(),
			expectedUser.Email,
		).
		Return(&expectedUser, nil)

	permissionClient.EXPECT().
		AddPolicy(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
		Return(true, nil).
		AnyTimes()

	permissionClient.EXPECT().
		AddRoleForUser(
			gomock.Any(),
			gomock.Any(),
			gomock.Any()).
		Return(true, nil).
		AnyTimes()

	ctx := context.Background()
	t.Run("Password validate success", func(t *testing.T) {

		id, err := service.CreateUser(ctx, expectedUser)
		assert.NoError(t, err)
		_, err = service.ReadUser(ctx, *id)
		assert.NoError(t, err)
		_, err = service.GetUserByEmail(ctx, expectedUser.Email)
		assert.NoError(t, err)
	})
}

func TestUserService_Create_NormalizesEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomock.NewMockUserRepository(ctrl)
	permissionClient := authmock.NewMockPermissionClient(ctrl)

	repo.EXPECT().
		CreateUser(gomock.Any(), gomock.AssignableToTypeOf(models.User{})).
		DoAndReturn(func(_ context.Context, user models.User) (*uuid.UUID, error) {
			assert.Equal(t, "testuser123@domain.com", user.Email, "email must be normalized before insert")
			id := uuid.New()
			return &id, nil
		}).
		Times(1)

	permissionClient.EXPECT().
		AddPolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil).
		AnyTimes()

	permissionClient.EXPECT().
		AddRoleForUser(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil).
		AnyTimes()

	service := NewUserService(repo, permissionClient)
	_, err := service.CreateUser(context.Background(), models.User{Email: "  TESTUser123@Domain.com  "})
	assert.NoError(t, err)
}

func TestUserService_GetUserByEmail_Normalizes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomock.NewMockUserRepository(ctrl)
	permissionClient := authmock.NewMockPermissionClient(ctrl)

	expected := &models.User{Email: "me@xomrkob.ru"}

	repo.EXPECT().
		ReadUserByEmail(gomock.Any(), "me@xomrkob.ru").
		Return(expected, nil).
		Times(1)

	service := NewUserService(repo, permissionClient)
	u, err := service.GetUserByEmail(context.Background(), "  ME@Xomrkob.RU  ")
	require.NoError(t, err)
	assert.Equal(t, "me@xomrkob.ru", u.Email)
}
