package auth

import "github.com/mrhumster/identity-service/internal/permission"

type PermissionClientWrapper struct {
	*permission.PermissionGRPCClient
}

func NewPermissionClient(url string) (*PermissionClientWrapper, error) {
	client, err := permission.NewPermissionGRPCClient(url)
	if err != nil {
		return nil, err
	}
	return &PermissionClientWrapper{PermissionGRPCClient: client}, nil
}

var _ PermissionClient = (*PermissionClientWrapper)(nil)
