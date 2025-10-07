package handler

// prepareLabelData exposes label-related fields
func prepareLabelData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if label, ok := payload["label"].(map[string]any); ok {
		data["label"] = label
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
