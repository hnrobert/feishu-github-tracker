package handler

// prepareWorkflowJobData populates data for workflow_job events
func prepareWorkflowJobData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract workflow job info
	if job, ok := payload["workflow_job"].(map[string]any); ok {
		data["workflow_job"] = job

		if id, ok := job["id"].(float64); ok {
			data["job_id"] = int(id)
		}

		if name, ok := job["name"].(string); ok {
			data["job_name"] = name
		}

		if status, ok := job["status"].(string); ok {
			data["job_status"] = status
		}

		if conclusion, ok := job["conclusion"].(string); ok {
			data["job_conclusion"] = conclusion
		}

		if htmlURL, ok := job["html_url"].(string); ok {
			data["job_url"] = htmlURL
		}

		if runID, ok := job["run_id"].(float64); ok {
			data["run_id"] = int(runID)
		}
	}
}
