package handler

// prepareWatchData handles watch events (star/watch changes)
func prepareWatchData(data map[string]any, payload map[string]any) {
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
	}
}
