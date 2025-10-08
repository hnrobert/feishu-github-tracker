package handler

// prepareProjectCardData exposes project_card
func prepareProjectCardData(data map[string]any, payload map[string]any) {
	if pc, ok := payload["project_card"].(map[string]any); ok {
		data["project_card"] = pc
	}
}
