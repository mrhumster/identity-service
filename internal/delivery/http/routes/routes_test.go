package routes

import (
	"testing"

	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/database"
	"github.com/mrhumster/identity-service/pkg/auth"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestRoutes(t *testing.T) {
	cfg, _ := config.TestConfig()
	db := database.SetupDatabase(cfg)
	permissionGRPCClient, _ := auth.NewPermissionClient(cfg.Server.AuthServiceAddr)
	defer permissionGRPCClient.Close()
	SetupRoutes(db, "test", permissionGRPCClient)
	assert.IsType(t, &gorm.DB{}, db)
}
