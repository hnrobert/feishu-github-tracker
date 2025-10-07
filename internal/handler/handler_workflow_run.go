package handler

// prepareWorkflowRunData exposes workflow_run object and provides a small
// compatibility shim: templates expect `workflow.name` while payload contains
// `workflow_run.name`, so we populate `workflow` with the name. We also
// normalize numeric `id` when it's an integer to avoid scientific notation in
// templates.
func prepareWorkflowRunData(data map[string]any, payload map[string]any) {
	if wr, ok := payload["workflow_run"].(map[string]any); ok {
		// normalize id: if it's a float64 but has no fractional part, convert to int64
		if idv, okid := wr["id"].(float64); okid {
			if float64(int64(idv)) == idv {
				wr["id"] = int64(idv)
			}
		}

		data["workflow_run"] = wr

		// expose workflow.name for templates that reference `workflow.name`
		if name, ok := wr["name"].(string); ok {
			data["workflow_name"] = name
			data["workflow"] = map[string]any{"name": name}
		}
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
