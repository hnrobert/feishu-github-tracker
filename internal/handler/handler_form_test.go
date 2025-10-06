package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
)

func TestServeHTTP_FormEncodedPayload(t *testing.T) {
	// Initialize logger for tests
	logger.Init("info", "/tmp")

	// Create a minimal config and handler
	cfg := &config.Config{
		Server: config.ServerConfig{
			Server: struct {
				Host           string `yaml:"host"`
				Port           int    `yaml:"port"`
				Secret         string `yaml:"secret"`
				LogLevel       string `yaml:"log_level"`
				MaxPayloadSize string `yaml:"max_payload_size"`
				Timeout        int    `yaml:"timeout"`
			}{Secret: ""},
		},
		Repos: config.ReposConfig{
			Repos: []config.RepoPattern{
				{Pattern: "*", NotifyTo: []string{"test"}},
			},
		},
		Events: config.EventsConfig{
			Events: map[string]any{
				"push": map[string]any{"ref": "*"},
			},
		},
		Templates: config.TemplatesConfig{
			Templates: map[string]config.EventTemplate{
				"push": {
					Payloads: []config.PayloadTemplate{
						{
							Tags: []string{"push", "default"},
							Payload: map[string]any{
								"msg_type": "text",
								"content": map[string]any{
									"text": "Test push: {{repository.full_name}}",
								},
							},
						},
					},
				},
			},
		},
	}

	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	// Create a form-encoded payload
	jsonPayload := `{"repository":{"full_name":"test/repo"},"ref":"refs/heads/main","commits":[]}`
	formData := url.Values{}
	formData.Set("payload", jsonPayload)

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-GitHub-Event", "push")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Should succeed (200 OK) even though we don't have a real notifier endpoint
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestServeHTTP_FormEncodedMissingPayload(t *testing.T) {
	// Initialize logger for tests
	logger.Init("info", "/tmp")

	cfg := &config.Config{
		Server: config.ServerConfig{
			Server: struct {
				Host           string `yaml:"host"`
				Port           int    `yaml:"port"`
				Secret         string `yaml:"secret"`
				LogLevel       string `yaml:"log_level"`
				MaxPayloadSize string `yaml:"max_payload_size"`
				Timeout        int    `yaml:"timeout"`
			}{Secret: ""},
		},
	}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	// Create form data without payload field
	formData := url.Values{}
	formData.Set("other", "value")

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-GitHub-Event", "push")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Missing payload field") {
		t.Fatalf("Expected 'Missing payload field' error, got: %s", w.Body.String())
	}
}
