package handler

// prepareBranchProtectionRuleData populates data for branch_protection_rule events
func prepareBranchProtectionRuleData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract rule information
	if rule, ok := payload["rule"].(map[string]any); ok {
		data["rule"] = rule
		if name, ok := rule["name"].(string); ok {
			data["rule_name"] = name
		}
	}

	data["branch_protection_rule"] = payload
}
