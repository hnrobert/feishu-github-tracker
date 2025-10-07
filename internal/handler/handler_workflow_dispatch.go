package handler

// prepareWorkflowDispatchData populates data for workflow_dispatch events
func prepareWorkflowDispatchData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract workflow info
	if workflow, ok := payload["workflow"].(string); ok {
		data["workflow"] = workflow
	}

	// Extract inputs
	if inputs, ok := payload["inputs"].(map[string]any); ok {
		data["inputs"] = inputs
	}

	// Extract ref
	if ref, ok := payload["ref"].(string); ok {
		data["ref"] = ref
	}

	data["workflow_dispatch"] = payload
}
