package handler

// prepareProjectsV2StatusUpdateData populates data for projects_v2_status_update events
func prepareProjectsV2StatusUpdateData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract status update info
	if statusUpdate, ok := payload["projects_v2_status_update"].(map[string]any); ok {
		data["projects_v2_status_update"] = statusUpdate

		if id, ok := statusUpdate["id"].(float64); ok {
			data["status_update_id"] = int(id)
		}

		if body, ok := statusUpdate["body"].(string); ok {
			data["status_update_body"] = body
		}

		if status, ok := statusUpdate["status"].(string); ok {
			data["status"] = status
		}
	}
}
