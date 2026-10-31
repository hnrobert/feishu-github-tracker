package handler

// prepareStatusData exposes status event fields. GitHub's status payload keeps
// state/context/description at the TOP LEVEL (there is no "status" object), so
// those are read directly; status_state is kept as a legacy alias.
func prepareStatusData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if state, ok := payload["state"].(string); ok {
		data["state"] = state
		data["status_state"] = state
	}
	if context, ok := payload["context"].(string); ok {
		data["context"] = context
	}
	if description, ok := payload["description"].(string); ok {
		data["status_description"] = description
	}
	if targetURL, ok := payload["target_url"].(string); ok {
		data["status_target_url"] = targetURL
	}
	if branches, ok := payload["branches"].([]any); ok {
		data["status_branches"] = branches
	}
}
