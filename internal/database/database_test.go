package database

import (
	"testing"

	"github.com/mrhumster/identity-service/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupDatabase(t *testing.T) {
	t.Run("should create database connection with valid config", func(t *testing.T) {
		cfg, err := config.TestConfig()

		if err != nil {
			t.Errorf("⚠️ TestSetupDatabase error: %v", err)
		}

		assert.NotPanics(t, func() {
			db := SetupDatabase(cfg)
			require.NotNil(t, db)

			sqlDB, err := db.DB()
			if assert.NoError(t, err) {
				defer sqlDB.Close()
				assert.NoError(t, sqlDB.Ping())
			}
		})
	})
}
