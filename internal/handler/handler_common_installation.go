package handler

// prepareInstallationData fills installation-related fields used by templates
func prepareInstallationCommonData(data map[string]any, payload map[string]any) {
	if installation, ok := payload["installation"].(map[string]any); ok {
		data["installation_id"] = installation["id"]
		data["installation"] = installation
	}
}
