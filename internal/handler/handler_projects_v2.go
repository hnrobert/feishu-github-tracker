package handler

// prepareProjectsV2Data populates data for projects_v2 events
func prepareProjectsV2Data(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract project v2 info
	if project, ok := payload["projects_v2"].(map[string]any); ok {
		data["projects_v2"] = project

		if id, ok := project["id"].(float64); ok {
			data["project_id"] = int(id)
		}

		if title, ok := project["title"].(string); ok {
			data["project_title"] = title
		}

		if shortDescription, ok := project["short_description"].(string); ok {
			data["project_description"] = shortDescription
		}
	}
}
