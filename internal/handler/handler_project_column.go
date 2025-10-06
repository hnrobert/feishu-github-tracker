package handler

// prepareProjectColumnData exposes project_column object
func prepareProjectColumnData(data map[string]any, payload map[string]any) {
	if pc, ok := payload["project_column"].(map[string]any); ok {
		data["project_column"] = pc
		if name, ok := pc["name"].(string); ok {
			data["project_column_name"] = name
		}
	}
}
