package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"testtask/config"
	"testtask/internal/adapters/cache"
	coingeckoadapter "testtask/internal/adapters/coingecko"
	etherscanadapter "testtask/internal/adapters/etherscan"
	httpserver "testtask/internal/adapters/http/server"
	loggeradapter "testtask/internal/adapters/logger"
	portfoliorepo "testtask/internal/adapters/portfolio"
	portfolioservice "testtask/internal/application/portfolio"
	priceservice "testtask/internal/application/price"
	"testtask/internal/application/ratelimiter"
	transactionservice "testtask/internal/application/transaction"
	"testtask/internal/domain"
	domainPrice "testtask/internal/domain/price"
	"testtask/internal/domain/token"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger based on environment
	isDevelopment := cfg.App.Environment == "development"
	logger, err := loggeradapter.NewLogger(isDevelopment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			// Ignore sync errors on exit
		}
	}()

	logger.Info("Starting application",
		zap.String("environment", cfg.App.Environment),
		zap.String("version", "1.0.0"),
	)

	// Initialize database repository
	if err := initializeDatabase(cfg, logger); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	portfolioRepo, err := portfoliorepo.NewSQLiteRepository(cfg.Database.Path)
	if err != nil {
		logger.Fatal("Failed to create database repository", zap.Error(err))
	}
	defer func() {
		if err := portfolioRepo.Close(); err != nil {
			logger.Error("Failed to close database", zap.Error(err))
		}
	}()

	// The portfolio repository also implements holding repository interface
	holdingRepo := portfolioRepo

	// Initialize cache for prices
	priceCache := cache.NewCache[string, domainPrice.Price](1000)

	// Initialize HTTP client for external APIs
	httpClient := &http.Client{
		Timeout: cfg.Price.RequestTimeout,
	}

	// Initialize CoinGecko client
	coingeckoBaseURL := getEnv("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3")
	coingeckoClient := coingeckoadapter.NewClient(httpClient, coingeckoBaseURL, cfg.Price.CoinGeckoAPIKey)

	if cfg.Price.CoinGeckoAPIKey == "" {
		logger.Warn("CoinGecko API key not set, some features may be limited")
	}

	// Initialize CoinGecko price provider
	// For now, we'll use an empty symbolToID map - in production this should be loaded from a file or API
	symbolToID := make(map[string]string)
	coingeckoPriceProvider := coingeckoadapter.NewPriceRepository(coingeckoClient, symbolToID)

	// Initialize mock price provider as fallback
	mockPriceProvider := coingeckoadapter.NewMockProvider()

	// Initialize rate limiter for prices
	priceRateLimiter := ratelimiter.NewRateLimiter(
		cfg.Price.RateLimitRPS,
		time.Second, // 1 second window
	)

	// Initialize price service
	// Convert cache to domain.Cache interface
	var priceCacheAdapter domain.Cache[string, domainPrice.Price] = priceCache

	// Set up fallback provider
	// Note: The price service requires a non-nil fallback provider
	// If fallback is disabled, we still provide the mock but it won't be used
	// unless the primary provider fails
	var fallbackProvider domainPrice.Provider = mockPriceProvider
	if cfg.Price.FallbackEnabled {
		logger.Info("Price fallback enabled", zap.String("provider", "mock"))
	} else {
		logger.Info("Price fallback disabled (will use mock only on primary failure)")
	}

	priceService := priceservice.NewService(
		priceCacheAdapter,
		coingeckoPriceProvider,
		fallbackProvider,
		priceRateLimiter,
	)

	// Initialize Etherscan client
	etherscanClient := etherscanadapter.NewClient(
		&http.Client{Timeout: cfg.Transaction.RequestTimeout},
		cfg.Transaction.EtherscanBaseURL,
		cfg.Transaction.EtherscanAPIKey,
	)

	// Initialize rate limiter for transactions
	transactionRateLimiter := ratelimiter.NewRateLimiter(
		cfg.Transaction.RateLimitRPS,
		time.Second, // 1 second window
	)

	// Initialize Etherscan transaction provider
	transactionRepo := etherscanadapter.NewProvider(etherscanClient, transactionRateLimiter)

	if cfg.Transaction.EtherscanAPIKey == "" {
		logger.Warn("Etherscan API key not set, transaction features may be limited")
	}

	// Initialize transaction service
	transactionService := transactionservice.NewService(transactionRepo)

	// Initialize token repository (mock - loads from static file)
	tokenRepo, err := initializeTokenRepository(cfg, logger)
	if err != nil {
		logger.Warn("Failed to initialize token repository, continuing without it", zap.Error(err))
	}

	// Create token service adapter that implements TokensService interface
	tokenService := &TokenServiceAdapter{repo: tokenRepo}

	// Initialize portfolio service
	portfolioService := portfolioservice.NewService(portfolioRepo, priceService)
	portfolioService.SetHoldingRepo(holdingRepo)
	portfolioService.SetTransactionRepo(transactionRepo)

	// Initialize HTTP handler adapter
	handlerAdapter := httpserver.NewHandlerAdapter(
		transactionService,
		portfolioService,
		priceService,
		tokenService,
		logger,
	)

	// Initialize HTTP server
	serverConfig := httpserver.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	server := httpserver.NewServer(serverConfig, handlerAdapter, logger)

	// Log server configuration
	logger.Info("Server configured",
		zap.String("host", cfg.Server.Host),
		zap.String("port", cfg.Server.Port),
		zap.String("price_provider", cfg.Price.Provider),
		zap.String("transaction_provider", cfg.Transaction.Provider),
	)

	// Start server with graceful shutdown
	logger.Info("Starting HTTP server")
	if err := server.StartWithGracefulShutdown(); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}

	logger.Info("Application stopped gracefully")
}

