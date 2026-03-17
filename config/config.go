package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SslMode  string
	TimeZone string
}

type Redis struct {
	Addr     string
	Password string
}

type Server struct {
	ServerAddr      string
	JwtSecret       string
	CasbinModel     string
	Domain          string
	AuthServiceAddr string
}

type JWT struct {
	AccessPrivateKey   string
	AccessPublicKey    string
	RefreshPrivateKey  string
	RefreshPublicKey   string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string `mapstructure:"jwt_issuer"`
}
type Config struct {
	Database `mapstructure:",squash"`
	Server   Server `mapstructure:"server"`
	JWT      JWT    `mapstructure:"jwt"`
	Redis    Redis  `mapstructure:"redis"`
}

func GetRootDir() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "..")
}

func LoadConfig() (*Config, error) {
	accessTokenExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("Config error. Plase set ENV JWT_ACCESS_TOKEN_EXPIRY. %v", err)
	}
	refreshTokenExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_TOKEN_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("Config error. Plase set ENV JWT_REFRESH_TOKEN_EXPIRY. %v", err)
	}

	cfg := &Config{
		Database: Database{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASS"),
			Name:     os.Getenv("DB_NAME"),
			SslMode:  "disable",
			TimeZone: "UTC",
		},
		Server: Server{
			ServerAddr:      os.Getenv("SERVER_ADDR"),
			JwtSecret:       os.Getenv("JWT_SECRET"),
			CasbinModel:     os.Getenv("CASBIN_MODEL"),
			Domain:          os.Getenv("DOMAIN"),
			AuthServiceAddr: os.Getenv("AUTH_SERVICE_ADDRESS"),
		},
		JWT: JWT{
			AccessPrivateKey:   getEnv("JWT_ACCESS_PRIVATE_KEY", ""),
			AccessPublicKey:    getEnv("JWT_ACCESS_PUBLIC_KEY", ""),
			RefreshPrivateKey:  getEnv("JWT_REFRESH_PRIVATE_KEY", ""),
			RefreshPublicKey:   getEnv("JWT_REFRESH_PUBLIC_KEY", ""),
			AccessTokenExpiry:  accessTokenExpiry,
			RefreshTokenExpiry: refreshTokenExpiry,
			Issuer:             getEnv("JWT_ISSUER", "auth-service"),
		},
		Redis: Redis{
			Addr:     getEnv("REDIS_ADDR", "localhost"),
			Password: getEnv("REDIS_PASS", ""),
		},
	}
	return cfg, nil
}

func (config *Config) GetDsn() string {
	if config == nil {
		panic("⚠️ Config is nil error: ")
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.Host,
		config.User,
		config.Password,
		config.Name,
		config.Port,
		config.SslMode,
		config.TimeZone)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestConfig() (*Config, error) {
	accessTokenExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_TOKEN_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("Config error. Plase set ENV JWT_ACCESS_TOKEN_EXPIRY. %v", err)
	}
	refreshTokenExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_TOKEN_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("Config error. Plase set ENV JWT_REFRESH_TOKEN_EXPIRY. %v", err)
	}

	rootDir := GetRootDir()

	accessPrivateKeyPath := filepath.Join(rootDir, "config", "keys", "accessPrivate.pem")
	accessPublicKeyPath := filepath.Join(rootDir, "config", "keys", "accessPublic.pem")
	refreshPrivateKeyPath := filepath.Join(rootDir, "config", "keys", "refreshPrivate.pem")
	refreshPublicKeyPath := filepath.Join(rootDir, "config", "keys", "refreshPublic.pem")

	casbinModelPath := filepath.Join(rootDir, "config", "model.conf")

	accessPrivateKey, err := os.ReadFile(accessPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Config error. AccessPrivateKey not read: %w", err)
	}

	accessPublicKey, err := os.ReadFile(accessPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Config error. AccessPublicKey not read: %w", err)
	}

	refreshPrivateKey, err := os.ReadFile(refreshPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Config error. RefreshPrivateKey not read: %w", err)
	}

	refreshPublicKey, err := os.ReadFile(refreshPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Config error. RefreshPublicKey not read: %w", err)
	}

	return &Config{
		Database: Database{
			Host:     getEnv("TEST_DB_HOST", "localhost"),
			Port:     getEnv("TEST_DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASS", "Master1234"),
			Name:     getEnv("TEST_DB_NAME", "test_database1"),
			SslMode:  "disable",
			TimeZone: "UTC",
		},
		Server: Server{
			ServerAddr:      getEnv("TEST_SERVER_ADDR", ":8080"),
			JwtSecret:       getEnv("TEST_JWT_SECRET", "jwt-secret-jwt-secret"),
			CasbinModel:     getEnv("TEST_CASBIN_MODEL", casbinModelPath),
			Domain:          getEnv("TEST_DOMAIN", "localhost"),
			AuthServiceAddr: getEnv("TEST_AUTH_SERVICE_ADDRESS", "localhost:50051"),
		},
		JWT: JWT{
			AccessPrivateKey:   string(accessPrivateKey),
			AccessPublicKey:    string(accessPublicKey),
			RefreshPrivateKey:  string(refreshPrivateKey),
			RefreshPublicKey:   string(refreshPublicKey),
			AccessTokenExpiry:  accessTokenExpiry,
			RefreshTokenExpiry: refreshTokenExpiry,
			Issuer:             getEnv("JWT_ISSUER", "auth-service"),
		},
	}, nil
}
