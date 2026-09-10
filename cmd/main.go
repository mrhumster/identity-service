package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/mrhumster/identity-service/config"
	permissionpb "github.com/mrhumster/identity-service/gen/go/permission"
	"github.com/mrhumster/identity-service/internal/database"
	"github.com/mrhumster/identity-service/internal/delivery/http/routes"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"github.com/mrhumster/identity-service/internal/permission"
	"github.com/mrhumster/identity-service/internal/repository"
	"github.com/mrhumster/identity-service/internal/service"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/mrhumster/identity-service/pkg/grpctls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gorm.io/gorm"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	slog.SetDefault(logger)
	slog.Info("🚀 Start identity-service", "version", version, "build_date", buildDate)

	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("❌ Config: %s", err.Error()))
	}
	db := database.SetupDatabase(cfg)

	clientCreds := clientTLSCreds(cfg)
	permGRPCClient, err := auth.NewPermissionClientWithTLS(cfg.Server.AuthServiceAddr, clientCreds)
	if err != nil {
		panic(fmt.Sprintf("❌ Permission gRPC client: %s", err.Error()))
	}

	r := routes.SetupRoutes(db, "release", permGRPCClient)

	defer func() {
		log.Println("🟡 Closing database pool...")
		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("failed to get sql.DB: %s", err.Error())
		}
		if err := sqlDB.Close(); err != nil {
			log.Println("🟢 Database pool closed")
		}
		log.Printf("🟡 Closing gRPC client...")
		permGRPCClient.Close()
	}()

	httpErr := make(chan error, 1)
	grpcErr := make(chan error, 1)

	srv := &http.Server{
		Addr:         cfg.Server.ServerAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server starting on %s\n", cfg.Server.ServerAddr)
		log.Printf("ENV DOMAIN: %s", cfg.Server.Domain)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("🔴 Server error: ", err)
			httpErr <- err
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("🔴 Failed to listen: %v", err)
		}

		grpcOpts := []grpc.ServerOption{}
		if cfg.Server.GRPCTLSEnabled {
			serverCreds, terr := grpctls.ServerTLSCreds(cfg.Server.GRPCTLSCertFile, cfg.Server.GRPCTLSKeyFile, cfg.Server.GRPCTLSCAFile)
			if terr != nil {
				log.Fatalf("🔴 Failed to load gRPC server TLS: %v", terr)
			}
			grpcOpts = append(grpcOpts, grpc.Creds(serverCreds))
			if len(cfg.Server.GRPCTLSAllowedOUs) > 0 {
				grpcOpts = append(grpcOpts, grpc.UnaryInterceptor(grpctls.AllowOUsInterceptor(cfg.Server.GRPCTLSAllowedOUs...)))
			}
		}

		grpcServer := grpc.NewServer(grpcOpts...)

		adapter, err := gormadapter.NewAdapterByDB(db)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
		}

		enforcer, err := casbin.NewEnforcer(cfg.Server.CasbinModel, adapter)
		if err != nil {
			log.Printf("⚠️ Casbin Load Error, %s", err.Error())
			panic("⚠️ Error loading roles config")
		}

		permissionService, err := service.NewPermissionService(enforcer, cfg.Redis)
		if err != nil {
			slog.Error("Error init permission service", "error", err)
			panic("⚠️ Error init permission service")
		}

		if err := bootstrapRBAC(db, permissionService, cfg.Server.AdminEmail); err != nil {
			slog.Error("❌ RBAC bootstrap failed", "error", err)
		}

		defer func() {
			log.Println("🟡 Closing Permission Service (Watcher)...")
			permissionService.Close()
		}()

		permissionServer := permission.NewPermissionGRPCServer(permissionService)
		permissionpb.RegisterPermissionServiceServer(grpcServer, permissionServer)

		log.Printf("🛰️ gRPC server listened at %v", lis.Addr())

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("🔴 Failed to serve: %v", err)
			grpcErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🟡 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("🔴 Server shutdown error: ", err)
	}

	log.Println("🟢 Server stoped")
}

