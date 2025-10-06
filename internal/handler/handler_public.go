package handler

// preparePublicData exposes public event fields
func preparePublicData(data map[string]any, payload map[string]any) {
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
