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
	"github.com/mrhumster/identity-service/internal/permission"
	"github.com/mrhumster/identity-service/internal/service"
	"github.com/mrhumster/identity-service/pkg/auth"
	"google.golang.org/grpc"
)

func main() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	slog.SetDefault(logger)
	slog.Info("🚀 Start identity-service", "version", "1.2.14")

	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("❌ Config: %s", err.Error()))
	}
	db := database.SetupDatabase(cfg)

	permGRPCClient, err := auth.NewPermissionClient(cfg.Server.AuthServiceAddr)
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

		grpcServer := grpc.NewServer()

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
