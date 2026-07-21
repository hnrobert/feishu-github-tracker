package handler

// prepareRepositoryImportData exposes basic fields for repository_import events
func prepareRepositoryImportData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if importData, ok := payload["import"].(map[string]any); ok {
		data["repository_import"] = importData
	}
}
