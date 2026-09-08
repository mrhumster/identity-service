package grpctls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ServerTLSCreds builds server-side mTLS credentials: it presents its own
// certificate (certFile/keyFile) and requires every client certificate to be
// signed by the CA in caFile.
func ServerTLSCreds(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}), nil
}

// ClientTLSCreds builds client-side mTLS credentials: it presents its own
// certificate (certFile/keyFile) and verifies the server against the CA in
// caFile for serverName.
func ClientTLSCreds(certFile, keyFile, caFile, serverName string) (credentials.TransportCredentials, error) {
	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client keypair: %w", err)
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}), nil
}

// PeerOU returns the organizational unit of the authenticated client peer.
func PeerOU(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", errors.New("no peer info in context")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", errors.New("peer is not authenticated over TLS")
	}

	certs := tlsInfo.State.PeerCertificates
	if len(certs) == 0 {
		return "", errors.New("peer presented no certificates")
	}

	ou := certs[0].Subject.OrganizationalUnit
	if len(ou) == 0 {
		return "", errors.New("peer certificate has no organizational unit")
	}

	return ou[0], nil
}

// AllowOUsInterceptor is a unary interceptor that permits only presenters of
// certificates whose organizational unit is in the allowed set. It must be
// combined with mTLS credentials; otherwise it rejects every call.
func AllowOUsInterceptor(allowed ...string) grpc.UnaryServerInterceptor {
	allow := make(map[string]struct{}, len(allowed))
	for _, ou := range allowed {
		allow[ou] = struct{}{}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ou, err := PeerOU(ctx)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "peer authentication failed")
		}
		if _, ok := allow[ou]; !ok {
			return nil, status.Errorf(codes.PermissionDenied, "peer OU %q not allowed", ou)
		}
		return handler(ctx, req)
	}
}