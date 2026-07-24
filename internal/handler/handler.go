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
	config    *config.Config
	notifier  *notifier.Notifier
	hotReload bool
	configDir string
	// OnReload, if set, is invoked after a successful hot-reload of config (e.g.
	// to run file-normalization side effects). It must not panic.
	OnReload func(configDir string)
}

// New creates a new Handler
func New(cfg *config.Config, n *notifier.Notifier) *Handler {
	return &Handler{
		config:    cfg,
		notifier:  n,
		hotReload: false,
		configDir: "",
	}
}

// EnableHotReload enables configuration hot reload on each webhook request
func (h *Handler) EnableHotReload(configDir string) {
	h.hotReload = true
	h.configDir = configDir
	logger.Info("Hot reload enabled for config directory: %s", configDir)
}

// Reload re-reads the configuration from disk and swaps it into the handler
// (rebuilding the notifier and running the OnReload hook). It is called on each
// webhook when hot reload is enabled, and also by the management panel after a
// configuration edit so that changes take effect immediately without a restart.
func (h *Handler) Reload() {
	if h.configDir == "" {
		return
	}
	logger.Debug("Reloading configuration from %s", h.configDir)
	cfg, err := config.Load(h.configDir)
	if err != nil {
		logger.Error("Failed to reload configuration: %v", err)
		return
	}
	changed := false
	if h.config != nil {
		oldB, _ := json.Marshal(h.config)
		newB, _ := json.Marshal(cfg)
		if string(oldB) != string(newB) {
			logger.Info("Configuration changes detected, applying new configuration")
			changed = true
		}
	} else {
		logger.Info("Configuration loaded")
		changed = true
	}

	h.config = cfg
	h.notifier = notifier.New(cfg.FeishuBots)

	if h.OnReload != nil {
		h.OnReload(h.configDir)
	}

	if !changed {
		logger.Debug("Configuration reloaded successfully (no changes detected)")
	}
}

