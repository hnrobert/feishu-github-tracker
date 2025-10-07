package handler

// prepareCustomPropertyData populates data for custom_property events
func prepareCustomPropertyData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract custom property definition
	if definition, ok := payload["definition"].(map[string]any); ok {
		data["definition"] = definition
		if name, ok := definition["property_name"].(string); ok {
			data["property_name"] = name
		}
	}

	data["custom_property"] = payload
}
