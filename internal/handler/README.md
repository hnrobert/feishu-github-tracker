# Handler data map

This file documents the top-level keys the handler populates in the `data` map used by templates.

Keys are grouped into families. Each family has a `Common` subsection listing keys shared across the family's events. Event sections only list keys that are specific to that event (i.e., not repeated in the family's Common list). Keys are alphabetized inside each list.

## Tagging and template selection (overview)

Templates are chosen by matching a set of tags. Each template payload in `configs/templates*.jsonc` carries a `tags` list (examples: `tags: [issue, type:feature]`, `tags: [push, force]`, or `tags: [pr, closed, merged]`).

- What `tags` mean: the first element is typically the event family (e.g. `issue`, `push`, `pr`). Additional elements refine the selection (for example `type:feature` narrows issue templates to feature-type issues).
- Why we have `tags: [issue, type:feature]`: the handler detects an issue's semantic type (bug/feature/task/unknown) from either a `type` field in the payload or from labels (see `detectIssueTypeFromLabels` in the handler). When an issue is classified as a feature, templates tagged with `type:feature` become selectable so you can show a different card (color, wording, actions) for feature requests.
  Where tags come from:
  - static template configuration (`configs/templates*.yaml`) — authors declare the tags they want a payload to match.
  - runtime tag detection in code — the handler and template selection helpers derive tags from payload shape and content (for example: `push` vs `push+force`, PR `closed` + `merged` vs `closed` + `unmerged`, issue `type:...`).

General handling rules used by the codebase:

- Templates are evaluated in priority order using tag matching. A more specific tag set (e.g. `pr, closed, merged`) will be preferred over a generic `pr, default` payload when tags match.
- The handler exposes convenience fields (e.g., `issue_type`, `issue_type_name`, `sender_link_md`) and detection helpers so templates can both select by tags and render using the same keys.
- If no template payload matches a refined tag set, the system falls back to a payload tagged with `[default]` for that event family.

- Special note for `workflow_run`: the handler inspects `workflow_run.status` and `workflow_run.conclusion` and will append tags like `completed` plus an outcome tag such as `success` or `failure` (when available). This enables templates to provide outcome-specific payloads (for example `[workflow_run, completed, success]` vs `[workflow_run, completed, failure]`).

See the family sections below for which tags are used in the shipped templates and a short note about how they are produced.

## Global / common fields

These keys are useful across many events when the corresponding objects are present in the payload.

- `repo_full_name` (string) — repository.full_name
- `repo_name` (string) — repository.name
- `repo_url` (string) — repository.html_url
- `repository` (object) — the raw `repository` object from the payload
- `repository_link_md` (string) — markdown link to the repository (e.g. "[owner/repo](https://github.com/owner/repo)")
- `sender` (object) — the raw `sender` object from the payload
- `sender_avatar` (string) — sender.avatar_url
- `sender_link_md` (string) — markdown link to the sender profile (e.g. "[user](https://github.com/user)")
- `sender_name` (string) — sender.login

---

## Code & repository events family

### Common

- `repo_full_name` (string)
- `repo_name` (string)
- `repo_url` (string)
- `repository` (object)

- Tags: push → `[push, default]`, `[push, force]`; create/delete/fork/gollum/repository → `[default]`.
- Condition: `payload.forced == true` → use `force`; otherwise use family tag.

### push (event-specific)

- `branch_link_md` (string)
- `branch_name` (string)
- `branch_url` (string)
- `commit_authors` ([]string)
- `commit_authors_with_links` ([]string)
- `commit_authors_with_links_joined` (string)
- `commit_authors_joined` (string)
- `commit_message` (string)
- `commit_messages` ([]string)
- `commit_messages_joined` (string)
- `commits` (array)
- `commits_count` (int)
- `compare_url` (string)
- `forced` (bool)
- `pusher` (object)
- `pusher_link_md` (string)
- `ref` (string)

### create

- `master_branch` (string)
- `ref` (string)
- `ref_type` (string)

### delete

- `ref` (string)
- `ref_type` (string)

### fork

- `forkee` (object)
- `forkee_full_name` (string)
- `forkee_url` (string)

### gollum (wiki)

- `pages` (array)

---

## Pull request family

### Common

