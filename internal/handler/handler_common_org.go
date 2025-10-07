package handler

import (
	"fmt"
)

// prepareOrgData fills organization-related fields used by templates
func prepareOrgData(data map[string]any, payload map[string]any) {
	if org, ok := payload["organization"].(map[string]any); ok {
		data["org_name"] = org["login"]
		data["org_avatar"] = org["avatar_url"]
		data["org_url"] = org["html_url"]
		data["organization"] = org

		if login, ok := org["login"].(string); ok {
			if url, ok2 := org["html_url"].(string); ok2 {
				data["org_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
			}
		}
	}
}
