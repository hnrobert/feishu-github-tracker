package handler

// prepareGitHubAppAuthorizationData populates data for github_app_authorization events
func prepareGitHubAppAuthorizationData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	data["github_app_authorization"] = payload
}
