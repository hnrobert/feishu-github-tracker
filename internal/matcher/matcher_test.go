package matcher

import (
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

func TestMatchRepo(t *testing.T) {
	repos := []config.RepoPattern{
		{Pattern: "org/specific-repo", Events: nil, NotifyTo: nil},
		{Pattern: "org/*", Events: nil, NotifyTo: nil},
		{Pattern: "*", Events: nil, NotifyTo: nil},
	}

	tests := []struct {
		name     string
		fullName string
		wantIdx  int
	}{
		{"exact match", "org/specific-repo", 0},
		{"wildcard match", "org/another-repo", 1},
		{"catch all", "other/repo", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := MatchRepo(tt.fullName, repos)
			if err != nil {
				t.Fatalf("MatchRepo() error = %v", err)
			}
			if matched == nil {
				t.Fatal("Expected match, got nil")
			}
			if matched.Pattern != repos[tt.wantIdx].Pattern {
				t.Errorf("Expected pattern %s, got %s", repos[tt.wantIdx].Pattern, matched.Pattern)
			}
		})
	}
}

func TestMatchEvent(t *testing.T) {
	tests := []struct {
		name             string
		eventType        string
		action           string
		ref              string
		configuredEvents map[string]interface{}
		want             bool
	}{
		{
			name:             "match push with wildcard branch",
			eventType:        "push",
			action:           "",
			ref:              "refs/heads/main",
			configuredEvents: map[string]interface{}{"push": map[string]interface{}{"branches": []interface{}{"*"}}},
			want:             true,
		},
		{
			name:             "match push with specific branch",
			eventType:        "push",
			action:           "",
			ref:              "refs/heads/main",
			configuredEvents: map[string]interface{}{"push": map[string]interface{}{"branches": []interface{}{"main"}}},
			want:             true,
		},
		{
			name:             "no match push with wrong branch",
			eventType:        "push",
			action:           "",
			ref:              "refs/heads/develop",
			configuredEvents: map[string]interface{}{"push": map[string]interface{}{"branches": []interface{}{"main"}}},
			want:             false,
		},
		{
			name:             "match pull_request with correct type",
			eventType:        "pull_request",
			action:           "opened",
			ref:              "",
			configuredEvents: map[string]interface{}{"pull_request": map[string]interface{}{"types": []interface{}{"opened", "closed"}}},
			want:             true,
		},
		{
			name:             "no match pull_request with wrong type",
			eventType:        "pull_request",
			action:           "labeled",
			ref:              "",
			configuredEvents: map[string]interface{}{"pull_request": map[string]interface{}{"types": []interface{}{"opened", "closed"}}},
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchEvent(tt.eventType, tt.action, tt.ref, nil, tt.configuredEvents)
			if got != tt.want {
				t.Errorf("MatchEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
