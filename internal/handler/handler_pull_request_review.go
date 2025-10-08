package handler

// preparePullRequestReviewData handles pull_request_review events
func preparePullRequestReviewData(data map[string]any, payload map[string]any) {
	if review, ok := payload["review"].(map[string]any); ok {
		data["review_state"] = review["state"]
		data["review_body"] = review["body"]
		data["review_url"] = review["html_url"]
		data["review"] = review
		if user, ok := review["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["review_user_link_md"] = "[" + login + "](" + url + ")"
				}
			}
		}
	}
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		data["pr_number"] = pr["number"]
		data["pr_title"] = pr["title"]
		data["pr_url"] = pr["html_url"]
		data["pull_request"] = pr
	}
}
