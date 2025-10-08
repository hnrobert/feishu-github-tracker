package handler

// prepareDeploymentData exposes deployment object and summary fields
func prepareDeploymentData(data map[string]any, payload map[string]any) {
	if d, ok := payload["deployment"].(map[string]any); ok {
		data["deployment"] = d
		if id, ok := d["id"]; ok {
			data["deployment_id"] = id
		}
		if url, ok := d["url"].(string); ok {
			data["deployment_url"] = url
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
