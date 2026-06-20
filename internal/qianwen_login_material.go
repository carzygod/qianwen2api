package internal

import (
	"encoding/json"
	"strings"
)

var qianwenStrongLoginCookieMarkers = []string{
	"tongyi_sso_ticket",
	"tongyi_sso_ticket_hash",
	"login_aliyunid_ticket",
	"aliyun_choice",
	"munb",
	"unb",
	"cookie2",
	"_tb_token_",
	"sgcookie",
	"x5sec",
	"havana",
}

func hasLikelyLoginCookie(cookies []capturedCookie) bool {
	return len(qianwenLoginCookieNames(cookies)) > 0
}

func qianwenLoginCookieNames(cookies []capturedCookie) []string {
	names := []string{}
	seen := map[string]bool{}
	for _, cookie := range cookies {
		name := strings.ToLower(strings.TrimSpace(cookie.Name))
		if name == "" {
			continue
		}
		for _, marker := range qianwenStrongLoginCookieMarkers {
			if strings.Contains(name, marker) {
				label := cookie.Name
				if cookie.Domain != "" {
					label = cookie.Domain + "/" + cookie.Name
				}
				if !seen[label] {
					names = append(names, label)
					seen[label] = true
				}
				break
			}
		}
	}
	return names
}

func accountHasQianwenLoginMaterial(account AccountRecord) bool {
	var cookies []qwenCookie
	if strings.TrimSpace(account.CookieJSON) != "" {
		if err := json.Unmarshal([]byte(account.CookieJSON), &cookies); err == nil {
			for _, cookie := range cookies {
				if qianwenCookieNameLooksLoggedIn(cookie.Name) {
					return true
				}
			}
		}
	}
	for _, part := range strings.Split(account.CookieString, ";") {
		name := strings.TrimSpace(part)
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		if qianwenCookieNameLooksLoggedIn(name) {
			return true
		}
	}
	return false
}

func qianwenCookieNameLooksLoggedIn(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	for _, marker := range qianwenStrongLoginCookieMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
