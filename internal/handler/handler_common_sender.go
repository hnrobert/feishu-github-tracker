package handler

import (
	"fmt"
)

// prepareSenderData fills sender-related fields used by templates
func prepareSenderData(data map[string]any, payload map[string]any) {
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
