package handler

import (
	"fmt"
)

// prepareCommonData fills the common fields used by templates across events
func prepareCommonData(data map[string]any, payload map[string]any) {
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

	if sender, ok := payload["sender"].(map[string]any); ok {
		data["sender_name"] = sender["login"]
		data["sender_avatar"] = sender["avatar_url"]
		data["sender_url"] = sender["html_url"]
		data["sender"] = sender

		if login, ok := sender["login"].(string); ok {
			if surl, ok2 := sender["html_url"].(string); ok2 {
				data["sender_link_md"] = fmt.Sprintf("[%s](%s)", login, surl)
			}
		}
	}
}
