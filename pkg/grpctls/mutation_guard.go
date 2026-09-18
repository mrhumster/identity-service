package grpctls

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// permissionMutations lists every mutating RPC of the permission service and
// the peer OUs allowed to invoke it. Reads (CheckPermission) are open to any
// authenticated peer; each mutation requires a fixed mTLS client.
var permissionMutations = map[string][]string{
	"AddPolicy":            {"stream-service", "identity-service"},
	"AddPolicyIfNotExists": {"stream-service", "identity-service"},
	"RemovePolicy":         {"identity-service"},
	"AddRoleForUser":       {"identity-service"},
	"RemoveRoleForUser":    {"identity-service"},
}

// MutationGuard is a unary interceptor that restricts mutating permission
// RPCs to a fixed set of peer OUs (service-to-service mTLS). Read-only calls
// pass through untouched. When the peer is not authenticated over TLS the
// guard fails closed, so a plaintext gRPC endpoint can never alter policies.
func MutationGuard() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		method := info.FullMethod
		if i := strings.LastIndex(method, "/"); i >= 0 {
			method = method[i+1:]
		}
		allowed, isMutation := permissionMutations[method]
		if !isMutation {
			return handler(ctx, req)
		}

		ou, err := PeerOU(ctx)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "peer authentication failed")
		}
		for _, a := range allowed {
			if ou == a {
				return handler(ctx, req)
			}
		}
		return nil, status.Errorf(codes.PermissionDenied, "peer OU %q not allowed for permission mutation %q", ou, method)
	}
}