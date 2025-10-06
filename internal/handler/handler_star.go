package handler

// prepareStarData handles star (watch) event payload
func prepareStarData(data map[string]any, payload map[string]any) {
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
