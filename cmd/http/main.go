package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"testtask/config"
	httpadapter "testtask/internal/adapters/http/server"
	loggeradapter "testtask/internal/adapters/logger"
	portfolioadapter "testtask/internal/adapters/portfolio"
	priceadapter "testtask/internal/adapters/price"
	tokensadapter "testtask/internal/adapters/tokens"
	portfolioservice "testtask/internal/application/portfolio"
	priceapp "testtask/internal/application/price"
	transactionservice "testtask/internal/application/transaction"

	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	host := flag.String("host", "", "Server host (default: empty, listens on all interfaces)")
	port := flag.String("port", "8080", "Server port")
	dev := flag.Bool("dev", false, "Enable development mode (colored logs)")
	flag.Parse()

	// Initialize structured logger
	logger, err := loggeradapter.NewLogger(*dev)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Load configuration from environment
	appCfg := config.Load()

	// Create server configuration
	cfg := httpadapter.Config{
		Host:         *host,
		Port:         *port,
		ReadTimeout:  appCfg.Server.ReadTimeout,
		WriteTimeout: appCfg.Server.WriteTimeout,
		IdleTimeout:  appCfg.Server.IdleTimeout,
	}

	// Initialize in-memory portfolio repository
	portfolioRepo := portfolioadapter.NewRepository()

	// Initialize price providers
	// Convert CacheTTL from time.Duration to seconds (uint8)
	cacheTTLSeconds := uint8(appCfg.Price.CacheTTL.Seconds())
	if cacheTTLSeconds == 0 {
		cacheTTLSeconds = 60 // Default to 60 seconds if conversion results in 0
	}
	priceCache := priceadapter.NewCache(cacheTTLSeconds)

	// Initialize CoinGecko provider if configured
	var coingeckoProvider *priceadapter.CoinGeckoProvider
	if appCfg.Price.Provider == "coingecko" {
		httpClient := &http.Client{
			Timeout: appCfg.Price.RequestTimeout,
		}
		coingeckoProvider = priceadapter.NewCoinGeckoProvider(httpClient, appCfg.Price.CoinGeckoAPIKey)
	}

	mockPriceProvider := priceadapter.NewMockProvider()

	// Setup price cache service with fallback
	priceCacheService := priceapp.NewCacheService(priceCache)
	if coingeckoProvider != nil && appCfg.Price.FallbackEnabled {
		priceCacheService.SetProviders(coingeckoProvider, mockPriceProvider)
	} else {
		// Use mock as both primary and fallback if CoinGecko not configured
		priceCacheService.SetProviders(mockPriceProvider, mockPriceProvider)
	}
	priceProvider := priceCacheService

	// Initialize portfolio service
	portfolioService := portfolioservice.NewService(portfolioRepo, priceProvider)

	// Initialize tokens repository
	tokensRepo, err := tokensadapter.NewRepository("static/coins.json")
	if err != nil {
		logger.Fatal("Failed to initialize tokens repository", zap.Error(err))
	}

	// Initialize transaction provider
	transactionProvider := transactionservice.NewTransactionProvider()

	// Initialize transaction service
	transactionService := transactionservice.NewService(transactionProvider)

	// Create handler adapter
	handler := httpadapter.NewHandlerAdapter(
		transactionService,
		portfolioService,
		priceProvider,
		tokensRepo,
		logger,
	)

	// Create and start server
	server := httpadapter.NewServer(cfg, handler, logger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		logger.Info("Starting HTTP server", zap.String("host", cfg.Host), zap.String("port", cfg.Port))
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for either an error or a signal
	select {
	case err := <-errChan:
		logger.Fatal("Server error", zap.Error(err))
	case sig := <-sigChan:
		logger.Info("Received signal, starting graceful shutdown", zap.String("signal", sig.String()))
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error during shutdown", zap.Error(err))
		}
		logger.Info("Server shutdown complete")
	}
}
