package handler

// prepareDeleteData handles delete event payload
func prepareDeleteData(data map[string]any, payload map[string]any) {
	if ref, ok := payload["ref"].(string); ok {
		data["ref"] = ref
	}
	if rtype, ok := payload["ref_type"].(string); ok {
		data["ref_type"] = rtype
	}
}
