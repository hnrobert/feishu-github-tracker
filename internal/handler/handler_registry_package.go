package handler

// prepareRegistryPackageData populates data for registry_package events
func prepareRegistryPackageData(data map[string]any, payload map[string]any) {
	prepareCommonData(data, payload)

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}

	// Extract registry package info (legacy GitHub Packages)
	if pkg, ok := payload["registry_package"].(map[string]any); ok {
		data["registry_package"] = pkg

		if name, ok := pkg["name"].(string); ok {
			data["package_name"] = name
		}

		if pkgType, ok := pkg["package_type"].(string); ok {
			data["package_type"] = pkgType
		}

		if version, ok := pkg["package_version"].(map[string]any); ok {
			if versionStr, ok := version["version"].(string); ok {
				data["package_version"] = versionStr
			}
		}
	}
}
