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
				// Check pkg["package_version"] instead of payload["package_version"]
				if pv, okpv := pkg["package_version"].(map[string]any); okpv {
					if vname, vok := pv["version"].(string); vok && vname != "" {
						pkg["version"] = vname
						// default to version (often a digest)
						pkg["tag_name"] = vname
						data["package_version_name"] = vname
						data["package_version"] = vname
					}
					if purl, okurl := pv["html_url"].(string); okurl && purl != "" {
						pkg["html_url"] = purl
					}
					// For container packages, a human-friendly tag may be present in container_metadata.tag.name
					if cm, okcm := pv["container_metadata"].(map[string]any); okcm {
						if tagObj, okt := cm["tag"].(map[string]any); okt {
							if tname, tokk := tagObj["name"].(string); tokk && tname != "" {
								pkg["tag_name"] = tname
								data["package_tag_name"] = tname
							}
						}
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
						// extract container tag name from registry_package.package_version.container_metadata.tag.name when available
						if cm, okcm := pvRp["container_metadata"].(map[string]any); okcm {
							if tagObj, okt := cm["tag"].(map[string]any); okt {
								if tname, tokk := tagObj["name"].(string); tokk && tname != "" {
									pkg["tag_name"] = tname
									data["package_tag_name"] = tname
								}
							}
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

				// Prepare tag display: if there's a tag, present it as a markdown link to the
				// best known URL for that tag/version. If there's no tag, do not set
				// package_tag_name (templates can treat it as null/absent).
				if tval, okt := pkg["tag_name"].(string); okt && tval != "" {
					// prefer package_version.html_url or finalURL as the target for the tag link
					target := ""
					// check pkg.package_version (from payload.package.package_version)
					if pv, okpv := pkg["package_version"].(map[string]any); okpv {
						if ph, okph := pv["html_url"].(string); okph && ph != "" {
							target = ph
						}
					}
					// if still empty, prefer registry_package.package_version.html_url
					if target == "" {
						if rp, okrp := payload["registry_package"].(map[string]any); okrp {
							if pvRp, okpv := rp["package_version"].(map[string]any); okpv {
								if ph, okph := pvRp["html_url"].(string); okph && ph != "" {
									target = ph
								}
							}
						}
					}
					// if still empty, use finalURL (which may point to package or repo page)
					if target == "" {
						target = finalURL
					}
					// if still empty and we have repo info, construct a repo container page with tag query
					if target == "" {
						if repoObj, okrepo := payload["repository"].(map[string]any); okrepo {
							if full, okfull := repoObj["full_name"].(string); okfull && full != "" {
								// escape tag name for URL
								imported := tval
								// avoid importing net/url at top multiple times; build simple escaped string
								// use QueryEscape for safety
								target = fmt.Sprintf("https://github.com/%s/pkgs/container/%s?tag=%s", full, pname, strings.ReplaceAll(imported, " ", "%20"))
							}
						}
					}
					// finally set the markdown link; if no target, set plain tag string
					if target != "" {
						data["package_tag_name"] = fmt.Sprintf("[%s](%s)", tval, target)
					} else {
						data["package_tag_name"] = tval
					}
				} else {
					// no tag found: ensure templates see it as absent / None
					delete(data, "package_tag_name")
				}
			}
		}
	}
	data["action"] = payload["action"]
}
