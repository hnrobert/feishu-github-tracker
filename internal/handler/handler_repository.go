package handler

// prepareRepositoryData exposes repository object and convenient fields
func prepareRepositoryData(data map[string]any, payload map[string]any) {
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
		if full, ok2 := repo["full_name"].(string); ok2 {
			data["repo_full_name"] = full
		}
		if url, ok2 := repo["html_url"].(string); ok2 {
			data["repo_url"] = url
		}
	}

	// Ensure templates can access the event action for repository events
	// Many other prepare* functions set data["action"] = payload["action"].
	// repository events did not — that caused {{action}} to be empty in templates.
	if a, ok := payload["action"]; ok {
		data["action"] = a
	}
}
