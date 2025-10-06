package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/logger"
	"github.com/hnrobert/feishu-github-tracker/internal/matcher"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
	"github.com/hnrobert/feishu-github-tracker/internal/template"
)

// Handler handles GitHub webhook requests
type Handler struct {
	config   *config.Config
	notifier *notifier.Notifier
}

// New creates a new Handler
func New(cfg *config.Config, n *notifier.Notifier) *Handler {
	return &Handler{
		config:   cfg,
		notifier: n,
	}
}

// ServeHTTP handles incoming webhook requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature if secret is configured
	if h.config.Server.Server.Secret != "" {
		if !h.verifySignature(r.Header.Get("X-Hub-Signature-256"), body) {
			logger.Warn("Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Get event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		logger.Warn("Missing X-GitHub-Event header")
		http.Error(w, "Missing X-GitHub-Event header", http.StatusBadRequest)
		return
	}

	// Parse payload based on content type
	var payload map[string]any
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// GitHub form-encoded webhook payload is in the "payload" form field
		// Parse the form data from the body
		values, err := url.ParseQuery(string(body))
		if err != nil {
			logger.Error("Failed to parse form data: %v", err)
			http.Error(w, "Failed to parse form data", http.StatusBadRequest)
			return
		}

		payloadStr := values.Get("payload")
		if payloadStr == "" {
			logger.Error("Missing payload field in form data")
			http.Error(w, "Missing payload field", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			logger.Error("Failed to parse JSON payload from form: %v", err)
			http.Error(w, "Invalid JSON in payload field", http.StatusBadRequest)
			return
		}
	} else {
		// Default to JSON parsing
		if err := json.Unmarshal(body, &payload); err != nil {
			logger.Error("Failed to parse JSON payload: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	logger.Info("Received %s event", eventType)
	logger.Debug("Payload: %v", payload)

	// Process the webhook
	if err := h.processWebhook(eventType, payload); err != nil {
		logger.Error("Failed to process webhook: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) verifySignature(signature string, body []byte) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.config.Server.Server.Secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	receivedMAC := strings.TrimPrefix(signature, "sha256=")

	return hmac.Equal([]byte(receivedMAC), []byte(expectedMAC))
}

func (h *Handler) processWebhook(eventType string, payload map[string]any) error {
	// Extract repository full name
	repoFullName := h.extractRepoFullName(payload)
	if repoFullName == "" {
		return fmt.Errorf("failed to extract repository name from payload")
	}

	logger.Info("Processing event for repository: %s", repoFullName)

	// Match repository pattern
	repoPattern, err := matcher.MatchRepo(repoFullName, h.config.Repos.Repos)
	if err != nil {
		return fmt.Errorf("failed to match repository: %w", err)
	}
	if repoPattern == nil {
		logger.Info("No matching repository pattern found for %s, skipping", repoFullName)
		return nil
	}

	logger.Info("Matched repository pattern: %s", repoPattern.Pattern)

	// Expand events (resolve templates)
	expandedEvents := matcher.ExpandEvents(
		repoPattern.Events,
		h.config.Events.EventSets,
		h.config.Events.Events,
	)

	// Extract event details
	action := h.extractAction(payload)
	ref := h.extractRef(payload)

	// Match event
	if !matcher.MatchEvent(eventType, action, ref, payload, expandedEvents) {
		logger.Info("Event %s (action: %s, ref: %s) does not match configured events, skipping", eventType, action, ref)
		return nil
	}

	logger.Info("Event matched, preparing notification")

	// Determine tags for template selection
	tags := template.DetermineTags(eventType, payload)

	// Select template
	tmpl, err := template.SelectTemplate(eventType, tags, h.config.Templates)
	if err != nil {
		return fmt.Errorf("failed to select template: %w", err)
	}

	// Prepare data for template filling
	data := h.prepareTemplateData(eventType, payload)

	// Fill template
	filledPayload, err := template.FillTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("failed to fill template: %w", err)
	}

	// Send notifications
	if err := h.notifier.Send(repoPattern.NotifyTo, filledPayload); err != nil {
		return fmt.Errorf("failed to send notifications: %w", err)
	}

	return nil
}

func (h *Handler) extractRepoFullName(payload map[string]any) string {
	if repo, ok := payload["repository"].(map[string]any); ok {
		if fullName, ok := repo["full_name"].(string); ok {
			return fullName
		}
	}
	return ""
}

func (h *Handler) extractAction(payload map[string]any) string {
	if action, ok := payload["action"].(string); ok {
		return action
	}
	return ""
}

func (h *Handler) extractRef(payload map[string]any) string {
	if ref, ok := payload["ref"].(string); ok {
		return ref
	}
	// For pull requests, get the base branch
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		if base, ok := pr["base"].(map[string]any); ok {
			if ref, ok := base["ref"].(string); ok {
				return "refs/heads/" + ref
			}
		}
	}
	return ""
}

func (h *Handler) prepareTemplateData(eventType string, payload map[string]any) map[string]any {
	data := make(map[string]any)

	// Common fields
	if repo, ok := payload["repository"].(map[string]any); ok {
		data["repo_name"] = repo["name"]
		data["repo_full_name"] = repo["full_name"]
		data["repo_url"] = repo["html_url"]
		// Provide nested object for templates that use {{repository.full_name}} style
		data["repository"] = repo

		// repository link (Markdown)
		if full, ok := repo["full_name"].(string); ok {
			if url, ok2 := repo["html_url"].(string); ok2 {
				data["repository_link_md"] = fmt.Sprintf("[%s](%s)", full, url)
			}
		}
	}

	if sender, ok := payload["sender"].(map[string]any); ok {
		data["sender_name"] = sender["login"]
		data["sender_avatar"] = sender["avatar_url"]
		data["sender_url"] = sender["html_url"]
		// Provide nested object for templates that use {{sender.login}} style
		data["sender"] = sender

		// sender markdown link
		if login, ok := sender["login"].(string); ok {
			if surl, ok2 := sender["html_url"].(string); ok2 {
				data["sender_link_md"] = fmt.Sprintf("[%s](%s)", login, surl)
			}
		}
	}

	// Event-specific fields
	switch eventType {
	case "push":
		data["ref"] = payload["ref"]
		data["compare_url"] = payload["compare"]
		// include commits list and its count for templates
		if commits, ok := payload["commits"].([]any); ok {
			data["commits_count"] = len(commits)
			data["commits"] = commits

			// collect messages and authors
			var msgs []string
			var authors []string
			var authorsWithLinks []string
			for _, c := range commits {
				if cm, ok := c.(map[string]any); ok {
					if m, ok := cm["message"].(string); ok {
						msgs = append(msgs, m)
					}
					if author, ok := cm["author"].(map[string]any); ok {
						if name, ok := author["name"].(string); ok {
							authors = append(authors, name)
							// try to build a GitHub profile link from username if available
							if uname, ok := author["username"].(string); ok && uname != "" {
								authorsWithLinks = append(authorsWithLinks, fmt.Sprintf("[%s](https://github.com/%s)", name, uname))
							} else {
								authorsWithLinks = append(authorsWithLinks, name)
							}
						}
					}
				}
			}

			if len(msgs) > 0 {
				data["commit_messages"] = msgs
				// a single string with each message on a new line, prefixed
				joined := ""
				for i, m := range msgs {
					if i == 0 {
						joined = m
					} else {
						joined = joined + "\n- " + m
					}
				}
				data["commit_messages_joined"] = joined
				// also expose first commit message for backward compatibility
				data["commit_message"] = msgs[0]
			}
			if len(authors) > 0 {
				data["commit_authors"] = authors
				data["commit_authors_joined"] = strings.Join(authors, ", ")
				if len(authorsWithLinks) > 0 {
					data["commit_authors_with_links"] = authorsWithLinks
					data["commit_authors_with_links_joined"] = strings.Join(authorsWithLinks, ", ")
				}
			}
		}

		// include pusher object so templates can reference {{pusher.name}}
		if pusher, ok := payload["pusher"].(map[string]any); ok {
			data["pusher"] = pusher

			// pusher markdown link (pusher.name is usually username)
			if pname, ok := pusher["name"].(string); ok {
				// assume name is GitHub username
				data["pusher_link_md"] = fmt.Sprintf("[%s](https://github.com/%s)", pname, pname)
			}
		}
		data["forced"] = payload["forced"]

		// branch link
		if ref, ok := payload["ref"].(string); ok {
			branch := strings.TrimPrefix(ref, "refs/heads/")
			data["branch_name"] = branch
			if repo, ok := payload["repository"].(map[string]any); ok {
				if url, ok2 := repo["html_url"].(string); ok2 {
					data["branch_url"] = fmt.Sprintf("%s/tree/%s", url, branch)
					data["branch_link_md"] = fmt.Sprintf("[%s](%s/tree/%s)", branch, url, branch)
				}
			}
		}

	case "pull_request":
		if pr, ok := payload["pull_request"].(map[string]any); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
			data["pr_state"] = pr["state"]
			data["pr_merged"] = pr["merged"]
			data["pr_body"] = pr["body"]
			// Provide nested object
			data["pull_request"] = pr
			// pr author link
			if user, ok := pr["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["pr_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
			if head, ok := pr["head"].(map[string]any); ok {
				data["pr_head_ref"] = head["ref"]
				// pr head branch link
				if repo, ok := payload["repository"].(map[string]any); ok {
					if url, ok2 := repo["html_url"].(string); ok2 {
						if href, ok3 := head["ref"].(string); ok3 {
							data["pr_head_branch_link_md"] = fmt.Sprintf("[%s](%s/tree/%s)", href, url, href)
						}
					}
				}
			}
			if base, ok := pr["base"].(map[string]any); ok {
				data["pr_base_ref"] = base["ref"]
				// pr base branch link
				if repo, ok := payload["repository"].(map[string]any); ok {
					if url, ok2 := repo["html_url"].(string); ok2 {
						if bref, ok3 := base["ref"].(string); ok3 {
							data["pr_base_branch_link_md"] = fmt.Sprintf("[%s](%s/tree/%s)", bref, url, bref)
						}
					}
				}
			}
		}
		data["action"] = payload["action"]

	case "issues":
		if issue, ok := payload["issue"].(map[string]any); ok {
			data["issue_number"] = issue["number"]
			data["issue_title"] = issue["title"]
			data["issue_url"] = issue["html_url"]
			data["issue_state"] = issue["state"]
			data["issue_body"] = issue["body"]
			data["issue"] = issue
			// issue author link
			if user, ok := issue["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["issue_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}

			// issue markdown link (title -> url)
			if iu, ok := issue["html_url"].(string); ok {
				if it, ok2 := issue["title"].(string); ok2 && it != "" {
					data["issue_link_md"] = fmt.Sprintf("[#%v %s](%s)", issue["number"], it, iu)
				} else {
					data["issue_link_md"] = iu
				}
			}

			// determine issue type for templates (try explicit type object, then payload.type, then labels)
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

			// fallback: inspect labels
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

			data["issue_type_name"] = issueTypeName
			data["issue_type"] = issueTypeNormalized
		}
		data["action"] = payload["action"]

	case "issue_comment":
		if comment, ok := payload["comment"].(map[string]any); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
			data["comment"] = comment
			// comment author link
			if user, ok := comment["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["comment_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
		}
		if issue, ok := payload["issue"].(map[string]any); ok {
			data["issue_number"] = issue["number"]
			data["issue_title"] = issue["title"]
			data["issue_url"] = issue["html_url"]
			data["issue"] = issue
		}

	case "release":
		if release, ok := payload["release"].(map[string]any); ok {
			data["release_name"] = release["name"]
			data["release_tag"] = release["tag_name"]
			data["release_url"] = release["html_url"]
			data["release_body"] = release["body"]
			data["release"] = release
		}
		data["action"] = payload["action"]

	case "pull_request_review":
		if review, ok := payload["review"].(map[string]any); ok {
			data["review_state"] = review["state"]
			data["review_body"] = review["body"]
			data["review_url"] = review["html_url"]
			data["review"] = review
			// review author link
			if user, ok := review["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["review_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
		}
		if pr, ok := payload["pull_request"].(map[string]any); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
			data["pull_request"] = pr
		}

	case "pull_request_review_comment":
		if comment, ok := payload["comment"].(map[string]any); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
			// comment author link
			if user, ok := comment["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["comment_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
		}
		if pr, ok := payload["pull_request"].(map[string]any); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
		}

	case "discussion":
		if discussion, ok := payload["discussion"].(map[string]any); ok {
			data["discussion_title"] = discussion["title"]
			data["discussion_url"] = discussion["html_url"]
			data["discussion_body"] = discussion["body"]
			// discussion author link
			if user, ok := discussion["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["discussion_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
		}
		data["action"] = payload["action"]

	case "discussion_comment":
		if comment, ok := payload["comment"].(map[string]any); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
			data["comment"] = comment
			// comment author link
			if user, ok := comment["user"].(map[string]any); ok {
				if login, ok2 := user["login"].(string); ok2 {
					if url, ok3 := user["html_url"].(string); ok3 {
						data["comment_user_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
					}
				}
			}
		}
		if discussion, ok := payload["discussion"].(map[string]any); ok {
			data["discussion_title"] = discussion["title"]
			data["discussion_url"] = discussion["html_url"]
			data["discussion"] = discussion
		}
	}

	// package event
	if eventType == "package" {
		if pkg, ok := payload["package"].(map[string]any); ok {
			data["package"] = pkg
			if name, ok := pkg["name"]; ok {
				data["package_name"] = name
				// package_version may contain version/tag and uploader info; if present merge useful fields
				if pname, ok2 := name.(string); ok2 {
					// prefer package_version html_url when available
					if pv, okpv := payload["package_version"].(map[string]any); okpv {
						// version name
						if vname, vok := pv["version"].(string); vok && vname != "" {
							// set into package map so templates can use {{package.version}}
							pkg["version"] = vname
							pkg["tag_name"] = vname
							data["package_version_name"] = vname
						}
						// prefer html_url from package_version
						if purl, okurl := pv["html_url"].(string); okurl && purl != "" {
							pkg["html_url"] = purl
						}

						// uploader / publisher info
						if up, okuk := pv["uploader"].(map[string]any); okuk {
							if login, lok := up["login"].(string); lok {
								if url, uok := up["html_url"].(string); uok {
									data["package_publisher_link_md"] = fmt.Sprintf("[%s](%s)", login, url)
									// if sender_link_md wasn't provided, populate it from uploader
									if _, exists := data["sender_link_md"]; !exists {
										data["sender_link_md"] = data["package_publisher_link_md"]
									}
								}
							}
						}
						// also check common alternate fields
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

					// copy common package fields if present
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

					// now build package link markdown preferring updated pkg["html_url"]
					// prefer registry_package.html_url when available (GitHub Packages event payload)
					if rp, okrp := payload["registry_package"].(map[string]any); okrp {
						if rh, okh := rp["html_url"].(string); okh && rh != "" {
							data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, rh)
						} else if pv, okpv := rp["package_version"].(map[string]any); okpv {
							if ph, okph := pv["html_url"].(string); okph && ph != "" {
								data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, ph)
							}
						}
					} else if purl, ok3 := pkg["html_url"].(string); ok3 && purl != "" {
						data["package_link_md"] = fmt.Sprintf("[%s](%s)", pname, purl)
					}
				}
			}
		}
		data["action"] = payload["action"]
	}

	return data
}

// detectIssueTypeFromLabels inspects issue labels and returns a normalized type
// (bug/feature/task/unknown). This mirrors the logic in template.getIssueType
// but is duplicated here to avoid package cycles.
func detectIssueTypeFromLabels(labels []any) string {
	for _, label := range labels {
		if labelMap, ok := label.(map[string]any); ok {
			if name, ok := labelMap["name"].(string); ok {
				lowerName := strings.ToLower(name)
				if strings.Contains(lowerName, "bug") {
					return "bug"
				}
				if strings.Contains(lowerName, "feature") {
					return "feature"
				}
				if strings.Contains(lowerName, "task") {
					return "task"
				}
			}
		}
	}
	return "unknown"
}
