package handler

// prepareStatusData exposes status event fields
func prepareStatusData(data map[string]any, payload map[string]any) {
	if s, ok := payload["status"].(map[string]any); ok {
		data["status"] = s
		if state, ok := s["state"].(string); ok {
			data["status_state"] = state
		}
	}
}
