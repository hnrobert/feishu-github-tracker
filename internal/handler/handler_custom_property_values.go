package handler

// prepareCustomPropertyValuesData populates data for custom_property_values events
func prepareCustomPropertyValuesData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract property values
	if newValues, ok := payload["new_property_values"].([]any); ok {
		data["new_property_values"] = newValues
	}

	if oldValues, ok := payload["old_property_values"].([]any); ok {
		data["old_property_values"] = oldValues
	}

	data["custom_property_values"] = payload
}
