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
	// First process simple {{#if expr}}...{{/if}} blocks (non-nested)
	s = processIfBlocks(s, data)

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

// processIfBlocks evaluates and expands simple {{#if expr}}...{{/if}} blocks.
// It supports non-nested blocks and will recursively process inner blocks after
// a block is kept. If the condition is falsy, the entire block is removed.
func processIfBlocks(s string, data map[string]any) string {
	// (?s) enables dot to match newline
	reIf := regexp.MustCompile(`(?s)\{\{\s*#if\s+(.+?)\s*\}\}(.*?)\{\{\s*/if\s*\}\}`)
	for {
		loc := reIf.FindStringSubmatchIndex(s)
		if loc == nil {
			break
		}
		// Extract groups
		matches := reIf.FindStringSubmatch(s)
		if len(matches) < 3 {
			break
		}
		condExpr := strings.TrimSpace(matches[1])
		inner := matches[2]

		val, ok := evalExpression(condExpr, data)
		if ok && isTruthy(val) {
			// Keep inner, but process nested ifs inside it
			processedInner := processIfBlocks(inner, data)
			// Also replace placeholders inside the kept inner content
			processedInner = replacePlaceholdersInString(processedInner, data)
			// Replace the whole match with processedInner
			s = s[:loc[0]] + processedInner + s[loc[1]:]
		} else {
			// Remove whole block
			s = s[:loc[0]] + s[loc[1]:]
		}
	}
	return s
}

// isTruthy determines whether a value should be considered true for {{#if}}
func isTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case int, int32, int64, float32, float64:
		// numeric zero considered truthy here (like template languages), but treat 0 as false
		return fmt.Sprintf("%v", t) != "0"
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	default:
		return true
	}
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
// The event type is ALWAYS included as the first tag automatically
func DetermineTags(eventType string, payload map[string]any) []string {
	// Event type is ALWAYS the first tag
	tags := []string{eventType}

	// Extract action if present - this becomes a tag for all events
	if action, ok := payload["action"].(string); ok && action != "" {
		tags = append(tags, action)
	}

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
		}
		// Note: 'action' tag was already added above if present

	case "issues":
		// Action tag already added above

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
				tags = append(tags, status)
				if status == "completed" {
					if concl, ok := wr["conclusion"].(string); ok && concl != "" {
						// common conclusions: success, failure, cancelled
						tags = append(tags, concl)
					}
				}
			}
		}

	case "workflow_job":
		// Similar to workflow_run
		if wj, ok := payload["workflow_job"].(map[string]any); ok {
			if status, ok := wj["status"].(string); ok && status != "" {
				tags = append(tags, status)
			}
			if concl, ok := wj["conclusion"].(string); ok && concl != "" {
				tags = append(tags, concl)
			}
		}

	case "check_run":
		// Mirror workflow_run semantics for check runs: emit completion and conclusion
		if cr, ok := payload["check_run"].(map[string]any); ok {
			if status, ok := cr["status"].(string); ok && status != "" {
				tags = append(tags, status)
				if status == "completed" {
					if concl, ok := cr["conclusion"].(string); ok && concl != "" {
						tags = append(tags, concl)
					}
				}
			}
		}

	case "check_suite":
		// Similar to check_run
		if cs, ok := payload["check_suite"].(map[string]any); ok {
			if status, ok := cs["status"].(string); ok && status != "" {
				tags = append(tags, status)
			}
			if concl, ok := cs["conclusion"].(string); ok && concl != "" {
				tags = append(tags, concl)
			}
		}
	}

	// Add "default" tag if no specific tags were added (other than event type and action)
	if len(tags) <= 2 { // only event type and possibly action
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
