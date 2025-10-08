package handler

// Gollum (wiki) event: exposes pages and repository
func prepareGollumData(data map[string]any, payload map[string]any) {
	if pages, ok := payload["pages"].([]any); ok {
		data["pages"] = pages
	}
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
		if full, ok2 := repo["full_name"].(string); ok2 {
			data["repo_full_name"] = full
		}
		if url, ok2 := repo["html_url"].(string); ok2 {
			data["repo_url"] = url
		}
	}
}
