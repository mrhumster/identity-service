package database

import (
	"fmt"
	"log"
	"time"

	"github.com/mrhumster/identity-service/config"
	"github.com/mrhumster/identity-service/internal/domain/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabase(cfg *config.Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.GetDsn()), &gorm.Config{})
	if err != nil {
		fmt.Printf("⚠️ SetupDatabase error: %v", err)
		panic("⚠️ GORM not open DB")

	}
	sqlDb, err := db.DB()
	if err != nil {
		panic("⚠️ GORM not open DB")
	}
	sqlDb.SetMaxOpenConns(100)
	sqlDb.SetMaxIdleConns(10)
	sqlDb.SetConnMaxLifetime(time.Hour)
	sqlDb.SetConnMaxIdleTime(30 * time.Minute)
	log.Printf("🔌  Creating uuid-ossp extension...")
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	db.AutoMigrate(&models.User{})
	return db
}
