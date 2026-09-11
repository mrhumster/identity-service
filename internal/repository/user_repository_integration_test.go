package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateUniqueUser() models.User {
	timestamp := time.Now().UnixNano()
	uniqueEmail := fmt.Sprintf("testuser_%d@test.local", timestamp)

	return models.User{
		PasswordHash: "#########",
		Email:        uniqueEmail,
	}
}

func TestUserRepository_CreateRepo(t *testing.T) {
	db := testutils.GetTestDB()
	defer testutils.CleanTestDatabase()
	repo := NewGormUserRepository(db)
	ctx := context.Background()

	t.Run("Create and Read user", func(t *testing.T) {
		user := generateUniqueUser()
		id, err := repo.CreateUser(ctx, user)
		if id == nil {
			t.Errorf("⚠️ Error create user: %v", err)
		}
		assert.NoError(t, err)

		_, err = repo.ReadUserByID(ctx, *id)
		assert.NoError(t, err)
		notID, _ := uuid.NewUUID()
		_, err = repo.ReadUserByID(ctx, notID)
		assert.Error(t, err)
	})
}

func TestUserRepository_ReadList(t *testing.T) {
	db := testutils.GetTestDB()
	defer testutils.CleanTestDatabase()
	repo := NewGormUserRepository(db)
	ctx := context.Background()
	t.Run("Read userlist", func(t *testing.T) {
		user := models.User{
			PasswordHash: "#########",
			Email:        "testuser@test.local",
		}
		repo.CreateUser(ctx, user)
		user = models.User{
			PasswordHash: "#########",
			Email:        "testuser2@test.local",
		}
		repo.CreateUser(ctx, user)
		users, _, err := repo.ReadUserList(ctx, 2, 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
		assert.Equal(t, 2, len(users))
	})

}

func TestUserRepository_ReadByEmail(t *testing.T) {
	db := testutils.GetTestDB()
	defer testutils.CleanTestDatabase()
	repo := NewGormUserRepository(db)
	ctx := context.Background()
	t.Run("Create user and get user by email", func(t *testing.T) {
		user := models.User{
			Email:        "billy@test.local",
			PasswordHash: "***1234***",
		}
		id, err := repo.CreateUser(ctx, user)
		assert.NotEmpty(t, id)
		assert.NoError(t, err)
		u, _ := repo.ReadUserByID(ctx, *id)
		u, err = repo.ReadUserByEmail(ctx, "billy@test.local")
		assert.NoError(t, err)
		assert.Equal(t, *id, (*u).ID)
		assert.Equal(t, user.Email, u.Email)
	})
}

func TestUserRepository_UserExist(t *testing.T) {
	db := testutils.GetTestDB()
	defer testutils.CleanTestDatabase()
	repo := NewGormUserRepository(db)
	ctx := context.Background()
	t.Run("Create user and check that user exisr", func(t *testing.T) {
		user := models.User{
			Email:        "billy@test.local",
			PasswordHash: "***1234***",
		}
		id, err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Error(err.Error())
		}
		exists := repo.Exists(ctx, *id)
		assert.True(t, exists)
	})
}

func TestUserRepository_UpdateEmailVerified(t *testing.T) {
	db := testutils.GetTestDB()
	defer testutils.CleanTestDatabase()
	repo := NewGormUserRepository(db)
	ctx := context.Background()

	user := models.User{
		Email:        "verify@test.local",
		PasswordHash: "***1234***",
	}
	id, err := repo.CreateUser(ctx, user)
	require.NoError(t, err)

	u, err := repo.ReadUserByID(ctx, *id)
	require.NoError(t, err)
	assert.False(t, u.EmailVerified)

	require.NoError(t, repo.UpdateEmailVerified(ctx, *id, true))

	u, err = repo.ReadUserByID(ctx, *id)
	require.NoError(t, err)
	assert.True(t, u.EmailVerified)

	require.NoError(t, repo.UpdateEmailVerified(ctx, *id, false))
	u, err = repo.ReadUserByID(ctx, *id)
	require.NoError(t, err)
	assert.False(t, u.EmailVerified)
}
