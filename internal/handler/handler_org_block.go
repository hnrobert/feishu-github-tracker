package handler

// prepareOrgBlockData populates data for org_block events
func prepareOrgBlockData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract blocked user info
	if blockedUser, ok := payload["blocked_user"].(map[string]any); ok {
		data["blocked_user"] = blockedUser
		if login, ok := blockedUser["login"].(string); ok {
			data["blocked_user_login"] = login
			if htmlURL, ok := blockedUser["html_url"].(string); ok {
				data["blocked_user_link_md"] = "[" + login + "](" + htmlURL + ")"
			}
		}
	}

	data["org_block"] = payload
}