- `pr_number` (int)
- `pr_title` (string)
- `pr_url` (string)
- `pull_request` (object)

- Tags: `[pr, closed, merged]`, `[pr, closed, unmerged]`, `[pr, default]`.
- Condition: `action == "closed" && pull_request.merged == true` → `closed,merged`; `action == "closed" && pull_request.merged == false` → `closed,unmerged`; else `pr,default`.

### pull_request

- `action` (string)
- `pr_base_branch_link_md` (string)
- `pr_base_ref` (string)
- `pr_body` (string)
- `pr_head_branch_link_md` (string)
- `pr_head_ref` (string)
- `pr_merged` (bool)
- `pr_state` (string)
- `pr_user_link_md` (string)

### pull_request_review

- `action` (string)
- `review` (object)
- `review_body` (string)
- `review_state` (string)
- `review_url` (string)
- `review_user_link_md` (string)

### pull_request_review_comment

- `comment_body` (string)
- `comment_url` (string)
- `comment_user_link_md` (string)

---

## Issue family

### Common

- `issue` (object)
- `issue_number` (int)
- `issue_title` (string)
- `issue_url` (string)

- Tags: `[issue, typed]`, `[issue, type:bug]`, `[issue, type:feature]`, `[issue, type:task]`, `[issue, type:unknown]`.
- Condition: `issue.type` if present; otherwise inferred from labels via `detectIssueTypeFromLabels`.

### issues

- `action` (string)
- `issue_body` (string)
- `issue_link_md` (string)
- `issue_state` (string)
- `issue_type` (string)
- `issue_type_name` (string)
- `issue_user_link_md` (string)

### issue_comment

- `comment` (object)
- `comment_body` (string)
- `comment_url` (string)
- `comment_user_link_md` (string)

---

## Discussion family

### Common

- `discussion` (object)
- `discussion_title` (string)
- `discussion_url` (string)

- Tags: `[default]`.
- Condition: use family tag `discussion` (no extra qualifiers).

### discussion

- `action` (string)
- `discussion_body` (string)
- `discussion_user_link_md` (string)

### discussion_comment

- `comment` (object)
- `comment_body` (string)
- `comment_url` (string)
- `comment_user_link_md` (string)

---

## Release & packages

### Common

- `action` (string)

- Tags: `release` → `[default]`; `package` → `[default]`.
- Condition: use family tag (`release` / `package`).

### release

- `release` (object)
- `release_body` (string)
- `release_name` (string)
- `release_tag` (string)
- `release_url` (string)

### package

- `package` (object)
- `package_link_md` (string)
- `package_name` (string)
- `package_publisher_link_md` (string)
- `package_tag_name` (string)
- `package_type` (string)
- `package_version` (string)
- `package_version_name` (string)

---

## CI / deployment / status

### Common

- `action` (string)
- `deployment` (object)
- `deployment_status` (object)
- `status` (object)

