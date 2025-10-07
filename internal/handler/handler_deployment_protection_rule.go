package handler

// prepareDeploymentProtectionRuleData populates data for deployment_protection_rule events
func prepareDeploymentProtectionRuleData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract environment and deployment info
	if environment, ok := payload["environment"].(string); ok {
		data["environment"] = environment
	}

	if deployment, ok := payload["deployment"].(map[string]any); ok {
		data["deployment"] = deployment
	}

	if callbackURL, ok := payload["deployment_callback_url"].(string); ok {
		data["deployment_callback_url"] = callbackURL
	}

	data["deployment_protection_rule"] = payload
}