// clientTLSCreds builds mTLS client credentials for the PermissionService
// gRPC connection, or nil (insecure) when TLS is disabled.
func clientTLSCreds(cfg *config.Config) credentials.TransportCredentials {
	if !cfg.Server.GRPCTLSEnabled {
		return nil
	}
	creds, err := grpctls.ClientTLSCreds(cfg.Server.GRPCTLSCertFile, cfg.Server.GRPCTLSKeyFile, cfg.Server.GRPCTLSCAFile, "identity-service")
	if err != nil {
		panic(fmt.Sprintf("❌ gRPC client TLS: %s", err.Error()))
	}
	return creds
}

// bootstrapRBAC seeds role policies, migrates existing users to the "member"
// role and promotes the ADMIN_EMAIL user to "admin". It uses the local
// enforcer (not the gRPC client) because the gRPC server is not yet serving
// when the caller runs.
func bootstrapRBAC(db *gorm.DB, ps *service.PermissionService, adminEmail string) error {
	ctx := context.Background()
	logger := slog.Default()

	rolePolicies := []struct {
		role, obj, act string
	}{
		{"admin", "users", "read"},
		{"admin", "users/*", "read"},
		{"admin", "users/*", "write"},
		{"admin", "users/*", "delete"},
		{"admin", "stream", "read"},
		{"admin", "stream/*", "read"},
		{"admin", "stream/*", "write"},
		{"admin", "stream/*", "delete"},
		{"member", "stream", "read"},
		{"member", "stream", "write"},
	}

	for _, p := range rolePolicies {
		added, err := ps.AddPolicyIfNotExists(p.role, p.obj, p.act)
		if err != nil {
			logger.Error("❌ RBAC: failed to seed policy", "policy", p, "error", err)
			continue
		}
		if added {
			logger.Info("✅ RBAC: seeded policy", "role", p.role, "obj", p.obj, "act", p.act)
		}
	}

	userRepo := repository.NewGormUserRepository(db)
	users, _, err := userRepo.ReadUserList(ctx, 100000, 1)
	if err != nil {
		return fmt.Errorf("failed to load users for RBAC migration: %w", err)
	}

	for _, u := range users {
		uid := u.ID.String()

		roleAdded, err := ps.AddRoleForUser(uid, "member")
		if err != nil {
			logger.Error("❌ RBAC: failed to assign member role", "user", uid, "error", err)
		} else if roleAdded {
			logger.Info("✅ RBAC: assigned member role", "user", uid)
		}

		for _, p := range [][]string{{"users", "read"}, {"stream", "read"}, {"stream", "write"}} {
			removed, err := ps.RemovePolicy(uid, p[0], p[1])
			if err != nil {
				logger.Error("❌ RBAC: failed to remove legacy policy", "user", uid, "obj", p[0], "act", p[1], "error", err)
				continue
			}
			if removed {
				logger.Info("✅ RBAC: removed legacy collection policy", "user", uid, "obj", p[0], "act", p[1])
			}
		}
	}

	if adminEmail == "" {
		return nil
	}

	admin, err := userRepo.ReadUserByEmail(ctx, models.NormalizeEmail(adminEmail))
	if err != nil {
		return fmt.Errorf("ADMIN_EMAIL user not found: %w", err)
	}

	if admin.Role != "admin" {
		if err := userRepo.UpdateUserRole(ctx, admin.ID, "admin"); err != nil {
			return fmt.Errorf("failed to promote ADMIN_EMAIL user to admin: %w", err)
		}
	}
	roleAdded, err := ps.AddRoleForUser(admin.ID.String(), "admin")
	if err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}
	logger.Info("✅ RBAC: admin role ensured", "user", admin.ID.String(), "email", adminEmail, "added", roleAdded)
	return nil
}
