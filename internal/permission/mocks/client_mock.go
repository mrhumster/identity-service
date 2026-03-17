package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type PermissionClientMock struct {
	mock.Mock
}

func (m *PermissionClientMock) AddPolicy(ctx context.Context, userID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *PermissionClientMock) RemovePolicy(ctx context.Context, userID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *PermissionClientMock) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *PermissionClientMock) AddPolicyIfNotExists(ctx context.Context, userID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *PermissionClientMock) Close() error {
	args := m.Called()
	return args.Error(0)
}
