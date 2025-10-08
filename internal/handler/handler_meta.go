package handler

// prepareMetaData populates data for meta events (webhook lifecycle)
func prepareMetaData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract hook info
	if hook, ok := payload["hook"].(map[string]any); ok {
		data["hook"] = hook
		if id, ok := hook["id"].(float64); ok {
			data["hook_id"] = int(id)
		}
		if hookType, ok := hook["type"].(string); ok {
			data["hook_type"] = hookType
		}
	}

	if hookID, ok := payload["hook_id"].(float64); ok {
		data["hook_id"] = int(hookID)
	}

	data["meta"] = payload
}
