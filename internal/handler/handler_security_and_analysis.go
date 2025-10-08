package handler

// prepareSecurityAndAnalysisData populates data for security_and_analysis events
func prepareSecurityAndAnalysisData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract changes in security and analysis features
	if changes, ok := payload["changes"].(map[string]any); ok {
		data["changes"] = changes
	}

	data["security_and_analysis"] = payload
}
