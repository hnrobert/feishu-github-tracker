package handler

// prepareInstallationData populates data for installation events
func prepareInstallationData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract installation info
	if installation, ok := payload["installation"].(map[string]any); ok {
		data["installation"] = installation
		if id, ok := installation["id"].(float64); ok {
			data["installation_id"] = int(id)
		}
	}

	// Extract repositories
	if repositories, ok := payload["repositories"].([]any); ok {
		data["repositories"] = repositories
		data["repositories_count"] = len(repositories)
	}

	data["installation_event"] = payload
}
