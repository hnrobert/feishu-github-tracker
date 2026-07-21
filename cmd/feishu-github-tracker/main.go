package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/fs"
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
	"github.com/hnrobert/feishu-github-tracker/internal/panel"
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

	if defaultConfigDir := os.Getenv("DEFAULT_CONFIG_DIR"); defaultConfigDir != "" {
		if err := initializeConfigDir(defaultConfigDir, configDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize configuration: %v\n", err)
			os.Exit(1)
		}
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

	// Normalize the panel password once at startup: if server.yaml has a
	// plaintext panel.password, convert it to password_hash and drop the
	// plaintext line. Also run on each hot-reload so manual edits are converted.
	if changed, err := panel.NormalizePanelPassword(configDir); err != nil {
		logger.Warn("Panel password normalization failed: %v", err)
	} else if changed {
		logger.Info("Converted panel plaintext password to password_hash")
	}
	h.OnReload = func(dir string) {
		if changed, err := panel.NormalizePanelPassword(dir); err != nil {
			logger.Warn("Panel password normalization failed: %v", err)
		} else if changed {
			logger.Info("Converted panel plaintext password to password_hash")
		}
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/webhook", h)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Mount the web management panel at "/" (ServeMux longest-prefix matching
	// keeps /webhook and /health routed to their handlers above). The panel
	// resolves admin username/password from server.yaml + env on each login.
	panelApp, err := panel.New(panel.Options{
		ConfigDir: configDir,
		LogDir:    logDir,
		JWTSecret: resolvePanelSecret(cfg),
		OnSave:    h.Reload, // reload running config after any panel edit
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize panel: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Management panel mounted at / (default login: admin / admin)")
	mux.Handle("/", panelApp)

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

// initializeConfigDir copies default configuration files that do not yet exist.
// Existing files are never overwritten so user configuration remains intact.
func initializeConfigDir(defaultConfigDir, configDir string) error {
	sourceInfo, err := os.Stat(defaultConfigDir)
	if err != nil {
		return fmt.Errorf("read default configuration directory: %w", err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("default configuration path %q is not a directory", defaultConfigDir)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}

	return filepath.WalkDir(defaultConfigDir, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(defaultConfigDir, sourcePath)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}

		targetPath := filepath.Join(configDir, relativePath)
		if entry.IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("create configuration subdirectory %q: %w", relativePath, err)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("default configuration entry %q is not a regular file", relativePath)
		}

		if _, err := os.Lstat(targetPath); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect configuration file %q: %w", relativePath, err)
		}

		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("open default configuration file %q: %w", relativePath, err)
		}

		info, err := entry.Info()
		if err != nil {
			_ = sourceFile.Close()
			return fmt.Errorf("read default configuration file metadata %q: %w", relativePath, err)
		}
		targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			_ = sourceFile.Close()
			return fmt.Errorf("create configuration file %q: %w", relativePath, err)
		}

		_, copyErr := io.Copy(targetFile, sourceFile)
		sourceCloseErr := sourceFile.Close()
		closeErr := targetFile.Close()
		if copyErr != nil {
			_ = os.Remove(targetPath)
			return fmt.Errorf("copy default configuration file %q: %w", relativePath, copyErr)
		}
		if sourceCloseErr != nil {
			_ = os.Remove(targetPath)
			return fmt.Errorf("close default configuration file %q: %w", relativePath, sourceCloseErr)
		}
		if closeErr != nil {
			_ = os.Remove(targetPath)
			return fmt.Errorf("close configuration file %q: %w", relativePath, closeErr)
		}
		return nil
	})
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

// resolvePanelSecret derives the panel JWT signing secret. Precedence:
// PANEL_JWT_SECRET env > server.yaml panel.secret > nil (the panel then uses
// an ephemeral random secret, which logs everyone out on restart).
//
// Admin username/password are resolved by the panel itself on each login from
// the same sources, so they don't need to be passed at startup.
func resolvePanelSecret(cfg *config.Config) []byte {
	secretText := os.Getenv("PANEL_JWT_SECRET")
	if secretText == "" {
		secretText = cfg.Server.Panel.Secret
	}
	if secretText == "" {
		return nil
	}
	var secret []byte
	if decoded, err := base64.RawURLEncoding.DecodeString(secretText); err == nil {
		secret = decoded
	} else {
		secret = []byte(secretText)
	}
	if len(secret) < 16 {
		pad := make([]byte, 16)
		copy(pad, secret)
		secret = pad
	}
	return secret
}
