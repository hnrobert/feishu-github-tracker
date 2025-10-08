package handler

// prepareCommitCommentData exposes basic fields for commit_comment events
func prepareCommitCommentData(data map[string]any, payload map[string]any) {
	// populate common fields
	prepareCommonData(data, payload)
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
	if comment, ok := payload["comment"].(map[string]any); ok {
		data["comment"] = comment
	}
}
