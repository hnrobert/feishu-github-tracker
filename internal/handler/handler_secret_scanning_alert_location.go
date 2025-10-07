package handler

// prepareSecretScanningAlertLocationData populates data for secret_scanning_alert_location events
func prepareSecretScanningAlertLocationData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract alert info
	if alert, ok := payload["alert"].(map[string]any); ok {
		data["alert"] = alert
		if number, ok := alert["number"].(float64); ok {
			data["alert_number"] = int(number)
		}
	}

	// Extract location info
	if location, ok := payload["location"].(map[string]any); ok {
		data["location"] = location
		if locationType, ok := location["type"].(string); ok {
			data["location_type"] = locationType
		}
	}

	data["secret_scanning_alert_location"] = payload
}