// ServeHTTP handles incoming webhook requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Hot reload configuration if enabled
	if h.hotReload && h.configDir != "" {
		h.Reload()
	}

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

	logger.Debug("Received %s event", eventType)
	logger.Debug("Payload: %v", payload)

	// Verify signature. The signing secret is resolved per-request: the global
	// server.secret plus any secret configured on the repo/org rule this
	// webhook matches (so each GitHub-side webhook can use its own secret). If
	// no secret is configured anywhere, verification is skipped (as before).
	secrets := h.candidateSecrets(payload)
	if len(secrets) > 0 {
		if !h.verifySignatureAny(r.Header.Get("X-Hub-Signature-256"), body, secrets) {
			logger.Warn("Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Process the webhook
	if err := h.processWebhook(eventType, payload); err != nil {
		logger.Error("Failed to process webhook: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// candidateSecrets returns the webhook signing secrets that may apply to this
// request: the global server.secret, plus any secret configured on the repo (or
// org) rule(s) the payload matches. Deduplicated. Empty (and thus no signature
// verification) when no secret is configured anywhere.
func (h *Handler) candidateSecrets(payload map[string]any) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	add(h.config.Server.Server.Secret)

	if repo := h.extractRepoFullName(payload); repo != "" {
		if rules, err := h.matchRepositoryRules(repo); err == nil {
			for _, rule := range rules {
				add(rule.Secret)
			}
		}
	} else if org := h.extractOrgName(payload); org != "" {
		for _, rp := range h.config.Repos.Repos {
			if rp.Pattern == org+"/*" {
				add(rp.Secret)
			}
		}
	}
	return out
}

// matchRepositoryRules returns either the first matching rule (the historical
// default) or every matching rule when match_all_rules is enabled.
func (h *Handler) matchRepositoryRules(fullName string) ([]*config.RepoPattern, error) {
	if h.config.Server.Server.MatchAllRules {
		return matcher.MatchAllRepos(fullName, h.config.Repos.Repos)
	}

	rule, err := matcher.MatchRepo(fullName, h.config.Repos.Repos)
	if err != nil || rule == nil {
		return nil, err
	}
	return []*config.RepoPattern{rule}, nil
}

// verifySignatureAny reports whether the X-Hub-Signature-256 header matches the
// body for any of the provided secrets (HMAC-SHA256, constant-time compare).
func (h *Handler) verifySignatureAny(signature string, body []byte, secrets []string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	receivedMAC := []byte(strings.TrimPrefix(signature, "sha256="))
	for _, secret := range secrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedMAC := []byte(hex.EncodeToString(mac.Sum(nil)))
		if hmac.Equal(receivedMAC, expectedMAC) {
			return true
		}
	}
	return false
}

func (h *Handler) processWebhook(eventType string, payload map[string]any) error {
	// Extract repository full name (may be empty for org-level webhooks or certain events)
	repoFullName := h.extractRepoFullName(payload)

	// Extract organization name (for org-level webhooks)
	orgName := h.extractOrgName(payload)
	if repoFullName != "" && h.config.Server.Server.MatchAllRules {
		rules, err := matcher.MatchAllRepos(repoFullName, h.config.Repos.Repos)
		if err != nil {
			return fmt.Errorf("failed to match repository: %w", err)
		}
		if len(rules) == 0 {
			logger.Debug("No matching repository pattern found for %s, skipping", repoFullName)
			return nil
		}
		return h.processAllRepositoryRules(eventType, payload, rules)
	}

	// Determine target bots based on repository or organization
	var repoPattern *config.RepoPattern
	var err error
	var targetBots []string

	if repoFullName != "" {
		// Repository-level webhook
		logger.Debug("Processing %s event for repository: %s", eventType, repoFullName)

		repoPattern, err = matcher.MatchRepo(repoFullName, h.config.Repos.Repos)
		if err != nil {
			return fmt.Errorf("failed to match repository: %w", err)
		}
		if repoPattern == nil {
			logger.Debug("No matching repository pattern found for %s, skipping", repoFullName)
			return nil
		}

		logger.Debug("Matched repository pattern: %s", repoPattern.Pattern)
		targetBots = repoPattern.NotifyTo

	} else if orgName != "" {
		// Organization-level webhook
		logger.Debug("Processing %s event for organization: %s", eventType, orgName)

		// Find all repo patterns matching this organization (exact match for org/*)
		for _, repo := range h.config.Repos.Repos {
			if repo.Pattern == orgName+"/*" {
				targetBots = append(targetBots, repo.NotifyTo...)
			}
		}

		if len(targetBots) == 0 {
			logger.Debug("No matching repository patterns found for organization %s (expected pattern: %s/*), skipping", orgName, orgName)
			return nil
		}

		// Remove duplicates
		targetBots = uniqueStrings(targetBots)

	} else {
		// No repository or organization info
		logger.Warn("Event %s does not contain repository or organization information, skipping", eventType)
		return nil
	}

	// For ping events, skip filter and send to all matched bots
	isPingEvent := (eventType == "ping")

	if !isPingEvent {
		// For non-ping events, check event filter
		if repoPattern == nil {
			logger.Warn("Event %s requires event filtering but no repo pattern matched", eventType)
			return nil
		}

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
			logger.Debug("Event %s (action: %s, ref: %s) does not match configured events, skipping", eventType, action, ref)
			return nil
		}
	} else {
		logger.Debug("Ping event - bypassing filter, will notify all matched bots")
	}

	logger.Info("Event matched: %s, sending notification", eventType)
	return h.sendNotification(eventType, payload, targetBots)
}

// processAllRepositoryRules evaluates every matching rule in configuration
// order. A rule that does not subscribe to this event is skipped; failures in
// one eligible rule do not prevent later eligible rules from being attempted.
func (h *Handler) processAllRepositoryRules(eventType string, payload map[string]any, rules []*config.RepoPattern) error {
	isPingEvent := eventType == "ping"
	action := h.extractAction(payload)
	ref := h.extractRef(payload)
	seenTargets := make(map[string]struct{})
	var errs []string

	for _, rule := range rules {
		logger.Debug("Matched repository pattern: %s", rule.Pattern)
		if !isPingEvent {
			expandedEvents := matcher.ExpandEvents(rule.Events, h.config.Events.EventSets, h.config.Events.Events)
			if !matcher.MatchEvent(eventType, action, ref, payload, expandedEvents) {
				logger.Debug("Event %s (action: %s, ref: %s) does not match rule %s, skipping", eventType, action, ref, rule.Pattern)
				continue
			}
		}

		targets := uniqueUnseenTargets(rule.NotifyTo, seenTargets)
		if len(targets) == 0 {
			logger.Debug("Rule %s has no new notification targets, skipping", rule.Pattern)
			continue
		}

		logger.Info("Event matched: %s (rule: %s), sending notification", eventType, rule.Pattern)
		if err := h.sendNotification(eventType, payload, targets); err != nil {
			logger.Error("Failed to send notifications for rule %s: %v", rule.Pattern, err)
			errs = append(errs, fmt.Sprintf("rule %s: %v", rule.Pattern, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to process some matching rules: %s", strings.Join(errs, "; "))
	}
	return nil
}

// uniqueUnseenTargets prevents duplicate sends when overlapping matching rules
// contain the same configured target.
func uniqueUnseenTargets(targets []string, seen map[string]struct{}) []string {
	var result []string
	for _, target := range targets {
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		result = append(result, target)
	}
	return result
}

func (h *Handler) sendNotification(eventType string, payload map[string]any, targets []string) error {
	tags := template.DetermineTags(eventType, payload)
	data := h.prepareTemplateData(eventType, payload)
	targetsByTemplate := h.groupTargetsByTemplate(targets)
	var errs []string
	for templateName, templateTargets := range targetsByTemplate {
		logger.Debug("Processing %d target(s) with template: %s", len(templateTargets), templateName)
		templatesConfig := h.config.GetTemplateConfig(templateName)
		tmpl, err := template.SelectTemplate(eventType, tags, templatesConfig)
		if err != nil {
			logger.Error("Failed to select template for %s: %v", templateName, err)
			errs = append(errs, fmt.Sprintf("template %s: %v", templateName, err))
			continue
		}
		filledPayload, err := template.FillTemplate(tmpl, data)
		if err != nil {
			logger.Error("Failed to fill template for %s: %v", templateName, err)
			errs = append(errs, fmt.Sprintf("template %s: %v", templateName, err))
			continue
		}
		if err := h.notifier.Send(templateTargets, filledPayload); err != nil {
			logger.Error("Failed to send notifications for template %s: %v", templateName, err)
			errs = append(errs, fmt.Sprintf("template %s: %v", templateName, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to process some templates: %s", strings.Join(errs, "; "))
	}
	return nil
}

// groupTargetsByTemplate groups notification targets by their template preference
func (h *Handler) groupTargetsByTemplate(targets []string) map[string][]string {
	result := make(map[string][]string)

	for _, target := range targets {
		templateName := h.config.GetBotTemplate(target)
		result[templateName] = append(result[templateName], target)
	}

	return result
}

func (h *Handler) extractRepoFullName(payload map[string]any) string {
	if repo, ok := payload["repository"].(map[string]any); ok {
		if fullName, ok := repo["full_name"].(string); ok {
			return fullName
		}
	}
	return ""
}

func (h *Handler) extractOrgName(payload map[string]any) string {
	if org, ok := payload["organization"].(map[string]any); ok {
		if login, ok := org["login"].(string); ok {
			return login
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

	// populate common fields shared across event types (repo, sender, org,
	// installation, action)
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
