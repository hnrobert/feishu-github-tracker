package template

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/pkg/logger"
)

// SelectTemplate selects the appropriate template based on event type and tags
func SelectTemplate(eventType string, tags []string, templates config.TemplatesConfig) (map[string]interface{}, error) {
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
func FillTemplate(template map[string]interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	// Deep copy the template
	templateJSON, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(templateJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Replace placeholders recursively
	replacePlaceholders(result, data)

	return result, nil
}

func replacePlaceholders(obj interface{}, data map[string]interface{}) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok {
				v[key] = replacePlaceholder(str, data)
			} else {
				replacePlaceholders(value, data)
			}
		}
	case []interface{}:
		for i, item := range v {
			if str, ok := item.(string); ok {
				v[i] = replacePlaceholder(str, data)
			} else {
				replacePlaceholders(item, data)
			}
		}
	}
}

func replacePlaceholder(str string, data map[string]interface{}) string {
	// Replace {{key}} or {{key.subkey}} placeholders
	result := str
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)

		// Handle nested keys
		if nestedMap, ok := value.(map[string]interface{}); ok {
			for nestedKey, nestedValue := range nestedMap {
				nestedPlaceholder := "{{" + key + "." + nestedKey + "}}"
				nestedValueStr := fmt.Sprintf("%v", nestedValue)
				result = strings.ReplaceAll(result, nestedPlaceholder, nestedValueStr)
			}
		}
	}
	return result
}

// DetermineTags determines which tags to use based on the webhook payload
func DetermineTags(eventType string, payload map[string]interface{}) []string {
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
			if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
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
		if issue, ok := payload["issue"].(map[string]interface{}); ok {
			if labels, ok := issue["labels"].([]interface{}); ok {
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

func getIssueType(labels []interface{}) string {
	for _, label := range labels {
		if labelMap, ok := label.(map[string]interface{}); ok {
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
