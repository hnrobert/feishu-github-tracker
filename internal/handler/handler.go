package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/config"
	"github.com/hnrobert/feishu-github-tracker/internal/matcher"
	"github.com/hnrobert/feishu-github-tracker/internal/notifier"
	"github.com/hnrobert/feishu-github-tracker/internal/template"
	"github.com/hnrobert/feishu-github-tracker/pkg/logger"
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

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.Error("Failed to parse JSON payload: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
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

func (h *Handler) processWebhook(eventType string, payload map[string]interface{}) error {
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

func (h *Handler) extractRepoFullName(payload map[string]interface{}) string {
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if fullName, ok := repo["full_name"].(string); ok {
			return fullName
		}
	}
	return ""
}

func (h *Handler) extractAction(payload map[string]interface{}) string {
	if action, ok := payload["action"].(string); ok {
		return action
	}
	return ""
}

func (h *Handler) extractRef(payload map[string]interface{}) string {
	if ref, ok := payload["ref"].(string); ok {
		return ref
	}
	// For pull requests, get the base branch
	if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
		if base, ok := pr["base"].(map[string]interface{}); ok {
			if ref, ok := base["ref"].(string); ok {
				return "refs/heads/" + ref
			}
		}
	}
	return ""
}

func (h *Handler) prepareTemplateData(eventType string, payload map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})

	// Common fields
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		data["repo_name"] = repo["name"]
		data["repo_full_name"] = repo["full_name"]
		data["repo_url"] = repo["html_url"]
	}

	if sender, ok := payload["sender"].(map[string]interface{}); ok {
		data["sender_name"] = sender["login"]
		data["sender_avatar"] = sender["avatar_url"]
		data["sender_url"] = sender["html_url"]
	}

	// Event-specific fields
	switch eventType {
	case "push":
		data["ref"] = payload["ref"]
		data["compare_url"] = payload["compare"]
		if commits, ok := payload["commits"].([]interface{}); ok {
			data["commits_count"] = len(commits)
			if len(commits) > 0 {
				if commit, ok := commits[0].(map[string]interface{}); ok {
					data["commit_message"] = commit["message"]
				}
			}
		}
		data["forced"] = payload["forced"]

	case "pull_request":
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
			data["pr_state"] = pr["state"]
			data["pr_merged"] = pr["merged"]
			data["pr_body"] = pr["body"]
			if head, ok := pr["head"].(map[string]interface{}); ok {
				data["pr_head_ref"] = head["ref"]
			}
			if base, ok := pr["base"].(map[string]interface{}); ok {
				data["pr_base_ref"] = base["ref"]
			}
		}
		data["action"] = payload["action"]

	case "issues":
		if issue, ok := payload["issue"].(map[string]interface{}); ok {
			data["issue_number"] = issue["number"]
			data["issue_title"] = issue["title"]
			data["issue_url"] = issue["html_url"]
			data["issue_state"] = issue["state"]
			data["issue_body"] = issue["body"]
		}
		data["action"] = payload["action"]

	case "issue_comment":
		if comment, ok := payload["comment"].(map[string]interface{}); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
		}
		if issue, ok := payload["issue"].(map[string]interface{}); ok {
			data["issue_number"] = issue["number"]
			data["issue_title"] = issue["title"]
			data["issue_url"] = issue["html_url"]
		}

	case "release":
		if release, ok := payload["release"].(map[string]interface{}); ok {
			data["release_name"] = release["name"]
			data["release_tag"] = release["tag_name"]
			data["release_url"] = release["html_url"]
			data["release_body"] = release["body"]
		}
		data["action"] = payload["action"]

	case "pull_request_review":
		if review, ok := payload["review"].(map[string]interface{}); ok {
			data["review_state"] = review["state"]
			data["review_body"] = review["body"]
			data["review_url"] = review["html_url"]
		}
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
		}

	case "pull_request_review_comment":
		if comment, ok := payload["comment"].(map[string]interface{}); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
		}
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			data["pr_number"] = pr["number"]
			data["pr_title"] = pr["title"]
			data["pr_url"] = pr["html_url"]
		}

	case "discussion":
		if discussion, ok := payload["discussion"].(map[string]interface{}); ok {
			data["discussion_title"] = discussion["title"]
			data["discussion_url"] = discussion["html_url"]
			data["discussion_body"] = discussion["body"]
		}
		data["action"] = payload["action"]

	case "discussion_comment":
		if comment, ok := payload["comment"].(map[string]interface{}); ok {
			data["comment_body"] = comment["body"]
			data["comment_url"] = comment["html_url"]
		}
		if discussion, ok := payload["discussion"].(map[string]interface{}); ok {
			data["discussion_title"] = discussion["title"]
			data["discussion_url"] = discussion["html_url"]
		}
	}

	return data
}
