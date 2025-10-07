package handler

// prepareTeamAddData populates data for team_add events
func prepareTeamAddData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract team info
	if team, ok := payload["team"].(map[string]any); ok {
		data["team"] = team
		if name, ok := team["name"].(string); ok {
			data["team_name"] = name
		}
		if slug, ok := team["slug"].(string); ok {
			data["team_slug"] = slug
		}
	}

	data["team_add"] = payload
}
