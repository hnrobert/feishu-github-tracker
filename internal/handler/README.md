# Handler data map

This file documents the top-level keys the handler populates in the `data` map used by templates.

Keys are grouped into families. Each family has a `Common` subsection listing keys shared across the family's events. Event sections only list keys that are specific to that event (i.e., not repeated in the family's Common list). Keys are alphabetized inside each list.

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

## Code & repository events

### Common

- `repo_full_name` (string)
- `repo_name` (string)
- `repo_url` (string)
- `repository` (object)

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

### status

- `status_state` (string)

---

## Project & board

### Common

- `action` (string)
- `project` (object)
- `project_card` (object)
- `project_column` (object)

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

### member

- `member_login` (string)

### organization

- `organization` (object)
- `organization_login` (string)

---

## Pages

### Common

- `page_build` (object)

---

## Community & visibility

### Common

- `action` (string)
- `repository` (object)

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

### security_advisory

- `security_advisory_id` (string)

---

## Notes

- The lists above reflect the keys the handler currently sets in code. Some keys are convenience fields (markdown links, joined strings) derived from the raw payload.
- Types listed are best-effort based on the typical payload shape. Templates should guard against missing keys.
- If you change code to add/remove keys, update this document and add unit tests that assert the presence and shape of the new keys.

## How to extend

- Add a small, focused `prepare<Event>Data` function that populates only the keys templates need.
- Document new keys in this README (alphabetically) under the appropriate event group.
- Add tests in `internal/handler/handler_test.go` verifying important keys for that event.
