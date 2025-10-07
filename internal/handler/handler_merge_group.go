package handler

// prepareMergeGroupData populates data for merge_group events
func prepareMergeGroupData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract merge group info
	if mergeGroup, ok := payload["merge_group"].(map[string]any); ok {
		data["merge_group"] = mergeGroup

		if headSha, ok := mergeGroup["head_sha"].(string); ok {
			data["head_sha"] = headSha
		}

		if headRef, ok := mergeGroup["head_ref"].(string); ok {
			data["head_ref"] = headRef
		}

		if baseSha, ok := mergeGroup["base_sha"].(string); ok {
			data["base_sha"] = baseSha
		}

		if baseRef, ok := mergeGroup["base_ref"].(string); ok {
			data["base_ref"] = baseRef
		}
	}
}
