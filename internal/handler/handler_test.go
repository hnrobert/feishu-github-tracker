package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"net/url"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
	"github.com/hnrobert/feishu-github-tracker/internal/template"
)

func TestPushTemplates(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "example-configs"))
	if err != nil {
		t.Fatal(err)
	}
	h := New(cfg, notifier.New(config.FeishuBotsConfig{}))
	tests := []struct {
		name, ref, kind, title, titleCN string
		deleted, forced                 bool
	}{
		{"branch", "refs/heads/feature/test", "branch", "Deleted:", "删除：", true, false},
		{"tag", "refs/tags/v1/test", "tag", "Deleted:", "删除：", true, false},
		{"normal empty push", "refs/heads/main", "", "New Push", "新推送", false, false},
		{"force", "refs/heads/main", "", "New Push", "新推送", false, true},
		{"deleted and forced", "refs/heads/feature/test", "branch", "Deleted:", "删除：", true, true},
	}
	for _, tt := range tests {
		for _, locale := range []string{"default", "cn"} {
			t.Run(tt.name+"/"+locale, func(t *testing.T) {
				payload := map[string]any{
					"ref": tt.ref, "deleted": tt.deleted, "forced": tt.forced, "commits": []any{},
					"compare":    "https://example.com/compare",
					"repository": map[string]any{"full_name": "org/repo", "html_url": "https://github.com/org/repo"},
					"sender":     map[string]any{"login": "alice", "html_url": "https://github.com/alice"},
					"pusher":     map[string]any{"name": "alice"},
				}
				data := h.prepareTemplateData("push", payload)
				selected, err := template.SelectTemplate("push", template.DetermineTags("push", payload), cfg.GetTemplateConfig(locale))
				if err != nil {
					t.Fatal(err)
				}
				card, err := template.FillTemplate(selected, data)
				if err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(card)
				if err != nil {
					t.Fatal(err)
				}
				got := string(encoded)
				title := tt.title
				if locale == "cn" {
					title = tt.titleCN
				}
				if !strings.Contains(got, title) {
					t.Fatalf("card missing title %q: %s", title, got)
				}
				if tt.deleted {
					for _, want := range []string{"alice", "org/repo", tt.kind, tt.ref} {
						if !strings.Contains(got, want) {
							t.Errorf("card missing %q: %s", want, got)
						}
					}
					for _, unwanted := range []string{"commits", "提交", "compare", "/tree/", `"tag":"action"`} {
						if strings.Contains(got, unwanted) {
							t.Errorf("deletion card contains %q: %s", unwanted, got)
						}
					}
				}
			})
		}
	}
}

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

// TestPrepareDiscussionData exposes the raw discussion object (including the
// nested category) so the shipped card templates' {{discussion.title}} and
// {{discussion.category.name}} placeholders resolve instead of rendering
// literally.
func TestPrepareDiscussionData(t *testing.T) {
	cfg := &config.Config{}
	h := New(cfg, notifier.New(config.FeishuBotsConfig{}))

	payload := map[string]any{
		"action": "edited",
		"discussion": map[string]any{
			"title":    "Week 3 lab",
			"html_url": "https://github.com/org/repo/discussions/1",
			"category": map[string]any{"name": "Q&A"},
		},
	}

	data := h.prepareTemplateData("discussion", payload)

	raw, ok := data["discussion"].(map[string]any)
	if !ok {
		t.Fatal("expected raw discussion object in data")
	}
	if raw["title"] != "Week 3 lab" {
		t.Errorf("discussion.title = %v", raw["title"])
	}
	if cat, ok := raw["category"].(map[string]any); !ok || cat["name"] != "Q&A" {
		t.Errorf("discussion.category.name missing or wrong: %#v", raw["category"])
	}
	if data["discussion_category"] != "Q&A" {
		t.Errorf("flat discussion_category = %v", data["discussion_category"])
	}
	if data["discussion_title"] != "Week 3 lab" {
		t.Errorf("flat discussion_title = %v", data["discussion_title"])
	}
}

