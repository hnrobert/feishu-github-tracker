package matcher

import (
	"fmt"
	"strings"

	"github.com/gobwas/glob"
	"github.com/hnrobert/feishu-github-tracker/internal/config"
)

// MatchRepo finds the first matching repository pattern
func MatchRepo(fullName string, repos []config.RepoPattern) (*config.RepoPattern, error) {
	for i := range repos {
		pattern := repos[i].Pattern
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		if g.Match(fullName) {
			return &repos[i], nil
		}
	}
	return nil, nil
}

// ExpandEvents expands event templates and merges them with custom events
func ExpandEvents(repoEvents map[string]any, eventSets map[string]map[string]any, baseEvents map[string]any) map[string]any {
	result := make(map[string]any)

	// Process each event in the repo configuration
	for key, value := range repoEvents {
		// Check if this is a reference to an event set (template)
		if eventSet, exists := eventSets[key]; exists {
			// It's a template, expand it
			for eventName, eventConfig := range eventSet {
				result[eventName] = eventConfig
			}
		} else if baseEvent, exists := baseEvents[key]; exists {
			// It's a base event from events.yaml
			if value == nil {
				// No customization, use base config
				result[key] = baseEvent
			} else {
				// Has customization, use custom config
				result[key] = value
			}
		} else {
			// Custom event not in base events
			result[key] = value
		}
	}

	return result
}

// MatchEvent checks if the webhook event matches the configured events
func MatchEvent(eventType string, action string, ref string, payload map[string]any, configuredEvents map[string]any) bool {
	eventConfig, exists := configuredEvents[eventType]
	if !exists {
		return false
	}

	// If event config is nil or empty, match all
	if eventConfig == nil {
		return true
	}

	configMap, ok := eventConfig.(map[string]any)
	if !ok {
		return true
	}

	// Check branches for push and pull_request events
	if eventType == "push" || eventType == "pull_request" {
		if branches, ok := configMap["branches"].([]any); ok {
			if ref != "" && !matchBranches(ref, branches) {
				return false
			}
		}
	}

	// Check types/actions
	if types, ok := configMap["types"].([]any); ok {
		if action != "" && !matchTypes(action, types) {
			return false
		}
	}

	return true
}

func matchBranches(ref string, branches []any) bool {
	// Extract branch name from ref (refs/heads/main -> main)
	branchName := ref
	if after, ok :=strings.CutPrefix(ref, "refs/heads/"); ok  {
		branchName = after
	}

	for _, b := range branches {
		pattern, ok := b.(string)
		if !ok {
			continue
		}

		// Handle wildcard
		if pattern == "*" {
			return true
		}

		// Use glob matching
		g, err := glob.Compile(pattern)
		if err != nil {
			continue
		}
		if g.Match(branchName) {
			return true
		}
	}
	return false
}

func matchTypes(action string, types []any) bool {
	for _, t := range types {
		typeStr, ok := t.(string)
		if !ok {
			continue
		}
		if typeStr == action {
			return true
		}
	}
	return false
}
