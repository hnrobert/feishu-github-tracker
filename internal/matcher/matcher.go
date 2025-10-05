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
func ExpandEvents(repoEvents map[string]interface{}, eventSets map[string]map[string]interface{}, baseEvents map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

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
func MatchEvent(eventType string, action string, ref string, payload map[string]interface{}, configuredEvents map[string]interface{}) bool {
	eventConfig, exists := configuredEvents[eventType]
	if !exists {
		return false
	}

	// If event config is nil or empty, match all
	if eventConfig == nil {
		return true
	}

	configMap, ok := eventConfig.(map[string]interface{})
	if !ok {
		return true
	}

	// Check branches for push and pull_request events
	if eventType == "push" || eventType == "pull_request" {
		if branches, ok := configMap["branches"].([]interface{}); ok {
			if ref != "" && !matchBranches(ref, branches) {
				return false
			}
		}
	}

	// Check types/actions
	if types, ok := configMap["types"].([]interface{}); ok {
		if action != "" && !matchTypes(action, types) {
			return false
		}
	}

	return true
}

func matchBranches(ref string, branches []interface{}) bool {
	// Extract branch name from ref (refs/heads/main -> main)
	branchName := ref
	if strings.HasPrefix(ref, "refs/heads/") {
		branchName = strings.TrimPrefix(ref, "refs/heads/")
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

func matchTypes(action string, types []interface{}) bool {
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
