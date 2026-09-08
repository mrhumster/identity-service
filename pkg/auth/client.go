package auth

import (
	"github.com/mrhumster/identity-service/internal/permission"
	"google.golang.org/grpc/credentials"
)

type PermissionClientWrapper struct {
	*permission.PermissionGRPCClient
}

// NewPermissionClient connects with insecure credentials (insecure, for tests/local).
func NewPermissionClient(url string) (*PermissionClientWrapper, error) {
	return NewPermissionClientWithTLS(url, nil)
}

// NewPermissionClientWithTLS connects with the given TLS client credentials.
func NewPermissionClientWithTLS(url string, creds credentials.TransportCredentials) (*PermissionClientWrapper, error) {
	client, err := permission.NewPermissionGRPCClient(url, creds)
	if err != nil {
		return nil, err
	}
	return &PermissionClientWrapper{PermissionGRPCClient: client}, nil
}

var _ PermissionClient = (*PermissionClientWrapper)(nil)
