package panel

import (
	"fmt"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// Topology is a payload-free projection of the notification configuration.
// It deliberately preserves YAML as the source of truth; it never writes it.
type Topology struct {
	Nodes  []TopologyNode
	Edges  []TopologyEdge
	Routes []TopologyRoute
}

// TopologyRoute is the high-signal summary used by the dashboard preview.
// It keeps the full graph available without making the overview repeat every
// event and target node at the same visual weight.
type TopologyRoute struct {
	Label       string
	Href        string
	EventCount  int
	TargetCount int
}

type TopologyNode struct {
	ID    string
	Kind  string
	Label string
	Href  string
}

type TopologyEdge struct {
	From string
	To   string
}

func topologyFromConfig(cfg *config.Config) Topology {
	if cfg == nil {
		return Topology{}
	}
	topology := Topology{}
	nodes := map[string]bool{}
	addNode := func(id, kind, label, href string) {
		if !nodes[id] {
			nodes[id] = true
			topology.Nodes = append(topology.Nodes, TopologyNode{ID: id, Kind: kind, Label: label, Href: href})
		}
	}
	addEdge := func(from, to string) { topology.Edges = append(topology.Edges, TopologyEdge{From: from, To: to}) }

	for index, rule := range cfg.Repos.Repos {
		ruleID := fmt.Sprintf("repo-%d", index)
		addNode(ruleID, "repo", rule.Pattern, fmt.Sprintf("/repos/edit?index=%d", index))
		route := TopologyRoute{Label: rule.Pattern, Href: fmt.Sprintf("/repos/edit?index=%d", index)}
		for event := range rule.Events {
			eventID := "event-" + event
			kind := "event"
			if _, ok := cfg.Events.EventSets[event]; ok {
				kind = "set"
			}
			addNode(eventID, kind, event, "/events")
			addEdge(ruleID, eventID)
			route.EventCount++
		}
		for _, target := range rule.NotifyTo {
			id := "bot-" + target
			href := "/bots"
			kind := "bot"
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				id = fmt.Sprintf("target-%d-%d", index, len(topology.Edges))
				kind = "url"
			}
			addNode(id, kind, target, href)
			addEdge(ruleID, id)
			route.TargetCount++
		}
		topology.Routes = append(topology.Routes, route)
	}
	return topology
}
