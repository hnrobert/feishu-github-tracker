package handler

// prepareSubIssuesData populates data for sub_issues events
func prepareSubIssuesData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract parent issue info
	if parentIssue, ok := payload["parent_issue"].(map[string]any); ok {
		data["parent_issue"] = parentIssue
		if number, ok := parentIssue["number"].(float64); ok {
			data["parent_issue_number"] = int(number)
		}
		if title, ok := parentIssue["title"].(string); ok {
			data["parent_issue_title"] = title
		}
	}

	// Extract sub issue info
	if subIssue, ok := payload["sub_issue"].(map[string]any); ok {
		data["sub_issue"] = subIssue
		if number, ok := subIssue["number"].(float64); ok {
			data["sub_issue_number"] = int(number)
		}
		if title, ok := subIssue["title"].(string); ok {
			data["sub_issue_title"] = title
		}
	}

	data["sub_issues"] = payload
}