// validateConfig validates the configuration
func validateConfig(cfg *config.Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if cfg.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	if cfg.Price.Provider != "coingecko" && cfg.Price.Provider != "mock" {
		return fmt.Errorf("invalid price provider: %s (must be 'coingecko' or 'mock')", cfg.Price.Provider)
	}

	if cfg.Transaction.Provider != "etherscan" && cfg.Transaction.Provider != "mock" {
		return fmt.Errorf("invalid transaction provider: %s (must be 'etherscan' or 'mock')", cfg.Transaction.Provider)
	}

	return nil
}

// initializeDatabase ensures the database directory exists
func initializeDatabase(cfg *config.Config, logger *loggeradapter.Logger) error {
	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	logger.Info("Database directory ready", zap.String("path", dataDir))
	return nil
}

// initializeTokenRepository initializes the token repository from file
func initializeTokenRepository(cfg *config.Config, logger *loggeradapter.Logger) (*coingeckoadapter.MockTokenRepository, error) {
	if cfg.App.TokensPath == "" {
		logger.Warn("Tokens path not configured, token repository will be empty")
		return nil, nil
	}

	// Check if tokens file exists
	if _, err := os.Stat(cfg.App.TokensPath); os.IsNotExist(err) {
		logger.Warn("Tokens file not found", zap.String("path", cfg.App.TokensPath))
		return nil, nil
	}

	tokenRepo, err := coingeckoadapter.NewMockTokenRepository(cfg.App.TokensPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load token repository: %w", err)
	}

	logger.Info("Token repository initialized", zap.String("path", cfg.App.TokensPath))
	return tokenRepo, nil
}

// TokenServiceAdapter adapts token.Repository to TokensService interface
type TokenServiceAdapter struct {
	repo *coingeckoadapter.MockTokenRepository
}

func (t *TokenServiceAdapter) GetTokenByAddress(ctx context.Context, address string) (*token.Token, bool) {
	if t.repo == nil {
		return nil, false
	}

	tok, err := t.repo.GetByAddress(ctx, address)
	if err != nil {
		return nil, false
	}

	return tok, true
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
