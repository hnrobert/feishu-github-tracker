package handler

import "fmt"

func preparePullRequestData(data map[string]any, payload map[string]any) {
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		data["pr_number"] = pr["number"]
		data["pr_title"] = pr["title"]
		data["pr_url"] = pr["html_url"]
		data["pr_state"] = pr["state"]
		data["pr_merged"] = pr["merged"]
		data["pr_body"] = pr["body"]
		data["pull_request"] = pr

		if user, ok := pr["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["pr_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
				}
			}
		}

		if head, ok := pr["head"].(map[string]any); ok {
			data["pr_head_ref"] = head["ref"]
			if repo, ok := payload["repository"].(map[string]any); ok {
				if url, ok2 := repo["html_url"].(string); ok2 {
					if href, ok3 := head["ref"].(string); ok3 {
						data["pr_head_branch_link_md"] = fmt.Sprintf("[%s](%s/tree/%s)", href, url, href)
					}
				}
			}
		}

		if base, ok := pr["base"].(map[string]any); ok {
			data["pr_base_ref"] = base["ref"]
			if repo, ok := payload["repository"].(map[string]any); ok {
				if url, ok2 := repo["html_url"].(string); ok2 {
					if bref, ok3 := base["ref"].(string); ok3 {
						data["pr_base_branch_link_md"] = fmt.Sprintf("[%s](%s/tree/%s)", bref, url, bref)
					}
				}
			}
		}
	}
	data["action"] = payload["action"]
}
