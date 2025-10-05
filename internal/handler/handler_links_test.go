package handler

import (
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
)

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
