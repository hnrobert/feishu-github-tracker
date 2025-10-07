package handler

// prepareBranchProtectionConfigurationData populates data for branch_protection_configuration events
func prepareBranchProtectionConfigurationData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Add the raw payload for templates that need more detail
	data["branch_protection_configuration"] = payload
}
