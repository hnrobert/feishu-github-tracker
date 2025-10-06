package handler

// prepareCreateData handles create event payload
func prepareCreateData(data map[string]any, payload map[string]any) {
	if ref, ok := payload["ref"].(string); ok {
		data["ref"] = ref
	}
	if rtype, ok := payload["ref_type"].(string); ok {
		data["ref_type"] = rtype
	}
	if mb, ok := payload["master_branch"].(string); ok {
		data["master_branch"] = mb
	}
}
