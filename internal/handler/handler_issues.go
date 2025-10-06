package handler

import (
	"fmt"
	"strings"
)

func prepareIssuesData(data map[string]any, payload map[string]any) {
	if issue, ok := payload["issue"].(map[string]any); ok {
		data["issue_number"] = issue["number"]
		data["issue_title"] = issue["title"]
		data["issue_url"] = issue["html_url"]
		data["issue_state"] = issue["state"]
		data["issue_body"] = issue["body"]
		data["issue"] = issue

		if user, ok := issue["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["issue_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
				}
			}
		}

		if iu, ok := issue["html_url"].(string); ok {
			if it, ok2 := issue["title"].(string); ok2 && it != "" {
				data["issue_link_md"] = fmt.Sprintf("[#%v %s](%s)", issue["number"], it, iu)
			} else {
				data["issue_link_md"] = iu
			}
		}

		issueTypeName := ""
		if tmap, ok := issue["type"].(map[string]any); ok {
			if name, ok2 := tmap["name"].(string); ok2 {
				issueTypeName = name
			}
		}
		if issueTypeName == "" {
			if tmap, ok := payload["type"].(map[string]any); ok {
				if name, ok2 := tmap["name"].(string); ok2 {
					issueTypeName = name
				}
			}
		}

		issueTypeNormalized := "unknown"
		if issueTypeName != "" {
			lower := strings.ToLower(issueTypeName)
			if strings.Contains(lower, "bug") {
				issueTypeNormalized = "bug"
			} else if strings.Contains(lower, "feature") {
				issueTypeNormalized = "feature"
			} else if strings.Contains(lower, "task") {
				issueTypeNormalized = "task"
			} else {
				issueTypeNormalized = lower
			}
		} else {
			if labels, ok := issue["labels"].([]any); ok {
				issueTypeNormalized = detectIssueTypeFromLabels(labels)
			}
		}

		data["issue_type_name"] = issueTypeName
		data["issue_type"] = issueTypeNormalized
	}
	data["action"] = payload["action"]
}
