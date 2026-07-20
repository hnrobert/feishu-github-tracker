package handler

import "fmt"

// prepareIssueCommentData handles issue_comment events
func prepareIssueCommentData(data map[string]any, payload map[string]any) {
	if comment, ok := payload["comment"].(map[string]any); ok {
		data["comment_body"] = comment["body"]
		data["comment_url"] = comment["html_url"]
		data["comment"] = comment
		if user, ok := comment["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					data["comment_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
				}
			}
		}
	}
	if issue, ok := payload["issue"].(map[string]any); ok {
		data["issue_number"] = issue["number"]
		data["issue_title"] = issue["title"]
		data["issue_url"] = issue["html_url"]
		data["issue"] = issue
		if url, ok := issue["html_url"].(string); ok {
			if title, ok := issue["title"].(string); ok && title != "" {
				data["issue_link_md"] = fmt.Sprintf("[#%v %s](%s)", issue["number"], title, url)
			} else {
				data["issue_link_md"] = url
			}
		}
	}
}
