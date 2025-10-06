package handler

// prepareForkData handles fork event payload
func prepareForkData(data map[string]any, payload map[string]any) {
	if forkee, ok := payload["forkee"].(map[string]any); ok {
		data["forkee"] = forkee
		if full, ok2 := forkee["full_name"].(string); ok2 {
			data["forkee_full_name"] = full
		}
		if url, ok2 := forkee["html_url"].(string); ok2 {
			data["forkee_url"] = url
		}
	}
}
