package handler

// prepareReleaseData handles release event fields
func prepareReleaseData(data map[string]any, payload map[string]any) {
	if release, ok := payload["release"].(map[string]any); ok {
		data["release_name"] = release["name"]
		data["release_tag"] = release["tag_name"]
		data["release_url"] = release["html_url"]
		data["release_body"] = release["body"]
		data["release"] = release
	}
	data["action"] = payload["action"]
}
