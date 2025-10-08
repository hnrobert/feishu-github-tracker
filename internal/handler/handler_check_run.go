package handler

// prepareCheckRunData exposes check_run and action
func prepareCheckRunData(data map[string]any, payload map[string]any) {
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	if cr, ok := payload["check_run"].(map[string]any); ok {
		data["check_run"] = cr
	}
}
