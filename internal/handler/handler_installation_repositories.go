package handler

// prepareInstallationRepositoriesData populates data for installation_repositories events
func prepareInstallationRepositoriesData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract added/removed repositories
	if added, ok := payload["repositories_added"].([]any); ok {
		data["repositories_added"] = added
		data["repositories_added_count"] = len(added)
	}

	if removed, ok := payload["repositories_removed"].([]any); ok {
		data["repositories_removed"] = removed
		data["repositories_removed_count"] = len(removed)
	}

	if selection, ok := payload["repository_selection"].(string); ok {
		data["repository_selection"] = selection
	}

	data["installation_repositories"] = payload
}
