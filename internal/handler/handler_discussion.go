package handler

// prepareDiscussionData handles discussion events
func prepareDiscussionData(data map[string]any, payload map[string]any) {
	if discussion, ok := payload["discussion"].(map[string]any); ok {
		// Expose the raw object so templates can use dotted paths such as
		// {{discussion.title}} and {{discussion.category.name}} (the shipped
		// discussion cards rely on them; the flat *_title/_url keys alone are
		// not enough). Mirrors prepareDiscussionCommentData.
		data["discussion"] = discussion
		data["discussion_title"] = discussion["title"]
		data["discussion_url"] = discussion["html_url"]
		data["discussion_body"] = discussion["body"]
		if category, ok := discussion["category"].(map[string]any); ok {
			data["discussion_category"] = category["name"]
		}
		if user, ok := discussion["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["discussion_user_link_md"] = "[" + login + "](" + url + ")"
				}
			}
		}
	}
}
