package handler

// prepareMilestoneData exposes milestone object
func prepareMilestoneData(data map[string]any, payload map[string]any) {
	if m, ok := payload["milestone"].(map[string]any); ok {
		data["milestone"] = m
		if title, ok := m["title"].(string); ok {
			data["milestone_title"] = title
		}
		if desc, ok := m["description"].(string); ok {
			data["milestone_description"] = desc
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
