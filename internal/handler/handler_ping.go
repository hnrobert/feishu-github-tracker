package handler

// preparePingData populates data for ping events
func preparePingData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract zen message
	if zen, ok := payload["zen"].(string); ok {
		data["zen"] = zen
	}

	// Extract hook info
	if hook, ok := payload["hook"].(map[string]any); ok {
		data["hook"] = hook
		if id, ok := hook["id"].(float64); ok {
			data["hook_id"] = int(id)
		}
		if hookType, ok := hook["type"].(string); ok {
			data["hook_type"] = hookType
		}
	}

	if hookID, ok := payload["hook_id"].(float64); ok {
		data["hook_id"] = int(hookID)
	}

	data["ping"] = payload
}

// uniqueStrings returns a slice with duplicate strings removed
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, str := range input {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}
