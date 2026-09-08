//go:generate mockgen -source=permission_client.go -destination=./mock/permission_client_mock.go -package=authmock

package auth

import "context"

type PermissionClient interface {
	CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)
	AddPolicy(ctx context.Context, userID, resource, action string) (bool, error)
	RemovePolicy(ctx context.Context, userID, resource, action string) (bool, error)
	AddPolicyIfNotExists(ctx context.Context, userID, resource, action string) (bool, error)
	AddRoleForUser(ctx context.Context, userID, role string) (bool, error)
	RemoveRoleForUser(ctx context.Context, userID, role string) (bool, error)
	Close() error
}