- Tags: `[default]` for deployment, deployment_status, check_run, check_suite, workflow_run, status.
- Condition: use family tag (shipped templates don't add qualifiers).

### deployment

- `deployment_id` (any)
- `deployment_url` (string)

### deployment_status

- (see deployment_status object)

### check_run

- (see check_run object)

### check_suite

- `action` (string)

### workflow_run

- `action` (string)
- `workflow_name` (string)
- `workflow_run` (object)
- `workflow_run_number` (int) — the run number (normalized to an integer when possible)
- `workflow_head_branch` (string) — the head branch for the run
- `workflow_head_sha` (string) — the commit SHA the run ran against
- `workflow_run_url` (string) — the run html_url
- `workflow_run_link_md` (string) — markdown link to the run (e.g. "[#123](https://github.com/owner/repo/actions/runs/123)")
- `workflow_repo_full_name` (string) — repository.full_name for the workflow's repository
- `workflow_repo_url` (string) — repository.html_url for the workflow's repository
- `workflow_repository_link_md` (string) — markdown link to the repository (e.g. "[owner/repo](https://github.com/owner/repo)")

- Tags: `[workflow_run, completed, success]`, `[workflow_run, completed, failure]`, `[default]`.
- Condition: the handler appends `completed` when `workflow_run.status == "completed"` and appends `success` or `failure` when `workflow_run.conclusion` matches those values; non-completed statuses (e.g. `in_progress`) are emitted as tags as well so templates may opt to match them.

### status

- `status_state` (string)

---

## Project & board

### Common

- `action` (string)
- `project` (object)
- `project_card` (object)
- `project_column` (object)

- Tags: `[default]` for project, project_card, project_column.
- Condition: use family tag.

### project

- `project_name` (string)
- `project_url` (string)

### project_card

- (see project_card object)

### project_column

- `project_column_name` (string)

### milestone

- `action` (string)
- `milestone` (object)
- `milestone_description` (string)
- `milestone_title` (string)

---

## Access, membership & teams

### Common

- `action` (string)
- `member` (object)
- `membership` (object)
- `team` (object)

- Tags: `[default]` for membership, member, team, organization.
- Condition: use family tag; templates may vary on `action`.

### member

- `member_login` (string)

### organization

- `organization` (object)
- `organization_login` (string)

---

## Pages

### Common

- `page_build` (object)

- Tags: `[default]`.
- Condition: use family tag `page_build`.

---

## Community & visibility

### Common

- `action` (string)
- `repository` (object)

- Tags: `[default]` for public, star, watch.
- Condition: use family tag; `action` often contains the qualifier.

### public

- (see repository object)

### star

- (use `action`)

### watch

- (see repository object)

---

## Security

### Common

- `action` (string)
- `security_advisory` (object)

- Tags: `[default]`.
- Condition: use family tag `security_advisory`.

### security_advisory

- `security_advisory_id` (string)

---

## Notes

- The lists above reflect the keys the handler currently sets in code. Some keys are convenience fields (markdown links, joined strings) derived from the raw payload.
- Types listed are best-effort based on the typical payload shape. Templates should guard against missing keys.
- If you change code to add/remove keys, update this document and add unit tests that assert the presence and shape of the new keys.

---

## Additional Event Families

### Branch protection

#### Common

- `action` (string)
- `repository` (object)

- Tags: `[default]` for branch_protection_configuration, branch_protection_rule.
- Condition: use family tag.

#### branch_protection_configuration

- `branch_protection_configuration` (object)

#### branch_protection_rule

- `rule` (object)
- `rule_name` (string)
- `branch_protection_rule` (object)

---

### Custom properties

#### custom_property

- `action` (string)
- `definition` (object)
- `property_name` (string)
- `custom_property` (object)

#### custom_property_values

- `action` (string)
- `new_property_values` (array)
- `old_property_values` (array)
- `custom_property_values` (object)

---

### Deployment protection & review

#### deployment_protection_rule

- `action` (string)
- `environment` (string)
- `deployment` (object)
- `deployment_callback_url` (string)
- `deployment_protection_rule` (object)

#### deployment_review

- `action` (string)
- `approver` (object)
- `approver_login` (string)
- `approver_link_md` (string)
- `comment` (string)
- `workflow_run` (object)
- `deployment_review` (object)

---

### GitHub App lifecycle

#### github_app_authorization

- `action` (string)
- `github_app_authorization` (object)

#### installation

- `action` (string)
- `installation` (object)
- `installation_id` (int)
- `repositories` (array)
- `repositories_count` (int)
- `installation_event` (object)

#### installation_repositories

- `action` (string)
- `repositories_added` (array)
- `repositories_added_count` (int)
- `repositories_removed` (array)
- `repositories_removed_count` (int)
- `repository_selection` (string)
- `installation_repositories` (object)

#### installation_target

- `action` (string)
- `account` (object)
- `account_login` (string)
- `changes` (object)
- `old_login` (string)
- `target_type` (string)
- `installation_target` (object)

---

### Issue dependencies

#### issue_dependencies

- `action` (string)
- `blocked_issue` (object)
- `blocked_issue_number` (int)
- `blocked_issue_title` (string)
- `blocking_issue` (object)
- `blocking_issue_number` (int)
- `blocking_issue_title` (string)
- `blocking_issue_repo` (object)
- `blocking_issue_repo_name` (string)
- `issue_dependencies` (object)

---

### Marketplace

#### marketplace_purchase

- `action` (string)
- `marketplace_purchase` (object)
- `account_login` (string)
- `plan_name` (string)
- `previous_marketplace_purchase` (object)
- `effective_date` (string)

---

### Merge queue

#### merge_group

- `action` (string)
- `merge_group` (object)
- `head_sha` (string)
- `head_ref` (string)
- `base_sha` (string)
- `base_ref` (string)

---

### Webhooks & Meta

#### meta

- `action` (string)
- `hook` (object)
- `hook_id` (int)
- `hook_type` (string)
- `meta` (object)

#### ping

- `zen` (string)
- `hook` (object)
- `hook_id` (int)
- `hook_type` (string)
- `ping` (object)

---

### Organization blocking

#### org_block

- `action` (string)
- `blocked_user` (object)
- `blocked_user_login` (string)
- `blocked_user_link_md` (string)
- `org_block` (object)

---

### Packages (legacy)

#### registry_package

- `action` (string)
- `registry_package` (object)
- `package_name` (string)
- `package_type` (string)
- `package_version` (string)

---

### Repository security & advisories

#### repository_advisory

- `action` (string)
- `repository_advisory` (object)
- `advisory_id` (string)
- `advisory_summary` (string)
- `advisory_severity` (string)

#### repository_dispatch

- `event_type` (string)
- `client_payload` (object)
- `repository_dispatch` (object)

#### secret_scanning_alert_location

- `action` (string)
- `alert` (object)
- `alert_number` (int)
- `location` (object)
- `location_type` (string)
- `secret_scanning_alert_location` (object)

#### secret_scanning_scan

- `action` (string)
- `scan` (object)
- `scan_status` (string)
- `scan_completed_at` (string)
- `secret_scanning_scan` (object)

#### security_and_analysis

- `changes` (object)
- `security_and_analysis` (object)

---

### Sponsorship

#### sponsorship

- `action` (string)
- `sponsorship` (object)
- `sponsor_login` (string)
- `sponsor_link_md` (string)
- `sponsorable_login` (string)
- `tier_name` (string)
- `tier_monthly_price_cents` (int)
- `tier_monthly_price_dollars` (float)

---

### Sub-issues

#### sub_issues

- `action` (string)
- `parent_issue` (object)
- `parent_issue_number` (int)
- `parent_issue_title` (string)
- `sub_issue` (object)
- `sub_issue_number` (int)
- `sub_issue_title` (string)
- `sub_issues` (object)

---

### Team management

#### team_add

- `team` (object)
- `team_name` (string)
- `team_slug` (string)
- `team_add` (object)

---

### Projects V2

#### projects_v2

- `action` (string)
- `projects_v2` (object)
- `project_id` (int)
- `project_title` (string)
- `project_description` (string)

#### projects_v2_item

- `action` (string)
- `projects_v2_item` (object)
- `item_id` (int)
- `item_node_id` (string)
- `project_node_id` (string)
- `content_node_id` (string)
- `content_type` (string)
- `changes` (object)

#### projects_v2_status_update

- `action` (string)
- `projects_v2_status_update` (object)
- `status_update_id` (int)
- `status_update_body` (string)
- `status` (string)

---

### Pull request review threads

#### pull_request_review_thread

- `action` (string)
- `pull_request` (object)
- `pr_number` (int)
- `pr_title` (string)
- `pr_url` (string)
- `thread` (object)
- `thread_id` (string)
- `thread_comments` (array)
- `thread_comments_count` (int)
- `pull_request_review_thread` (object)

---

### Workflows

#### workflow_dispatch

- `workflow` (string)
- `inputs` (object)
- `ref` (string)
- `workflow_dispatch` (object)

#### workflow_job

- `action` (string)
- `workflow_job` (object)
- `job_id` (int)
- `job_name` (string)
- `job_status` (string)
- `job_conclusion` (string)
- `job_url` (string)
- `run_id` (int)

---

### Personal access tokens

#### personal_access_token_request

- `action` (string)
- `personal_access_token_request` (object)
- `request_id` (int)
- `token_owner_login` (string)
- `token_name` (string)
- `token_expired` (bool)

---

## How to extend

- Add a small, focused `prepare<Event>Data` function that populates only the keys templates need.
- Document new keys in this README (alphabetically) under the appropriate event group.
- Add tests in `internal/handler/handler_test.go` verifying important keys for that event.
