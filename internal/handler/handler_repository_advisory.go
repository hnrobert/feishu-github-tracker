package handler

// prepareRepositoryAdvisoryData populates data for repository_advisory events
func prepareRepositoryAdvisoryData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract repository advisory info
	if advisory, ok := payload["repository_advisory"].(map[string]any); ok {
		data["repository_advisory"] = advisory

		if ghsaID, ok := advisory["ghsa_id"].(string); ok {
			data["advisory_id"] = ghsaID
		}

		if summary, ok := advisory["summary"].(string); ok {
			data["advisory_summary"] = summary
		}

		if severity, ok := advisory["severity"].(string); ok {
			data["advisory_severity"] = severity
		}
	}
}
