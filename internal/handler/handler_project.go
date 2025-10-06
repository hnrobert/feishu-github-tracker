package handler

// prepareProjectData exposes project object and common fields
func prepareProjectData(data map[string]any, payload map[string]any) {
	if p, ok := payload["project"].(map[string]any); ok {
		data["project"] = p
		if name, ok := p["name"].(string); ok {
			data["project_name"] = name
		}
		if url, ok := p["html_url"].(string); ok {
			data["project_url"] = url
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
