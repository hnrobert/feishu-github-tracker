package handler

// prepareOrganizationData exposes organization object
func prepareOrganizationData(data map[string]any, payload map[string]any) {
	if org, ok := payload["organization"].(map[string]any); ok {
		data["organization"] = org
		if login, lok := org["login"].(string); lok {
			data["organization_login"] = login
		}
	}
}
