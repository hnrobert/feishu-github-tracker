package handler

// preparePageBuildData exposes the build object. GitHub's page_build payload
// has no `action` field — the build outcome lives in build.status.
func preparePageBuildData(data map[string]any, payload map[string]any) {
	if build, ok := payload["build"].(map[string]any); ok {
		data["build"] = build
		if status, ok := build["status"].(string); ok {
			data["build_status"] = status
		}
		if pusher, ok := build["pusher"].(map[string]any); ok {
			if login, ok := pusher["login"].(string); ok {
				data["build_pusher"] = login
			}
		}
		if errObj, ok := build["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok {
				data["build_error"] = msg
			}
		}
	}
}
