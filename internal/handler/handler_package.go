package handler

import (
	"fmt"
	"regexp"
	"strings"
)

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

				// If this is a CONTAINER package, prefer the GitHub package page under the repo:
				// https://github.com/{owner}/{repo}/pkgs/container/{package_name}
				if ptype, okpt := pkg["package_type"].(string); okpt && strings.ToUpper(ptype) == "CONTAINER" {
					if repoObj, okrepo := payload["repository"].(map[string]any); okrepo {
						if full, okfull := repoObj["full_name"].(string); okfull && full != "" {
							purl := fmt.Sprintf("https://github.com/%s/pkgs/container/%s", full, pname)
							data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, purl)
						}
					}
				}
				// Helper to detect owner-level /packages/<id> URLs which are less useful
				isOwnerPackagesURL := func(u string) bool {
					if u == "" {
						return false
					}
					// match e.g. https://github.com/owner/packages/12345 or .../packages/12345
					re := regexp.MustCompile(`/packages/\d+$`)
					return re.MatchString(u)
				}

				finalURL := ""
				// prefer registry_package.package_version.html_url when available and not an owner-level packages id
				if rp, okrp := payload["registry_package"].(map[string]any); okrp {
					if pvRp, okpv := rp["package_version"].(map[string]any); okpv {
						if ph, okph := pvRp["html_url"].(string); okph && ph != "" && !isOwnerPackagesURL(ph) {
							finalURL = ph
						}
					}
					if finalURL == "" {
						if rh, okh := rp["html_url"].(string); okh && rh != "" && !isOwnerPackagesURL(rh) {
							finalURL = rh
						}
					}
				}

				// fallback to pkg.html_url if available and not owner-level packages id
				if finalURL == "" {
					if purl, ok3 := pkg["html_url"].(string); ok3 && purl != "" && !isOwnerPackagesURL(purl) {
						finalURL = purl
					}
				}

				// If we still have no usable URL and this is a container package, prefer the repo-based container page
				if finalURL == "" {
					if ptype, okpt := pkg["package_type"].(string); okpt && strings.ToUpper(ptype) == "CONTAINER" {
						if repoObj, okrepo := payload["repository"].(map[string]any); okrepo {
							if full, okfull := repoObj["full_name"].(string); okfull && full != "" {
								finalURL = fmt.Sprintf("https://github.com/%s/pkgs/container/%s", full, pname)
							}
						}
					}
				}

				// Last resort: accept any available registry URL even if owner-level
				if finalURL == "" {
					if rp, okrp := payload["registry_package"].(map[string]any); okrp {
						if pvRp, okpv := rp["package_version"].(map[string]any); okpv {
							if ph, okph := pvRp["html_url"].(string); okph && ph != "" {
								finalURL = ph
							}
						}
						if finalURL == "" {
							if rh, okh := rp["html_url"].(string); okh && rh != "" {
								finalURL = rh
							}
						}
					}
				}

				if finalURL == "" {
					if purl, ok3 := pkg["html_url"].(string); ok3 && purl != "" {
						finalURL = purl
					}
				}

				if finalURL != "" {
					data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, finalURL)
				}
			}
		}
	}
	data["action"] = payload["action"]
}
