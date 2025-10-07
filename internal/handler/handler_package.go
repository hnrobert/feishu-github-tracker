package handler

import "fmt"

// preparePackageData handles "package" event specific data extraction and
// builds markdown links preferring registry_package.html_url when present.
func preparePackageData(data map[string]any, payload map[string]any) {
	if pkg, ok := payload["package"].(map[string]any); ok {
		data["package"] = pkg
		if name, ok := pkg["name"]; ok {
			data["package_name"] = name
			if pname, ok2 := name.(string); ok2 {
				if pv, okpv := payload["package_version"].(map[string]any); okpv {
					if vname, vok := pv["version"].(string); vok && vname != "" {
						pkg["version"] = vname
						pkg["tag_name"] = vname
						data["package_version_name"] = vname
					}
					if purl, okurl := pv["html_url"].(string); okurl && purl != "" {
						pkg["html_url"] = purl
					}
					if up, okuk := pv["uploader"].(map[string]any); okuk {
						if login, lok := up["login"].(string); lok {
							if url, uok := up["html_url"].(string); uok {
								data["package_publisher_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
								if _, exists := data["sender_link_md"]; !exists {
									data["sender_link_md"] = data["package_publisher_link_md"]
								}
							}
						}
					}
					if au, aok := pv["author"].(map[string]any); aok {
						if login, lok := au["login"].(string); lok {
							if url, uok := au["html_url"].(string); uok {
								if _, exists := data["package_publisher_link_md"]; !exists {
									data["package_publisher_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
								}
								if _, exists := data["sender_link_md"]; !exists {
									data["sender_link_md"] = data["package_publisher_link_md"]
								}
							}
						}
					}
				}

				if ptype, okpt := pkg["package_type"].(string); okpt {
					data["package_type"] = ptype
					pkg["package_type"] = ptype
				}
				if pver, okpv := pkg["version"].(string); okpv {
					data["package_version"] = pver
					pkg["version"] = pver
				}
				if ptag, okptag := pkg["tag_name"].(string); okptag {
					data["package_tag_name"] = ptag
					pkg["tag_name"] = ptag
				}

				if rp, okrp := payload["registry_package"].(map[string]any); okrp {
					// prefer registry_package.package_version.html_url when available
					if pvRp, okpv := rp["package_version"].(map[string]any); okpv {
						if ph, okph := pvRp["html_url"].(string); okph && ph != "" {
							data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, ph)
						} else if rh, okh := rp["html_url"].(string); okh && rh != "" {
							data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, rh)
						}
					} else if rh, okh := rp["html_url"].(string); okh && rh != "" {
						data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, rh)
					}
				} else if purl, ok3 := pkg["html_url"].(string); ok3 && purl != "" {
					data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, purl)
				}
			}
		}
	}
	data["action"] = payload["action"]
}
