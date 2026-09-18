package grpctls_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/mrhumster/identity-service/pkg/grpctls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func testCertWithOU(t *testing.T, ou string) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{OrganizationalUnit: []string{ou}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func ctxWithPeerOU(t *testing.T, ou string) context.Context {
	cert := testCertWithOU(t, ou)
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}},
	})
}

func TestMutationGuard(t *testing.T) {
	pkg := "/permission.PermissionService"

	tests := []struct {
		name       string
		method     string
		ou         *string
		wantCode   codes.Code
		wantCalled bool
	}{
		{"read passes through for any OU", pkg + "/CheckPermission", strPtr("thumbnail-service"), codes.OK, true},
		{"read passes through without TLS", pkg + "/CheckPermission", nil, codes.OK, true},
		{"AddPolicy allowed for stream-service", pkg + "/AddPolicy", strPtr("stream-service"), codes.OK, true},
		{"AddPolicy allowed for identity-service", pkg + "/AddPolicy", strPtr("identity-service"), codes.OK, true},
		{"AddPolicy denied for other OU", pkg + "/AddPolicy", strPtr("thumbnail-service"), codes.PermissionDenied, false},
		{"AddPolicy denied without TLS (fail closed)", pkg + "/AddPolicy", nil, codes.PermissionDenied, false},
		{"AddPolicyIfNotExists allowed for stream-service", pkg + "/AddPolicyIfNotExists", strPtr("stream-service"), codes.OK, true},
		{"AddPolicyIfNotExists allowed for self", pkg + "/AddPolicyIfNotExists", strPtr("identity-service"), codes.OK, true},
		{"RemovePolicy allowed for self", pkg + "/RemovePolicy", strPtr("identity-service"), codes.OK, true},
		{"RemovePolicy denied for stream-service", pkg + "/RemovePolicy", strPtr("stream-service"), codes.PermissionDenied, false},
		{"AddRoleForUser allowed for self", pkg + "/AddRoleForUser", strPtr("identity-service"), codes.OK, true},
		{"AddRoleForUser denied for stream-service", pkg + "/AddRoleForUser", strPtr("stream-service"), codes.PermissionDenied, false},
		{"RemoveRoleForUser allowed for self", pkg + "/RemoveRoleForUser", strPtr("identity-service"), codes.OK, true},
		{"RemoveRoleForUser denied for stream-service", pkg + "/RemoveRoleForUser", strPtr("stream-service"), codes.PermissionDenied, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				return "ok", nil
			}
			info := &grpc.UnaryServerInfo{FullMethod: tc.method}

			ctx := context.Background()
			if tc.ou != nil {
				ctx = ctxWithPeerOU(t, *tc.ou)
			}

			interceptor := grpctls.MutationGuard()
			resp, err := interceptor(ctx, nil, info, handler)

			assert.Equal(t, tc.wantCalled, called, "handler invocation mismatch")
			if tc.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.wantCode, status.Code(err))
			}
		})
	}
}

func strPtr(s string) *string { return &s }