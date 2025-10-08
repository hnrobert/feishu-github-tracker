package handler

// prepareSecretScanningScanData populates data for secret_scanning_scan events
func prepareSecretScanningScanData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract scan info
	if scan, ok := payload["scan"].(map[string]any); ok {
		data["scan"] = scan
		if status, ok := scan["status"].(string); ok {
			data["scan_status"] = status
		}
		if completedAt, ok := scan["completed_at"].(string); ok {
			data["scan_completed_at"] = completedAt
		}
	}

	data["secret_scanning_scan"] = payload
}
