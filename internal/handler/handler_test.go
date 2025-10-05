package handler

import (
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
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
