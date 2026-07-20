package template

import (
	"reflect"
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

func TestFillTemplate_NestedPlaceholder(t *testing.T) {
	tmpl := map[string]any{
		"content": "Repo: {{repository.full_name}} by {{sender.login}}",
	}

	data := map[string]any{
		"repository": map[string]any{"full_name": "org/repo"},
		"sender":     map[string]any{"login": "alice"},
	}

	got, err := FillTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("FillTemplate returned error: %v", err)
	}

	if content, ok := got["content"].(string); !ok || content != "Repo: org/repo by alice" {
		t.Fatalf("unexpected filled content: %v", got)
	}

	// ensure original template not mutated
	if reflect.DeepEqual(tmpl, got) {
		t.Fatalf("expected different map after fill")
	}
}

// issueTemplates builds a small issue template set mirroring the real
// templates.jsonc ordering (bug-specific payloads first, then generic ones).
func issueTemplates() config.TemplatesConfig {
	mk := func(tags []string, title string) config.PayloadTemplate {
		return config.PayloadTemplate{
			Tags: tags,
			Payload: map[string]any{
				"msg_type": "text",
				"content":  map[string]any{"text": title},
			},
		}
	}
	return config.TemplatesConfig{
		Templates: map[string]config.EventTemplate{
			"issues": {Payloads: []config.PayloadTemplate{
				mk([]string{"opened", "type:bug"}, "BUG-OPENED"),
				mk([]string{"opened", "type:feature"}, "FEATURE-OPENED"),
				mk([]string{"opened"}, "ISSUE-OPENED"),
				mk([]string{"closed", "type:bug"}, "BUG-CLOSED"),
				mk([]string{"closed"}, "ISSUE-CLOSED"),
				mk([]string{"type:unknown"}, "ISSUE-UNKNOWN"),
				mk([]string{"default"}, "ISSUE-DEFAULT"),
			}},
		},
	}
}

func titleOf(t map[string]any) string {
	if c, ok := t["content"].(map[string]any); ok {
		if s, ok := c["text"].(string); ok {
			return s
		}
	}
	return ""
}

func TestSelectTemplate_PlainIssueNotBug(t *testing.T) {
	tt := issueTemplates()
	// A normal opened issue with no bug/feature/task label.
	got, err := SelectTemplate("issues", []string{"issues", "opened", "type:unknown"}, tt)
	if err != nil {
		t.Fatalf("SelectTemplate error: %v", err)
	}
	if titleOf(got) == "BUG-OPENED" {
		t.Fatalf("plain issue must not use the bug card; got BUG-OPENED")
	}
}

func TestSelectTemplate_BugIssueUsesBugCard(t *testing.T) {
	tt := issueTemplates()
	got, err := SelectTemplate("issues", []string{"issues", "opened", "type:bug"}, tt)
	if err != nil {
		t.Fatalf("SelectTemplate error: %v", err)
	}
	if titleOf(got) != "BUG-OPENED" {
		t.Fatalf("bug issue should use the bug card; got %q", titleOf(got))
	}
}

func TestSelectTemplate_MostSpecificWins(t *testing.T) {
	tt := issueTemplates()
	// opened + type:bug: the 2-tag payload beats the 1-tag "opened" payload.
	got, err := SelectTemplate("issues", []string{"issues", "opened", "type:bug"}, tt)
	if err != nil {
		t.Fatalf("SelectTemplate error: %v", err)
	}
	if titleOf(got) != "BUG-OPENED" {
		t.Fatalf("most specific (opened+type:bug) should win; got %q", titleOf(got))
	}
}

func TestSelectTemplate_ClosedBugBeatsGenericClosed(t *testing.T) {
	tt := issueTemplates()
	got, err := SelectTemplate("issues", []string{"issues", "closed", "type:bug"}, tt)
	if err != nil {
		t.Fatalf("SelectTemplate error: %v", err)
	}
	if titleOf(got) != "BUG-CLOSED" {
		t.Fatalf("closed bug should use BUG-CLOSED; got %q", titleOf(got))
	}
}
