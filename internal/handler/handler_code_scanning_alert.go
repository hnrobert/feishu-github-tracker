package handler

// prepareCodeScanningAlertData exposes basic fields for code_scanning_alert events
func prepareCodeScanningAlertData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["code_scanning_alert"] = alert
		// Flat convenience keys used by the shipped card templates.
		if state, ok := alert["state"].(string); ok {
			data["alert_state"] = state
		}
		if rule, ok := alert["rule"].(map[string]any); ok {
			if desc, ok := rule["description"].(string); ok {
				data["alert_rule_description"] = desc
			}
			if sev, ok := rule["severity"].(string); ok {
				data["alert_rule_severity"] = sev
			}
		}
	}
}
