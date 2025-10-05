package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/pkg/logger"
)

// SelectTemplate selects the appropriate template based on event type and tags
func SelectTemplate(eventType string, tags []string, templates config.TemplatesConfig) (map[string]any, error) {
	eventTemplate, exists := templates.Templates[eventType]
	if !exists {
		return nil, fmt.Errorf("no template found for event type: %s", eventType)
	}

	// Find the best matching payload based on tags
	var selectedPayload *config.PayloadTemplate
	maxMatchScore := -1

	for i := range eventTemplate.Payloads {
		payload := &eventTemplate.Payloads[i]
		score := calculateMatchScore(tags, payload.Tags)
		if score > maxMatchScore {
			maxMatchScore = score
			selectedPayload = payload
		}
	}

	if selectedPayload == nil {
		return nil, fmt.Errorf("no matching payload found for event type: %s with tags: %v", eventType, tags)
	}

	return selectedPayload.Payload, nil
}

// calculateMatchScore calculates how well the tags match
func calculateMatchScore(eventTags, templateTags []string) int {
	score := 0
	for _, eventTag := range eventTags {
		for _, templateTag := range templateTags {
			if eventTag == templateTag {
				score++
			}
		}
	}
	return score
}

// FillTemplate fills the template with actual data from the webhook payload
func FillTemplate(template map[string]any, data map[string]any) (map[string]any, error) {
	// Deep copy the template
	templateJSON, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(templateJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Replace placeholders recursively
	replacePlaceholders(result, data)

	return result, nil
}

func replacePlaceholders(obj any, data map[string]any) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			if str, ok := value.(string); ok {
				v[key] = replacePlaceholdersInString(str, data)
			} else {
				replacePlaceholders(value, data)
			}
		}
	case []any:
		for i, item := range v {
			if str, ok := item.(string); ok {
				v[i] = replacePlaceholdersInString(str, data)
			} else {
				replacePlaceholders(item, data)
			}
		}
	}
}

// replacePlaceholdersInString replaces placeholders like {{key}}, {{key.subkey}} and
// {{key | length}} with values from data. Supports dotted paths and the length operator.
func replacePlaceholdersInString(s string, data map[string]any) string {
	// regex: group1 = path (a.b.c), group2 = optional 'length'
	// patterns:
	// {{path}} or {{path | length}} or {{path | default('fallback')}}
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_\.]+)\s*(?:\|\s*(length)\s*|\|\s*default\(\'([^']*)\'\)\s*)?\}\}`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		parts := re.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		path := parts[1]
		// parts[2] is length if present; parts[3] is default value if present
		opLength := parts[2]
		defaultVal := parts[3]

		// resolve dotted path
		val, ok := getValueByPath(path, data)
		if !ok {
			if defaultVal != "" {
				return defaultVal
			}
			return m // leave unchanged if not found and no default
		}

		if opLength == "length" {
			switch t := val.(type) {
			case []any:
				return fmt.Sprintf("%d", len(t))
			case string:
				return fmt.Sprintf("%d", len(t))
			case map[string]any:
				return fmt.Sprintf("%d", len(t))
			default:
				return fmt.Sprintf("%v", val)
			}
		}

		return fmt.Sprintf("%v", val)
	})
}

// getValueByPath resolves dotted paths like 'repository.full_name' from data map.
func getValueByPath(path string, data map[string]any) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = data
	for _, p := range parts {
		if m, ok := cur.(map[string]any); ok {
			if v, exists := m[p]; exists {
				cur = v
				continue
			}
			return nil, false
		}
		return nil, false
	}
	return cur, true
}

// DetermineTags determines which tags to use based on the webhook payload
func DetermineTags(eventType string, payload map[string]any) []string {
	tags := []string{eventType}

	switch eventType {
	case "push":
		if forced, ok := payload["forced"].(bool); ok && forced {
			tags = append(tags, "force")
		} else {
			tags = append(tags, "default")
		}

	case "pull_request":
		action, _ := payload["action"].(string)
		if action == "closed" {
			if pr, ok := payload["pull_request"].(map[string]any); ok {
				if merged, ok := pr["merged"].(bool); ok && merged {
					tags = append(tags, "closed", "merged")
				} else {
					tags = append(tags, "closed", "unmerged")
				}
			}
		} else {
			tags = append(tags, "pr", "default")
		}

	case "issues":
		if issue, ok := payload["issue"].(map[string]any); ok {
			if labels, ok := issue["labels"].([]any); ok {
				issueType := getIssueType(labels)
				tags = append(tags, "issue", "type:"+issueType)
			} else {
				tags = append(tags, "issue", "type:unknown")
			}
		}

	default:
		tags = append(tags, "default")
	}

	logger.Debug("Determined tags for event %s: %v", eventType, tags)
	return tags
}

func getIssueType(labels []any) string {
	for _, label := range labels {
		if labelMap, ok := label.(map[string]any); ok {
			if name, ok := labelMap["name"].(string); ok {
				lowerName := strings.ToLower(name)
				if strings.Contains(lowerName, "bug") {
					return "bug"
				}
				if strings.Contains(lowerName, "feature") {
					return "feature"
				}
				if strings.Contains(lowerName, "task") {
					return "task"
				}
			}
		}
	}
	return "unknown"
}
