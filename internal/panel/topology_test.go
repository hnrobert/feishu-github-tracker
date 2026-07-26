package panel

import (
	"testing"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

func TestTopologyFromConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Repos.Repos = []config.RepoPattern{{
		Pattern:  "acme/api",
		Events:   map[string]any{"basic": nil, "push": nil},
		NotifyTo: []string{"dev-team", "https://example.test/hook"},
	}}
	cfg.Events.EventSets = map[string]map[string]any{"basic": {"push": nil}}

	graph := topologyFromConfig(cfg)
	if len(graph.Nodes) != 5 {
		t.Fatalf("nodes = %d, want 5: %#v", len(graph.Nodes), graph.Nodes)
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("edges = %d, want 4: %#v", len(graph.Edges), graph.Edges)
	}
	if len(graph.Routes) != 1 || graph.Routes[0].EventCount != 2 || graph.Routes[0].TargetCount != 2 {
		t.Fatalf("routes = %#v", graph.Routes)
	}
	if graph.Nodes[0].Href != "/repos/edit?index=0" {
		t.Fatalf("repo href = %q", graph.Nodes[0].Href)
	}
}
