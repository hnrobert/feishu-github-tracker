package handler

import "fmt"

// prepareWorkflowRunData exposes workflow_run object and provides small
// convenience keys used by workflow templates: besides `workflow_run` and
// `workflow.name`, populate repository-related and common workflow-run fields
// so templates can show repo name/URL, run number, branch, head SHA and a
// direct run link when available.
func prepareWorkflowRunData(data map[string]any, payload map[string]any) {
	if wr, ok := payload["workflow_run"].(map[string]any); ok {
		// normalize id: if it's a float64 but has no fractional part, convert to int64
		if idv, okid := wr["id"].(float64); okid {
			if float64(int64(idv)) == idv {
				wr["id"] = int64(idv)
			}
		}

		// expose raw object
		data["workflow_run"] = wr

		// workflow name compatibility
		if name, ok := wr["name"].(string); ok {
			data["workflow_name"] = name
			data["workflow"] = map[string]any{"name": name}
		}

		// run number
		if rn, ok := wr["run_number"].(float64); ok {
			if float64(int64(rn)) == rn {
				data["workflow_run_number"] = int64(rn)
			} else {
				data["workflow_run_number"] = rn
			}
		} else if rn, ok := wr["run_number"].(int); ok {
			data["workflow_run_number"] = rn
		}

		// head branch / sha
		if hb, ok := wr["head_branch"].(string); ok {
			data["workflow_head_branch"] = hb
		}
		if hs, ok := wr["head_sha"].(string); ok {
			data["workflow_head_sha"] = hs
		}

		// html_url for the run (if provided)
		if hur, ok := wr["html_url"].(string); ok && hur != "" {
			data["workflow_run_url"] = hur
			// also provide a markdown link
			if name, ok := wr["name"].(string); ok {
				data["workflow_run_link_md"] = fmt.Sprintf("[%s](%s)", name, hur)
			}
		}
	}

	// repository convenience keys (may already be set by prepareCommonData,
	// but provide workflow-scoped aliases for templates that prefer them)
	if repo, ok := payload["repository"].(map[string]any); ok {
		if full, okf := repo["full_name"].(string); okf {
			data["workflow_repo_full_name"] = full
		}
		if url, oku := repo["html_url"].(string); oku {
			data["workflow_repo_url"] = url
		}
		if full, okf := repo["full_name"].(string); okf {
			if url, oku := repo["html_url"].(string); oku {
				data["workflow_repository_link_md"] = fmt.Sprintf("[%s](%s)", full, url)
			}
		}
	}

	if action, ok := payload["action"].(string); ok {
		data["action"] = action
	}
}
