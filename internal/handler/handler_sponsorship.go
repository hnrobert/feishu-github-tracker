package handler

// prepareSponsorshipData populates data for sponsorship events
func prepareSponsorshipData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract sponsorship info
	if sponsorship, ok := payload["sponsorship"].(map[string]any); ok {
		data["sponsorship"] = sponsorship

		// Extract sponsor info
		if sponsor, ok := sponsorship["sponsor"].(map[string]any); ok {
			if login, ok := sponsor["login"].(string); ok {
				data["sponsor_login"] = login
				if htmlURL, ok := sponsor["html_url"].(string); ok {
					data["sponsor_link_md"] = "[" + login + "](" + htmlURL + ")"
				}
			}
		}

		// Extract sponsorable info
		if sponsorable, ok := sponsorship["sponsorable"].(map[string]any); ok {
			if login, ok := sponsorable["login"].(string); ok {
				data["sponsorable_login"] = login
			}
		}

		// Extract tier info
		if tier, ok := sponsorship["tier"].(map[string]any); ok {
			if name, ok := tier["name"].(string); ok {
				data["tier_name"] = name
			}
			if monthlyPriceInCents, ok := tier["monthly_price_in_cents"].(float64); ok {
				data["tier_monthly_price_cents"] = int(monthlyPriceInCents)
				data["tier_monthly_price_dollars"] = monthlyPriceInCents / 100
			}
		}
	}
}
