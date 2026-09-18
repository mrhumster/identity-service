package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestConfig_GetEnv(t *testing.T) {
	key := "NOT_EXISTS"
	v := getEnv(key, "exist")
	assert.Equal(t, v, "exist")
	v = getEnv("USER", "user")
	assert.NotEqual(t, v, "user")
}

func TestTestConfig(t *testing.T) {
	cfg, err := TestConfig()
	if err != nil {
		t.Errorf("Test config err: %v", err)
	}
	assert.NotEmpty(t, cfg.Database.Host)
	assert.NotEmpty(t, cfg.Database.Name)
	assert.NotEmpty(t, cfg.Server.ServerAddr)
	dsn := cfg.GetDsn()
	assert.NotEmpty(t, dsn)
}

func TestConfig_AuthRateLimitPerMin(t *testing.T) {
	t.Setenv("TEST_AUTH_RATE_LIMIT", "")
	cfg, err := TestConfig()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.Server.AuthRateLimitPerMin)

	t.Setenv("TEST_AUTH_RATE_LIMIT", "10")
	cfg, err = TestConfig()
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.Server.AuthRateLimitPerMin)
}
