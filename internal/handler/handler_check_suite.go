package handler

// prepareCheckSuiteData exposes check_suite object
func prepareCheckSuiteData(data map[string]any, payload map[string]any) {
	if cs, ok := payload["check_suite"].(map[string]any); ok {
		data["check_suite"] = cs
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
