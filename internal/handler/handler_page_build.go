package handler

// preparePageBuildData exposes page_build object
func preparePageBuildData(data map[string]any, payload map[string]any) {
	if pb, ok := payload["page_build"].(map[string]any); ok {
		data["page_build"] = pb
	}
}
