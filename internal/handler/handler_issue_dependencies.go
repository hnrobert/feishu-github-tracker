package handler

// prepareIssueDependenciesData populates data for issue_dependencies events
func prepareIssueDependenciesData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract blocked issue info
	if blockedIssue, ok := payload["blocked_issue"].(map[string]any); ok {
		data["blocked_issue"] = blockedIssue
		if number, ok := blockedIssue["number"].(float64); ok {
			data["blocked_issue_number"] = int(number)
		}
		if title, ok := blockedIssue["title"].(string); ok {
			data["blocked_issue_title"] = title
		}
	}

	// Extract blocking issue info
	if blockingIssue, ok := payload["blocking_issue"].(map[string]any); ok {
		data["blocking_issue"] = blockingIssue
		if number, ok := blockingIssue["number"].(float64); ok {
			data["blocking_issue_number"] = int(number)
		}
		if title, ok := blockingIssue["title"].(string); ok {
			data["blocking_issue_title"] = title
		}
	}

	// Extract blocking issue repository
	if blockingRepo, ok := payload["blocking_issue_repo"].(map[string]any); ok {
		data["blocking_issue_repo"] = blockingRepo
		if fullName, ok := blockingRepo["full_name"].(string); ok {
			data["blocking_issue_repo_name"] = fullName
		}
	}

	data["issue_dependencies"] = payload
}
