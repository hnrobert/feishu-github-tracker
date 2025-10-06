package handler

// prepareCheckRunData exposes check_run
func prepareCheckRunData(data map[string]any, payload map[string]any) {
	if cr, ok := payload["check_run"].(map[string]any); ok {
		data["check_run"] = cr
	}
}
