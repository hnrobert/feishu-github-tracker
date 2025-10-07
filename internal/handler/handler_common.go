package handler

// prepareCommonData fills the common fields used by templates across events
// This is a convenience wrapper that calls all specialized common data preparation functions
func prepareCommonData(data map[string]any, payload map[string]any) {
	prepareRepoData(data, payload)
	prepareSenderData(data, payload)
	prepareOrgData(data, payload)
	prepareInstallationCommonData(data, payload)
}
