package permission

import (
	"context"
	"fmt"

	"github.com/mrhumster/identity-service/gen/go/permission"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PermissionGRPCClient struct {
	conn    *grpc.ClientConn
	service permission.PermissionServiceClient
}

func NewPermissionGRPCClient(url string) (*PermissionGRPCClient, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("⚠️ failed to connect to auth service: %w", err)
	}

	return &PermissionGRPCClient{
		conn:    conn,
		service: permission.NewPermissionServiceClient(conn),
	}, nil
}

func (c *PermissionGRPCClient) Close() error {
	return c.conn.Close()
}

func (c *PermissionGRPCClient) CheckPermission(ctx context.Context, userId, resource, action string) (bool, error) {
	resp, err := c.service.CheckPermission(ctx, &permission.CheckPermissionRequest{
		UserId:   userId,
		Resource: resource,
		Action:   action,
	})
	if err != nil {
		return false, fmt.Errorf("permission check failed: %w", err)
	}
	return resp.Allowed, nil
}

func (c *PermissionGRPCClient) AddPolicy(ctx context.Context, userId, resource, action string) (bool, error) {
	resp, err := c.service.AddPolicy(ctx, &permission.AddPolicyRequest{
		Policy:     userId,
		Resorce:    resource,
		Permission: action,
	})
	if err != nil {
		return false, fmt.Errorf("add policy failed: %w", err)
	}
	return resp.Added, nil
}

func (c *PermissionGRPCClient) RemovePolicy(ctx context.Context, userId, resource, action string) (bool, error) {
	resp, err := c.service.RemovePolicy(ctx, &permission.RemovePolicyRequest{
		Policy:     userId,
		Resource:   resource,
		Permission: action,
	})
	if err != nil {
		return false, fmt.Errorf("remove policy failed: %w", err)
	}
	return resp.Removed, nil
}

func (c *PermissionGRPCClient) AddPolicyIfNotExists(ctx context.Context, userId, resource, action string) (bool, error) {
	resp, err := c.service.AddPolicyIfNotExists(ctx, &permission.AddPolicyIfNotExistsRequest{
		Policy:     userId,
		Resource:   resource,
		Permission: action,
	})
	if err != nil {
		return false, fmt.Errorf("add policy failed: %w", err)
	}
	return resp.Exists, nil
}
