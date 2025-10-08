package main

import (
	"net/http"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

func TestNewServerConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Server.Host = "127.0.0.1"
	cfg.Server.Server.Port = 0
	cfg.Server.Server.Timeout = 5

	h := http.NewServeMux()
	srv := NewServer(cfg, h)
	if srv == nil {
		t.Fatalf("expected server, got nil")
	}
	if srv.ReadTimeout == 0 || srv.WriteTimeout == 0 {
		t.Fatalf("expected non-zero timeouts")
	}
}
