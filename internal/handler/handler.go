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

	// populate common fields shared across event types
	prepareCommonData(data, payload)

	// delegate per-event handling into separate files for clarity
	switch eventType {
	case "push":
		preparePushData(data, payload)
	case "pull_request":
		preparePullRequestData(data, payload)
	case "pull_request_review":
		preparePullRequestReviewData(data, payload)
	case "pull_request_review_comment":
		preparePullRequestReviewCommentData(data, payload)
	case "issues":
		prepareIssuesData(data, payload)
	case "issue_comment":
		prepareIssueCommentData(data, payload)
	case "discussion":
		prepareDiscussionData(data, payload)
	case "discussion_comment":
		prepareDiscussionCommentData(data, payload)
	case "release":
		prepareReleaseData(data, payload)
	case "package":
		preparePackageData(data, payload)
	case "gollum":
		prepareGollumData(data, payload)
	case "create":
		prepareCreateData(data, payload)
	case "delete":
		prepareDeleteData(data, payload)
	case "fork":
		prepareForkData(data, payload)
	case "star":
		prepareStarData(data, payload)
	case "repository":
		prepareRepositoryData(data, payload)
	case "deployment_status":
		prepareDeploymentStatusData(data, payload)
	case "project_card":
		prepareProjectCardData(data, payload)
	case "page_build":
		preparePageBuildData(data, payload)
	case "team":
		prepareTeamData(data, payload)
	case "watch":
		prepareWatchData(data, payload)
	case "deployment":
		prepareDeploymentData(data, payload)
	case "project":
		prepareProjectData(data, payload)
	case "project_column":
		prepareProjectColumnData(data, payload)
	case "milestone":
		prepareMilestoneData(data, payload)
	case "membership":
		prepareMembershipData(data, payload)
	case "member":
		prepareMemberData(data, payload)
	case "organization":
		prepareOrganizationData(data, payload)
	case "check_run":
		prepareCheckRunData(data, payload)
	case "check_suite":
		prepareCheckSuiteData(data, payload)
	case "commit_comment":
		prepareCommitCommentData(data, payload)
	case "deploy_key":
		prepareDeployKeyData(data, payload)
	case "code_scanning_alert":
		prepareCodeScanningAlertData(data, payload)
	case "dependabot_alert":
		prepareDependabotAlertData(data, payload)
	case "secret_scanning_alert":
		prepareSecretScanningAlertData(data, payload)
	case "repository_import":
		prepareRepositoryImportData(data, payload)
	case "repository_ruleset":
		prepareRepositoryRulesetData(data, payload)
	case "repository_vulnerability_alert":
		prepareRepositoryVulnerabilityAlertData(data, payload)
	case "label":
		prepareLabelData(data, payload)
	case "branch_protection_configuration":
		prepareBranchProtectionConfigurationData(data, payload)
	case "branch_protection_rule":
		prepareBranchProtectionRuleData(data, payload)
	case "custom_property":
		prepareCustomPropertyData(data, payload)
	case "custom_property_values":
		prepareCustomPropertyValuesData(data, payload)
	case "deployment_protection_rule":
		prepareDeploymentProtectionRuleData(data, payload)
	case "deployment_review":
		prepareDeploymentReviewData(data, payload)
	case "github_app_authorization":
		prepareGitHubAppAuthorizationData(data, payload)
	case "installation":
		prepareInstallationData(data, payload)
	case "installation_repositories":
		prepareInstallationRepositoriesData(data, payload)
	case "installation_target":
		prepareInstallationTargetData(data, payload)
	case "issue_dependencies":
		prepareIssueDependenciesData(data, payload)
	case "marketplace_purchase":
		prepareMarketplacePurchaseData(data, payload)
	case "merge_group":
		prepareMergeGroupData(data, payload)
	case "meta":
		prepareMetaData(data, payload)
	case "org_block":
		prepareOrgBlockData(data, payload)
	case "registry_package":
		prepareRegistryPackageData(data, payload)
	case "repository_advisory":
		prepareRepositoryAdvisoryData(data, payload)
	case "repository_dispatch":
		prepareRepositoryDispatchData(data, payload)
	case "secret_scanning_alert_location":
		prepareSecretScanningAlertLocationData(data, payload)
	case "secret_scanning_scan":
		prepareSecretScanningScanData(data, payload)
	case "security_and_analysis":
		prepareSecurityAndAnalysisData(data, payload)
	case "sponsorship":
		prepareSponsorshipData(data, payload)
	case "sub_issues":
		prepareSubIssuesData(data, payload)
	case "team_add":
		prepareTeamAddData(data, payload)
	case "projects_v2":
		prepareProjectsV2Data(data, payload)
	case "projects_v2_item":
		prepareProjectsV2ItemData(data, payload)
	case "projects_v2_status_update":
		prepareProjectsV2StatusUpdateData(data, payload)
	case "pull_request_review_thread":
		preparePullRequestReviewThreadData(data, payload)
	case "workflow_dispatch":
		prepareWorkflowDispatchData(data, payload)
	case "workflow_job":
		prepareWorkflowJobData(data, payload)
	case "personal_access_token_request":
		preparePersonalAccessTokenRequestData(data, payload)
	case "ping":
		preparePingData(data, payload)
	case "workflow_run":
		prepareWorkflowRunData(data, payload)
	case "status":
		prepareStatusData(data, payload)
	case "public":
		preparePublicData(data, payload)
	case "security_advisory":
		prepareSecurityAdvisoryData(data, payload)
	default:
		// unknown event types: nothing extra to do
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
