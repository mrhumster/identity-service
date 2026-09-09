package routes

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/delivery/http/handler"
	"github.com/mrhumster/identity-service/internal/repository"
	"github.com/mrhumster/identity-service/internal/service"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/mrhumster/identity-service/pkg/middleware"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB, mode string, permissionClient auth.PermissionClient) *gin.Engine {
	// MODE
	if mode == "test" {
		gin.SetMode(gin.TestMode)
	}
	if mode == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	// GIN ROUTE
	r := gin.New()

	r.Use(middleware.StructuredLog())
	r.Use(gin.Recovery())

	// CONFIGURATION
	cfg, _ := config.LoadConfig()
	if mode == "test" || mode == "debug" {
		cfg, _ = config.TestConfig()
	}

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "PATCH", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// REPOSITORIES
	userRepo := repository.NewGormUserRepository(db)

	// SERVICES
	userService := service.NewUserService(userRepo, permissionClient)
	tokenService, err := service.NewTokenService(&cfg.JWT)
	if err != nil {
		fmt.Printf("⚠️ SetupRoutes: %v", err)
		panic("Error create new token service")
	}

	// HANDLERS
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(userService, tokenService, cfg.Server.JwtSecret, cfg.Server.Domain)
	commonHandler := handler.NewCommonHandler(tokenService)

	// ROUTE
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/users", userHandler.CreateUser)
	r.POST("/auth/refresh", authHandler.Refresh)

	auth := r.Group("/auth/", middleware.AuthMiddleware(tokenService))
	{
		auth.GET("/who", userHandler.GetAuthUser)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/logout-all", authHandler.LogoutAll)
		auth.GET("/users", middleware.Authorize(permissionClient, "users", "read"), userHandler.ReadUsers)
		auth.GET("/users/:id", middleware.Authorize(permissionClient, "users", "read"), userHandler.ReadUser)
		auth.PATCH("/users/:id", middleware.Authorize(permissionClient, "users", "write"), userHandler.Update)
		auth.DELETE("/users/:id", middleware.Authorize(permissionClient, "users", "delete"), userHandler.Delete)
	}

	r.GET("/auth/public-key", commonHandler.GetPublicKey)
	r.GET("/auth/health", func(c *gin.Context) {
		if _, err := db.DB(); err != nil {
			log.Println("⚠️ PG ERROR: ", err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})
	return r
}
