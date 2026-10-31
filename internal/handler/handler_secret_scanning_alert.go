package handler

// prepareSecretScanningAlertData exposes basic fields for secret_scanning_alert events
func prepareSecretScanningAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["secret_scanning_alert"] = alert
		// Flat convenience keys used by the shipped card templates.
		if state, ok := alert["state"].(string); ok {
			data["alert_state"] = state
		}
		if display, ok := alert["secret_type_display_name"].(string); ok && display != "" {
			data["alert_secret_type"] = display
		} else if raw, ok := alert["secret_type"].(string); ok {
			data["alert_secret_type"] = raw
		}
	}
}
