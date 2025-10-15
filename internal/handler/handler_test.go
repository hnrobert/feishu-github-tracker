package handler

import (
	"testing"

	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
)

func TestPrepareTemplateData_IncludesNestedObjects(t *testing.T) {
	// Minimal config and notifier stub
	cfg := &config.Config{}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	payload := map[string]any{
		"repository": map[string]any{"full_name": "org/repo", "html_url": "https://github.com/org/repo"},
		"sender":     map[string]any{"login": "alice", "html_url": "https://github.com/alice"},
	}

	data := h.prepareTemplateData("push", payload)

	if _, ok := data["repository"]; !ok {
		t.Fatalf("expected repository nested object in data")
	}
	if _, ok := data["sender"]; !ok {
		t.Fatalf("expected sender nested object in data")
	}
}

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
		Templates: map[string]config.TemplatesConfig{
			"default": {
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

func TestPrepareTemplateData_PushLinks(t *testing.T) {
	cfg := &config.Config{}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	payload := map[string]any{
		"repository": map[string]any{"full_name": "org/repo", "html_url": "https://github.com/org/repo"},
		"pusher":     map[string]any{"name": "bob"},
		"ref":        "refs/heads/main",
		"commits":    []any{},
	}

	data := h.prepareTemplateData("push", payload)

	if _, ok := data["repository_link_md"]; !ok {
		t.Fatalf("expected repository_link_md in prepared data")
	}
	if _, ok := data["branch_link_md"]; !ok {
		t.Fatalf("expected branch_link_md in prepared data")
	}
}

func TestPrepareTemplateData_IssueLinks(t *testing.T) {
	cfg := &config.Config{}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	payload := map[string]any{
		"issue":  map[string]any{"number": 2, "title": "Issue title", "html_url": "https://github.com/org/repo/issues/2", "user": map[string]any{"login": "hnrobert", "html_url": "https://github.com/hnrobert"}},
		"sender": map[string]any{"login": "hnrobert", "html_url": "https://github.com/hnrobert"},
	}

	data := h.prepareTemplateData("issues", payload)

	if v, ok := data["issue_link_md"]; !ok {
		t.Fatalf("expected issue_link_md in prepared data")
	} else {
		if s, ok := v.(string); !ok || s == "" {
			t.Fatalf("issue_link_md should be a non-empty string")
		}
	}
	if _, ok := data["issue_user_link_md"]; !ok {
		t.Fatalf("expected issue_user_link_md in prepared data")
	}
}

func TestPrepareTemplateData_PackageURL(t *testing.T) {
	cfg := &config.Config{}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	payload := map[string]any{
		"action": "published",
		"package": map[string]any{
			"name":         "feishu-github-tracker",
			"package_type": "CONTAINER",
		},
		"repository": map[string]any{"full_name": "hnrobert/feishu-github-tracker"},
	}

	data := h.prepareTemplateData("package", payload)

	v, ok := data["package_link_md"]
	if !ok {
		t.Fatalf("expected package_link_md in prepared data")
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("package_link_md should be a string")
	}
	want := "[feishu-github-tracker](https://github.com/hnrobert/feishu-github-tracker/pkgs/container/feishu-github-tracker)"
	if s != want {
		t.Fatalf("package_link_md mismatch: got %q want %q", s, want)
	}
}
