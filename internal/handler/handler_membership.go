package handler

// prepareMembershipData exposes membership event fields
func prepareMembershipData(data map[string]any, payload map[string]any) {
	if m, ok := payload["membership"].(map[string]any); ok {
		data["membership"] = m
	}
}
