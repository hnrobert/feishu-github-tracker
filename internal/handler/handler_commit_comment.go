package handler

// prepareCommitCommentData exposes basic fields for commit_comment events
func prepareCommitCommentData(data map[string]any, payload map[string]any) {
	// populate common fields
	prepareCommonData(data, payload)
	if comment, ok := payload["comment"].(map[string]any); ok {
		data["comment"] = comment
		// Flat convenience keys used by the shipped card templates.
		if body, ok := comment["body"].(string); ok {
			data["comment_body"] = body
		}
		if url, ok := comment["html_url"].(string); ok {
			data["comment_html_url"] = url
		}
		if commitID, ok := comment["commit_id"].(string); ok && len(commitID) > 7 {
			data["commit_sha_short"] = commitID[:7]
		} else if commitID, ok := comment["commit_id"].(string); ok {
			data["commit_sha_short"] = commitID
		}
	}
}
