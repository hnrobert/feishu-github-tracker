package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/handler"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
)

func main() {
	// Parse command line flags
	enableReload := flag.Bool("reload", false, "Enable configuration hot reload on each webhook request")
	flag.Parse()

	// Determine config directory
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		// Default to ./config relative to executable
		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
			os.Exit(1)
		}
		configDir = filepath.Join(filepath.Dir(execPath), "configs")
	}

	// Load configuration
	cfg, err := config.Load(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Determine log directory
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
			os.Exit(1)
		}
		logDir = filepath.Join(filepath.Dir(execPath), "logs")
	}

	// Initialize logger
	if err := logger.Init(cfg.Server.Server.LogLevel, logDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting GitHub to Feishu webhook forwarder")
	logger.Info("Config directory: %s", configDir)
	logger.Info("Log directory: %s", logDir)
	logger.Info("Hot reload enabled: %v", *enableReload)

	// Create notifier
	n := notifier.New(cfg.FeishuBots)

	// Create handler with hot reload support
	h := handler.New(cfg, n)
	if *enableReload {
		h.EnableHotReload(configDir)
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/webhook", h)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := NewServer(cfg, mux)

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}

// NewServer creates an *http.Server configured from cfg and handler.
func NewServer(cfg *config.Config, handler http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Server.Host, cfg.Server.Server.Port)
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.Server.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.Server.Timeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
