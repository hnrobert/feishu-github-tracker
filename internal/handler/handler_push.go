package handler

import (
	"fmt"
	"strings"
)

func preparePushData(data map[string]any, payload map[string]any) {
	data["ref"] = payload["ref"]
	data["compare_url"] = payload["compare"]

	if commits, ok := payload["commits"].([]any); ok {
		data["commits_count"] = len(commits)
		data["commits"] = commits

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
			joined := ""
			for i, m := range msgs {
				if i == 0 {
					joined = m
				} else {
					joined = joined + "\n- " + m
				}
			}
			data["commit_messages_joined"] = joined
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

	if pusher, ok := payload["pusher"].(map[string]any); ok {
		data["pusher"] = pusher
		if pname, ok := pusher["name"].(string); ok {
			data["pusher_link_md"] = fmt.Sprintf("[%s](https://github.com/%s)", pname, pname)
		}
	}
	data["forced"] = payload["forced"]

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
}
