package handler

// prepareInstallationTargetData populates data for installation_target events
func prepareInstallationTargetData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract account info
	if account, ok := payload["account"].(map[string]any); ok {
		data["account"] = account
		if login, ok := account["login"].(string); ok {
			data["account_login"] = login
		}
	}

	// Extract changes
	if changes, ok := payload["changes"].(map[string]any); ok {
		data["changes"] = changes
		if loginChange, ok := changes["login"].(map[string]any); ok {
			if from, ok := loginChange["from"].(string); ok {
				data["old_login"] = from
			}
		}
	}

	if targetType, ok := payload["target_type"].(string); ok {
		data["target_type"] = targetType
	}

	data["installation_target"] = payload
}
