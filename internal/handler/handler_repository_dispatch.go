package handler

// prepareRepositoryDispatchData populates data for repository_dispatch events
func prepareRepositoryDispatchData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	// Extract event type and client payload
	if eventType, ok := payload["event_type"].(string); ok {
		data["event_type"] = eventType
	}

	if clientPayload, ok := payload["client_payload"].(map[string]any); ok {
		data["client_payload"] = clientPayload
	}

	data["repository_dispatch"] = payload
}
