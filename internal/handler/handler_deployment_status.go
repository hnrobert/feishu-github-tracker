package handler

// prepareDeploymentStatusData exposes deployment_status object
func prepareDeploymentStatusData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)
	if ds, ok := payload["deployment_status"].(map[string]any); ok {
		data["deployment_status"] = ds
	}
	// The template renders {{deployment.ref}}; expose the raw object.
	if dep, ok := payload["deployment"].(map[string]any); ok {
		data["deployment"] = dep
	}
}
