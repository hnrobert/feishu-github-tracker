package handler

// prepareDependabotAlertData exposes basic fields for dependabot_alert events
func prepareDependabotAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["dependabot_alert"] = alert
		// Flat convenience keys used by the shipped card templates.
		if state, ok := alert["state"].(string); ok {
			data["alert_state"] = state
		}
		if advisory, ok := alert["security_advisory"].(map[string]any); ok {
			if sev, ok := advisory["severity"].(string); ok {
				data["alert_severity"] = sev
			}
		}
		if dependency, ok := alert["dependency"].(map[string]any); ok {
			if pkg, ok := dependency["package"].(map[string]any); ok {
				if name, ok := pkg["name"].(string); ok {
					data["alert_package_name"] = name
				}
			}
		}
	}
}
