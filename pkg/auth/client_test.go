package auth_test

import (
	"context"
	"testing"

	"github.com/mrhumster/identity-service/internal/permission/mocks"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionCLientInterface(t *testing.T) {
	var _ auth.PermissionClient = (*mocks.PermissionClientMock)(nil)
	t.Log("Interface satisfaction verified")
}

func TestPermissionClientMock(t *testing.T) {
	ctx := context.Background()
	mockClient := &mocks.PermissionClientMock{}

	t.Run("CheckPermission", func(t *testing.T) {
		mockClient.On("CheckPermission", ctx, "user123", "stream", "read").Return(true, nil)
		allowed, err := mockClient.CheckPermission(ctx, "user123", "stream", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
		mockClient.AssertExpectations(t)
	})

	t.Run("AddPolicy", func(t *testing.T) {
		mockClient.On("AddPolicy", ctx, "user123", "stream456", "update").Return(true, nil)
		added, err := mockClient.AddPolicy(ctx, "user123", "stream456", "update")
		require.NoError(t, err)
		assert.True(t, added)
		mockClient.AssertExpectations(t)
	})

}
