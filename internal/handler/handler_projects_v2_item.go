package handler

// prepareProjectsV2ItemData populates data for projects_v2_item events
func prepareProjectsV2ItemData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract project v2 item info
	if item, ok := payload["projects_v2_item"].(map[string]any); ok {
		data["projects_v2_item"] = item

		if id, ok := item["id"].(float64); ok {
			data["item_id"] = int(id)
		}

		if nodeID, ok := item["node_id"].(string); ok {
			data["item_node_id"] = nodeID
		}

		if projectNodeID, ok := item["project_node_id"].(string); ok {
			data["project_node_id"] = projectNodeID
		}

		if contentNodeID, ok := item["content_node_id"].(string); ok {
			data["content_node_id"] = contentNodeID
		}

		if contentType, ok := item["content_type"].(string); ok {
			data["content_type"] = contentType
		}
	}

	// Extract changes if present
	if changes, ok := payload["changes"].(map[string]any); ok {
		data["changes"] = changes
	}
}
