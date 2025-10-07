package handler

import (
	"fmt"
)

// prepareRepoData fills repository-related fields used by templates
func prepareRepoData(data map[string]any, payload map[string]any) {
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repo_name"] = repo["name"]
		data["repo_full_name"] = repo["full_name"]
		data["repo_url"] = repo["html_url"]
		data["repository"] = repo

		if full, ok := repo["full_name"].(string); ok {
			if url, ok2 := repo["html_url"].(string); ok2 {
				data["repository_link_md"] = fmt.Sprintf("[%s](%s)", full, url)
			}
		}
	}
}
