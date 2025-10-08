package handler

// preparePersonalAccessTokenRequestData populates data for personal_access_token_request events
func preparePersonalAccessTokenRequestData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract token request info
	if tokenRequest, ok := payload["personal_access_token_request"].(map[string]any); ok {
		data["personal_access_token_request"] = tokenRequest

		if id, ok := tokenRequest["id"].(float64); ok {
			data["request_id"] = int(id)
		}

		if owner, ok := tokenRequest["owner"].(map[string]any); ok {
			if login, ok := owner["login"].(string); ok {
				data["token_owner_login"] = login
			}
		}

		if tokenName, ok := tokenRequest["token_name"].(string); ok {
			data["token_name"] = tokenName
		}

		if tokenExpired, ok := tokenRequest["token_expired"].(bool); ok {
			data["token_expired"] = tokenExpired
		}
	}
}
