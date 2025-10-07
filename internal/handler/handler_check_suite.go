package handler

import "fmt"

// prepareCheckSuiteData exposes check_suite object and ensures repository link is available
func prepareCheckSuiteData(data map[string]any, payload map[string]any) {
	if cs, ok := payload["check_suite"].(map[string]any); ok {
		data["check_suite"] = cs
	}
	// ensure repository link is available for check_suite templates
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repository"] = repo
		if full, okf := repo["full_name"].(string); okf {
			if url, oku := repo["html_url"].(string); oku {
				data["repository_link_md"] = fmt.Sprintf("[%s](%s)", full, url)
			}
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
