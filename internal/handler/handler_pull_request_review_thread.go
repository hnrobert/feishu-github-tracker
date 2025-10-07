package handler

// preparePullRequestReviewThreadData populates data for pull_request_review_thread events
func preparePullRequestReviewThreadData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract pull request info
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		data["pull_request"] = pr
		if number, ok := pr["number"].(float64); ok {
			data["pr_number"] = int(number)
		}
		if title, ok := pr["title"].(string); ok {
			data["pr_title"] = title
		}
		if htmlURL, ok := pr["html_url"].(string); ok {
			data["pr_url"] = htmlURL
		}
	}

	// Extract thread info
	if thread, ok := payload["thread"].(map[string]any); ok {
		data["thread"] = thread
		if nodeID, ok := thread["node_id"].(string); ok {
			data["thread_id"] = nodeID
		}
		if comments, ok := thread["comments"].([]any); ok {
			data["thread_comments"] = comments
			data["thread_comments_count"] = len(comments)
		}
	}

	data["pull_request_review_thread"] = payload
}
