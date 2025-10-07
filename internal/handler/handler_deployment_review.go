package handler

// prepareDeploymentReviewData populates data for deployment_review events
func prepareDeploymentReviewData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract approver info
	if approver, ok := payload["approver"].(map[string]any); ok {
		data["approver"] = approver
		if login, ok := approver["login"].(string); ok {
			data["approver_login"] = login
			if htmlURL, ok := approver["html_url"].(string); ok {
				data["approver_link_md"] = "[" + login + "](" + htmlURL + ")"
			}
		}
	}

	// Extract comment
	if comment, ok := payload["comment"].(string); ok {
		data["comment"] = comment
	}

	// Extract workflow run info
	if workflowRun, ok := payload["workflow_run"].(map[string]any); ok {
		data["workflow_run"] = workflowRun
	}

	data["deployment_review"] = payload
}
