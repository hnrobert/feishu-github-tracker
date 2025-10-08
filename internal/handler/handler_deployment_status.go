package handler

// prepareDeploymentStatusData exposes deployment_status object
func prepareDeploymentStatusData(data map[string]any, payload map[string]any) {
	if ds, ok := payload["deployment_status"].(map[string]any); ok {
		data["deployment_status"] = ds
	}
}
