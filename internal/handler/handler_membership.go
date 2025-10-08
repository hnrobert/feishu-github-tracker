package handler

// prepareMembershipData exposes membership event fields
func prepareMembershipData(data map[string]any, payload map[string]any) {
	if m, ok := payload["membership"].(map[string]any); ok {
		data["membership"] = m
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
