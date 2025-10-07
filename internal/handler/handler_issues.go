package handler

import (
	"fmt"
	"strings"
)

func prepareIssuesData(data map[string]any, payload map[string]any) {
	if issue, ok := payload["issue"].(map[string]any); ok {
		data["issue_number"] = issue["number"]
		data["issue_title"] = issue["title"]
		data["issue_url"] = issue["html_url"]
		data["issue_state"] = issue["state"]
		if stateReason, ok := issue["state_reason"].(string); ok {
			data["issue_state_reason"] = stateReason
		}
		data["issue_body"] = issue["body"]
		data["issue"] = issue

		// set issue user link, but avoid duplication if sender == issue.user
		if user, ok := issue["user"].(map[string]any); ok {
			if login, ok2 := user["login"].(string); ok2 {
				if url, ok3 := user["html_url"].(string); ok3 {
					// compare with sender (if available) to avoid duplicate display
					dup := false
					if sender, sok := payload["sender"].(map[string]any); sok {
						if slog, sok2 := sender["login"].(string); sok2 && slog == login {
							dup = true
						}
					}
					if !dup {
						data["issue_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					} else {
						data["issue_user_link_md"] = ""
					}
				}
			}
		}

		if iu, ok := issue["html_url"].(string); ok {
			if it, ok2 := issue["title"].(string); ok2 && it != "" {
				data["issue_link_md"] = fmt.Sprintf("[#%v %s](%s)", issue["number"], it, iu)
			} else {
				data["issue_link_md"] = iu
			}
		}

		issueTypeName := ""
		if tmap, ok := issue["type"].(map[string]any); ok {
			if name, ok2 := tmap["name"].(string); ok2 {
				issueTypeName = name
			}
		}
		if issueTypeName == "" {
			if tmap, ok := payload["type"].(map[string]any); ok {
				if name, ok2 := tmap["name"].(string); ok2 {
					issueTypeName = name
				}
			}
		}

		issueTypeNormalized := "unknown"
		if issueTypeName != "" {
			lower := strings.ToLower(issueTypeName)
			if strings.Contains(lower, "bug") {
				issueTypeNormalized = "bug"
			} else if strings.Contains(lower, "feature") {
				issueTypeNormalized = "feature"
			} else if strings.Contains(lower, "task") {
				issueTypeNormalized = "task"
			} else {
				issueTypeNormalized = lower
			}
		} else {
			if labels, ok := issue["labels"].([]any); ok {
				issueTypeNormalized = detectIssueTypeFromLabels(labels)
			}
		}

		// collect label names for display
		labelsJoined := ""
		if labelsArr, ok := issue["labels"].([]any); ok {
			var names []string
			for _, l := range labelsArr {
				if lm, lok := l.(map[string]any); lok {
					if lname, lok2 := lm["name"].(string); lok2 && lname != "" {
						names = append(names, lname)
					}
				}
			}
			if len(names) > 0 {
				labelsJoined = strings.Join(names, ", ")
			}
		}

		data["issue_type_name"] = issueTypeName
		data["issue_type"] = issueTypeNormalized
		data["issue_labels_joined"] = labelsJoined

		// build a display string for templates: prefer explicit type, then labels; hide if unknown and no labels
		display := ""
		if issueTypeNormalized != "unknown" && issueTypeNormalized != "" {
			display = fmt.Sprintf("(%s)", issueTypeNormalized)
		} else if labelsJoined != "" {
			display = fmt.Sprintf("(%s)", labelsJoined)
		}
		data["issue_type_display"] = display
	}

	// if this webhook included a single `label` (for labeled/unlabeled actions), surface it
	if lab, lok := payload["label"].(map[string]any); lok {
		if lname, lok2 := lab["name"].(string); lok2 && lname != "" {
			data["labeled_label_name"] = lname
			// if label has a URL, provide a markdown link
			if lurl, lok3 := lab["url"].(string); lok3 && lurl != "" {
				data["labeled_label_link_md"] = fmt.Sprintf("[%s](%s)", lname, lurl)
			}
		}
	}

	// Extract assignee information (for assigned/unassigned actions)
	if assignee, ok := payload["assignee"].(map[string]any); ok {
		if login, ok2 := assignee["login"].(string); ok2 && login != "" {
			data["assignee_login"] = login
			if url, ok3 := assignee["html_url"].(string); ok3 && url != "" {
				data["assignee_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
			}
		}
	}

	// Extract milestone information (for milestoned/demilestoned actions)
	if milestone, ok := payload["milestone"].(map[string]any); ok {
		if title, ok2 := milestone["title"].(string); ok2 && title != "" {
			data["milestone_title"] = title
			if url, ok3 := milestone["html_url"].(string); ok3 && url != "" {
				data["milestone_link_md"] = fmt.Sprintf("[%s](%s)", title, url)
			}
		}
	}

	// Extract changes information (for edited/transferred actions)
	if changes, ok := payload["changes"].(map[string]any); ok {
		// Title changes
		if titleChange, ok2 := changes["title"].(map[string]any); ok2 {
			if from, ok3 := titleChange["from"].(string); ok3 {
				data["changes_title_from"] = from
			}
		}
		// Repository transfer
		if repoChange, ok2 := changes["new_repository"].(map[string]any); ok2 {
			if fullName, ok3 := repoChange["full_name"].(string); ok3 {
				data["changes_new_repository_full_name"] = fullName
			}
		}
	}

	data["action"] = payload["action"]
}
