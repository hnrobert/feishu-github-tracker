package handler

// prepareMarketplacePurchaseData populates data for marketplace_purchase events
func prepareMarketplacePurchaseData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract marketplace purchase info
	if purchase, ok := payload["marketplace_purchase"].(map[string]any); ok {
		data["marketplace_purchase"] = purchase

		if account, ok := purchase["account"].(map[string]any); ok {
			if login, ok := account["login"].(string); ok {
				data["account_login"] = login
			}
		}

		if plan, ok := purchase["plan"].(map[string]any); ok {
			if name, ok := plan["name"].(string); ok {
				data["plan_name"] = name
			}
		}
	}

	// Extract previous purchase if exists
	if prevPurchase, ok := payload["previous_marketplace_purchase"].(map[string]any); ok {
		data["previous_marketplace_purchase"] = prevPurchase
	}

	if effectiveDate, ok := payload["effective_date"].(string); ok {
		data["effective_date"] = effectiveDate
	}
}
