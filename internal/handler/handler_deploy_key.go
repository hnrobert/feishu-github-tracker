package handler

// prepareDeployKeyData exposes basic fields for deploy_key events
func prepareDeployKeyData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if key, ok := payload["key"].(map[string]any); ok {
		data["deploy_key"] = key
	}
}
