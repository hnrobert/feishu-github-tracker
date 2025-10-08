package handler

// prepareSecretScanningAlertData exposes basic fields for secret_scanning_alert events
func prepareSecretScanningAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["secret_scanning_alert"] = alert
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
