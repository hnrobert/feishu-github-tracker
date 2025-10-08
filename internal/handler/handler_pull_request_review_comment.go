package handler

// preparePullRequestReviewCommentData handles comments on pull request reviews
func preparePullRequestReviewCommentData(data map[string]any, payload map[string]any) {
	if comment, ok := payload["comment"].(map[string]any); ok {
		data["comment_body"] = comment["body"]
		data["comment_url"] = comment["html_url"]
		if user, ok := comment["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["comment_user_link_md"] = "[" + login + "](" + url + ")"
				}
			}
		}
	}
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		data["pr_number"] = pr["number"]
		data["pr_title"] = pr["title"]
		data["pr_url"] = pr["html_url"]
	}
}