func TestProcessWebhookMatchAllRules(t *testing.T) {
	logger.Init("error", os.TempDir())
	defer logger.Close()
	received := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received[r.URL.Path]++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	newHandler := func(matchAllRules bool) *Handler {
		cfg := &config.Config{
			Repos: config.ReposConfig{Repos: []config.RepoPattern{
				{Pattern: "org/repo", Events: map[string]any{"release": nil}, NotifyTo: []string{"sl"}},
				{Pattern: "org/repo", Events: map[string]any{"all": nil}, NotifyTo: []string{"action", "shared"}},
				{Pattern: "org/repo", Events: map[string]any{"reviewer": nil}, NotifyTo: []string{"reviewer", "shared"}},
			}},
			Events: config.EventsConfig{EventSets: map[string]map[string]any{
				"all":      {"issue_comment": nil},
				"reviewer": {"issue_comment": nil},
			}},
			FeishuBots: config.FeishuBotsConfig{FeishuBots: []config.FeishuBot{
				{Alias: "sl", URL: server.URL + "/sl"},
				{Alias: "action", URL: server.URL + "/action"},
				{Alias: "reviewer", URL: server.URL + "/reviewer"},
				{Alias: "shared", URL: server.URL + "/shared"},
			}},
			Templates: map[string]config.TemplatesConfig{"default": {
				Templates: map[string]config.EventTemplate{"issue_comment": {
					Payloads: []config.PayloadTemplate{{Tags: []string{"default"}, Payload: map[string]any{"msg_type": "text"}}},
				}},
			}},
		}
		cfg.Server.Server.MatchAllRules = matchAllRules
		return New(cfg, notifier.New(cfg.FeishuBots))
	}

	payload := map[string]any{
		"action":     "edited",
		"repository": map[string]any{"full_name": "org/repo"},
	}
	firstMatch := newHandler(false)
	if err := firstMatch.processWebhook("issue_comment", payload, firstMatch.snapshot()); err != nil {
		t.Fatalf("default matching returned error: %v", err)
	}
	if len(received) != 0 {
		t.Fatalf("default matching delivered %#v, want no messages", received)
	}

	matchAll := newHandler(true)
	if err := matchAll.processWebhook("issue_comment", payload, matchAll.snapshot()); err != nil {
		t.Fatalf("match_all_rules returned error: %v", err)
	}
	if received["/action"] != 1 || received["/reviewer"] != 1 || received["/shared"] != 1 || received["/sl"] != 0 {
		t.Fatalf("match_all_rules delivered %#v, want action, reviewer, and shared once each", received)
	}
}

func TestServeHTTP_FormEncodedPayload(t *testing.T) {
	// Initialize logger for tests
	logger.Init("info", "/tmp")
	defer logger.Close()

	// Create a minimal config and handler
	cfg := &config.Config{
		Server: config.ServerConfig{
			Server: struct {
				Host           string `yaml:"host"`
				Port           int    `yaml:"port"`
				Secret         string `yaml:"secret"`
				LogLevel       string `yaml:"log_level"`
				MatchAllRules  bool   `yaml:"match_all_rules"`
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
	defer logger.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Server: struct {
				Host           string `yaml:"host"`
				Port           int    `yaml:"port"`
				Secret         string `yaml:"secret"`
				LogLevel       string `yaml:"log_level"`
				MatchAllRules  bool   `yaml:"match_all_rules"`
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

func TestServeHTTP_RejectsOversizedPayload(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Server.MaxPayloadSize = "1KB"
	h := New(cfg, notifier.New(config.FeishuBotsConfig{}))
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(strings.Repeat("x", 2048)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized payload status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestParseMaxPayloadBytes(t *testing.T) {
	tests := map[string]int64{
		"1KB":  1 << 10,
		"5 MB": 5 << 20,
		"2MiB": 2 << 20,
		"4096": 4096,
		"bad":  defaultMaxPayloadBytes,
	}
	for input, want := range tests {
		if got := parseMaxPayloadBytes(input); got != want {
			t.Errorf("parseMaxPayloadBytes(%q) = %d, want %d", input, got, want)
		}
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

func TestPrepareTemplateData_ActionAndIssueCommentLink(t *testing.T) {
	cfg := &config.Config{}
	n := notifier.New(config.FeishuBotsConfig{})
	h := New(cfg, n)

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number":   2,
			"title":    "Issue title",
			"html_url": "https://github.com/org/repo/issues/2",
		},
	}

	for _, eventType := range []string{"team_add", "security_and_analysis", "pull_request_review", "issue_comment"} {
		data := h.prepareTemplateData(eventType, payload)
		if got := data["action"]; got != "created" {
			t.Errorf("%s action = %v, want created", eventType, got)
		}
	}

	data := h.prepareTemplateData("issue_comment", payload)
	if got := data["issue_link_md"]; got != "[#2 Issue title](https://github.com/org/repo/issues/2)" {
		t.Fatalf("issue_link_md = %v", got)
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
