package handler

// prepareDependabotAlertData exposes basic fields for dependabot_alert events
func prepareDependabotAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["dependabot_alert"] = alert
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
