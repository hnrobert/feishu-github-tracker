package handler

// prepareSecurityAdvisoryData exposes security_advisory object
func prepareSecurityAdvisoryData(data map[string]any, payload map[string]any) {
	if sa, ok := payload["security_advisory"].(map[string]any); ok {
		data["security_advisory"] = sa
		if ghsa, ok2 := sa["ghsa_id"].(string); ok2 {
			data["security_advisory_id"] = ghsa
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
