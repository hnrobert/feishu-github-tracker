package handler

// prepareMemberData exposes member object (when a member is added/removed)
func prepareMemberData(data map[string]any, payload map[string]any) {
	if m, ok := payload["member"].(map[string]any); ok {
		data["member"] = m
		if login, lok := m["login"].(string); lok {
			data["member_login"] = login
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
