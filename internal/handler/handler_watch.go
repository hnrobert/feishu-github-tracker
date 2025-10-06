package handler

// prepareWatchData handles watch events (star/watch changes)
func prepareWatchData(data map[string]any, payload map[string]any) {
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
	}
}
