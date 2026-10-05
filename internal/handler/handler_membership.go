package handler

// prepareMembershipData exposes membership event fields. GitHub's membership
// payload carries the affected user inside the top-level "member" object
// (member.user does NOT exist — the shipped templates were fixed accordingly).
func prepareMembershipData(data map[string]any, payload map[string]any) {
	if m, ok := payload["membership"].(map[string]any); ok {
		data["membership"] = m
	}
	if member, ok := payload["member"].(map[string]any); ok {
		data["member"] = member
		if login, ok := member["login"].(string); ok {
			data["member_login"] = login
		}
	}
}
