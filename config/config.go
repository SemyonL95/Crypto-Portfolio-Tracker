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
	App         AppConfig
}

type AppConfig struct {
	Environment string // "development" or "production"
	TokensPath  string // Path to tokens JSON file
	LogLevel    string // "debug", "info", "warn", "error"
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
	Provider         string // "etherscan" or "mock"
	RequestTimeout   time.Duration
	RateLimitRPS     int
	EtherscanAPIKey  string
	EtherscanBaseURL string
}

type DatabaseConfig struct {
	Path string // SQLite database file path
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
			Provider:         getEnv("TRANSACTION_PROVIDER", "etherscan"),
			RequestTimeout:   getDurationEnv("TRANSACTION_REQUEST_TIMEOUT", 10*time.Second),
			RateLimitRPS:     getIntEnv("TRANSACTION_RATE_LIMIT_RPS", 5),
			EtherscanAPIKey:  getEnv("ETHERSCAN_API_KEY", ""),
			EtherscanBaseURL: getEnv("ETHERSCAN_BASE_URL", "https://api.etherscan.io/api"),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./data/portfolio.db"),
		},
		App: AppConfig{
			Environment: getEnv("APP_ENV", "development"),
			TokensPath:  getEnv("TOKENS_PATH", "./static/tokens.json"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
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
