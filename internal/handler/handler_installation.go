package handler

// prepareInstallationData populates data for installation events
func prepareInstallationData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract installation info
	if installation, ok := payload["installation"].(map[string]any); ok {
		data["installation"] = installation
		if id, ok := installation["id"].(float64); ok {
			data["installation_id"] = int(id)
		}
		// The shipped template renders {{app_name}}.
		if slug, ok := installation["app_slug"].(string); ok {
			data["app_name"] = slug
		}
	}

	// Extract repositories
	if repositories, ok := payload["repositories"].([]any); ok {
		data["repositories"] = repositories
		data["repositories_count"] = len(repositories)
	}

	data["installation_event"] = payload
}
