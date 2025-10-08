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

			// Extract container tag name from container_metadata.tag.name
			if cm, okcm := version["container_metadata"].(map[string]any); okcm {
				if tagObj, okt := cm["tag"].(map[string]any); okt {
					if tagName, tokk := tagObj["name"].(string); tokk && tagName != "" {
						data["package_tag_name"] = tagName
					}
				}
			}
		}
	}
}
