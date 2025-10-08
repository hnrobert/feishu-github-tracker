package template

import (
	"reflect"
	"testing"
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
