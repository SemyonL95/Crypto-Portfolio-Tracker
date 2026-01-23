package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server      ServerConfig
	Price       PriceConfig
	Transaction TransactionConfig
	Database    DatabaseConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type PriceConfig struct {
	Provider        string // "coingecko" or "mock"
	CacheTTL        time.Duration
	RequestTimeout  time.Duration
	RateLimitRPS    int
	CoinGeckoAPIKey string
	FallbackEnabled bool
}

type TransactionConfig struct {
	RequestTimeout time.Duration
	RateLimitRPS   int
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", ""),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Price: PriceConfig{
			Provider:        getEnv("PRICE_PROVIDER", "coingecko"),
			CacheTTL:        getDurationEnv("PRICE_CACHE_TTL", 60*time.Second),
			RequestTimeout:  getDurationEnv("PRICE_REQUEST_TIMEOUT", 10*time.Second),
			RateLimitRPS:    getIntEnv("PRICE_RATE_LIMIT_RPS", 10),
			CoinGeckoAPIKey: getEnv("COINGECKO_API_KEY", ""),
			FallbackEnabled: getBoolEnv("PRICE_FALLBACK_ENABLED", true),
		},
		Transaction: TransactionConfig{
			RequestTimeout: getDurationEnv("TRANSACTION_REQUEST_TIMEOUT", 10*time.Second),
			RateLimitRPS:   getIntEnv("TRANSACTION_RATE_LIMIT_RPS", 5),
		},
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
