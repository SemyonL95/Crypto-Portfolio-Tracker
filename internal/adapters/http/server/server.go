package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	loggeradapter "testtask/internal/adapters/logger"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	echo   *echo.Echo
	config Config
	logger *loggeradapter.Logger
}

// Config holds server configuration
type Config struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewServer creates a new HTTP server with Echo
func NewServer(cfg Config, handler *HandlerAdapter, logger *loggeradapter.Logger) *Server {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Register routes
	registerRoutes(e, handler)

	// Configure server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	if addr == ":" {
		addr = ":8080"
	}

	e.Server.Addr = addr
	e.Server.ReadTimeout = cfg.ReadTimeout
	e.Server.WriteTimeout = cfg.WriteTimeout
	e.Server.IdleTimeout = cfg.IdleTimeout

	return &Server{
		echo:   e,
		config: cfg,
		logger: logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("address", s.echo.Server.Addr))
	return s.echo.Start(s.echo.Server.Addr)
}

// StartWithGracefulShutdown starts the server with graceful shutdown
func (s *Server) StartWithGracefulShutdown() error {
	// Channel to listen for errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("address", s.echo.Server.Addr))
		if err := s.echo.Start(s.echo.Server.Addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	// Channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for either an error or a signal
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		s.logger.Info("Received signal, starting graceful shutdown", zap.String("signal", sig.String()))
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.echo.Shutdown(ctx)
}
