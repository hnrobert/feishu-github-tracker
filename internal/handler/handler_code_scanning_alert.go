package handler

// prepareCodeScanningAlertData exposes basic fields for code_scanning_alert events
func prepareCodeScanningAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["code_scanning_alert"] = alert
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
