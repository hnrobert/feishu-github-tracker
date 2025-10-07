package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
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
	re := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		parts := re.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		expr := parts[1]
		val, ok := evalExpression(expr, data)
		if !ok {
			return m
		}
		return fmt.Sprintf("%v", val)
	})
}

// evalExpression evaluates an expression like:
// path | length | default(other | default('fallback'))
// Supports nested default(...) where the default argument can be a quoted literal
// or another expression.
func evalExpression(expr string, data map[string]any) (any, bool) {
	tokens := splitTopLevelPipes(expr)
	if len(tokens) == 0 {
		return nil, false
	}
	// first token is a path
	first := strings.TrimSpace(tokens[0])
	cur, ok := getValueByPath(first, data)

	// process filters
	for i := 1; i < len(tokens); i++ {
		filter := strings.TrimSpace(tokens[i])
		if filter == "length" {
			// compute length of current value regardless of ok
			switch t := cur.(type) {
			case []any:
				cur = len(t)
				ok = true
			case string:
				cur = len(t)
				ok = true
			case map[string]any:
				cur = len(t)
				ok = true
			default:
				// can't compute length
				cur = fmt.Sprintf("%v", cur)
				ok = true
			}
			continue
		}

		if strings.HasPrefix(filter, "default(") && strings.HasSuffix(filter, ")") {
			// default arg inside parentheses
			inner := strings.TrimSpace(filter[len("default(") : len(filter)-1])
			// if current exists and is non-empty string, keep it
			if ok {
				if s, isStr := cur.(string); isStr && s == "" {
					// treat empty string as missing, fall through to default
				} else {
					// current present; skip default
					continue
				}
			}

			// evaluate inner: if quoted literal, return that; else evaluate as expression
			if (strings.HasPrefix(inner, "'") && strings.HasSuffix(inner, "'")) || (strings.HasPrefix(inner, "\"") && strings.HasSuffix(inner, "\"")) {
				// strip quotes
				lit := inner[1 : len(inner)-1]
				cur = lit
				ok = true
			} else {
				// inner might itself be an expression (contain pipes)
				v, vok := evalExpression(inner, data)
				if vok {
					cur = v
					ok = true
				} else {
					ok = false
				}
			}
			continue
		}

		// unknown filter: ignore
	}

	return cur, ok
}

// splitTopLevelPipes splits an expression into tokens separated by top-level | characters
// i.e. it ignores | characters inside parentheses or quotes.
func splitTopLevelPipes(s string) []string {
	var res []string
	var cur strings.Builder
	depth := 0
	inSingle := false
	inDouble := false
	for _, r := range s {
		switch r {
		case '|':
			if depth == 0 && !inSingle && !inDouble {
				res = append(res, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteRune(r)
		case '(':
			if !inSingle && !inDouble {
				depth++
			}
			cur.WriteRune(r)
		case ')':
			if !inSingle && !inDouble && depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			cur.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	// trim tokens
	for i := range res {
		res[i] = strings.TrimSpace(res[i])
	}
	return res
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
			tags = append(tags, "default")
		}

	case "issues":
		// include the action name as a tag so templates can match specific actions
		if action, ok := payload["action"].(string); ok && action != "" {
			tags = append(tags, action)
		}

		// Try to infer issue type from explicit `type` object (provider-specific)
		issueTypeName := ""
		if issue, ok := payload["issue"].(map[string]any); ok {
			if tmap, ok2 := issue["type"].(map[string]any); ok2 {
				if name, ok3 := tmap["name"].(string); ok3 {
					issueTypeName = name
				}
			}
		}
		// also check top-level payload.type (some providers include it there)
		if issueTypeName == "" {
			if tmap, ok := payload["type"].(map[string]any); ok {
				if name, ok2 := tmap["name"].(string); ok2 {
					issueTypeName = name
				}
			}
		}

		if issueTypeName != "" {
			// normalize common known values
			lower := strings.ToLower(issueTypeName)
			if strings.Contains(lower, "bug") {
				lower = "bug"
			} else if strings.Contains(lower, "feature") {
				lower = "feature"
			} else if strings.Contains(lower, "task") {
				lower = "task"
			}
			tags = append(tags, "type:"+lower)
		} else {
			// fallback to label-based detection
			if issue, ok := payload["issue"].(map[string]any); ok {
				if labels, ok := issue["labels"].([]any); ok {
					issueType := getIssueType(labels)
					tags = append(tags, "type:"+issueType)
					// also expose each label as a tag: label:<name> (sanitized)
					reSan := regexp.MustCompile(`[^a-z0-9_-]`)
					for _, l := range labels {
						if lm, lok := l.(map[string]any); lok {
							if lname, lok2 := lm["name"].(string); lok2 && lname != "" {
								tn := strings.ToLower(lname)
								tn = strings.ReplaceAll(tn, " ", "_")
								tn = reSan.ReplaceAllString(tn, "")
								tags = append(tags, "label:"+tn)
							}
						}
					}
				} else {
					tags = append(tags, "type:unknown")
				}
			} else {
				tags = append(tags, "type:unknown")
			}
		}

		// if this is a labeled/unlabeled action and the payload contains a single label
		if lab, lok := payload["label"].(map[string]any); lok {
			if lname, lok2 := lab["name"].(string); lok2 && lname != "" {
				tn := strings.ToLower(lname)
				tn = strings.ReplaceAll(tn, " ", "_")
				reSan := regexp.MustCompile(`[^a-z0-9_-]`)
				tn = reSan.ReplaceAllString(tn, "")
				// add both generic labeled tag and label-specific tag
				tags = append(tags, "label:"+tn)
				if action, ok := payload["action"].(string); ok && action == "labeled" {
					tags = append(tags, "labeled", "labeled:"+tn)
				}
			}
		}

	case "workflow_run":
		// For workflow runs, emit tags that describe completion and outcome so
		// templates can select success/failure-specific payloads.
		if wr, ok := payload["workflow_run"].(map[string]any); ok {
			if status, ok := wr["status"].(string); ok && status != "" {
				if status == "completed" {
					tags = append(tags, "completed")
					if concl, ok := wr["conclusion"].(string); ok && concl != "" {
						// common conclusions: success, failure, cancelled
						// only append success/failure so templates can match them; otherwise fall back
						if concl == "success" || concl == "failure" {
							tags = append(tags, concl)
						} else {
							tags = append(tags, "default")
						}
					} else {
						tags = append(tags, "default")
					}
				} else {
					// non-completed statuses (in_progress, queued, etc.)
					tags = append(tags, status)
				}
			} else {
				tags = append(tags, "default")
			}
		} else {
			tags = append(tags, "default")
		}

	case "check_run":
		// Mirror workflow_run semantics for check runs: emit completion and conclusion
		if cr, ok := payload["check_run"].(map[string]any); ok {
			if status, ok := cr["status"].(string); ok && status != "" {
				if status == "completed" {
					tags = append(tags, "completed")
					if concl, ok := cr["conclusion"].(string); ok && concl != "" {
						if concl == "success" || concl == "failure" {
							tags = append(tags, concl)
						} else {
							tags = append(tags, "default")
						}
					} else {
						tags = append(tags, "default")
					}
				} else {
					// non-completed statuses
					tags = append(tags, status)
				}
			} else {
				tags = append(tags, "default")
			}
		} else {
			tags = append(tags, "default")
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
