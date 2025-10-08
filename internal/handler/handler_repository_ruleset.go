package handler

// prepareRepositoryRulesetData exposes basic fields for repository_ruleset events
func prepareRepositoryRulesetData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if ruleset, ok := payload["ruleset"].(map[string]any); ok {
		data["repository_ruleset"] = ruleset
	}
	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
