package handler

// prepareTeamData exposes team
func prepareTeamData(data map[string]any, payload map[string]any) {
	if team, ok := payload["team"].(map[string]any); ok {
		data["team"] = team
	}
}
