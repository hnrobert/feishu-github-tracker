package handler

// prepareDiscussionCommentData handles discussion_comment events
func prepareDiscussionCommentData(data map[string]any, payload map[string]any) {
	if comment, ok := payload["comment"].(map[string]any); ok {
		data["comment_body"] = comment["body"]
		data["comment_url"] = comment["html_url"]
		data["comment"] = comment
		if user, ok := comment["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["comment_user_link_md"] = "[" + login + "](" + url + ")"
				}
			}
		}
	}
	if discussion, ok := payload["discussion"].(map[string]any); ok {
		data["discussion_title"] = discussion["title"]
		data["discussion_url"] = discussion["html_url"]
		data["discussion"] = discussion
	}
}
