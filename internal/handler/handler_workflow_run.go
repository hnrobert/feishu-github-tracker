package handler

// prepareWorkflowRunData exposes workflow_run object
func prepareWorkflowRunData(data map[string]any, payload map[string]any) {
	if wr, ok := payload["workflow_run"].(map[string]any); ok {
		data["workflow_run"] = wr
		if name, ok := wr["name"].(string); ok {
			data["workflow_name"] = name
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
