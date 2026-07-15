package internal

import "strings"

func recordQianwenProviderFailure(accountID string, err error) {
	if AppStore == nil || err == nil || strings.TrimSpace(accountID) == "" {
		return
	}
	message := err.Error()
	if looksQianwenAccountStateError(message) {
		_ = AppStore.UpdateAccountRuntimeFailure(accountID, accountStatusInvalid, message)
		return
	}
	_ = AppStore.UpdateAccountRuntimeFailure(accountID, accountStatusUnknown, message)
}

func looksQianwenAccountStateError(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	patterns := []string{
		"account has no qianwen cookie material",
		"login_account_invalid",
		"unauthorized",
		"unauthenticated",
		"not authenticated",
		"not logged",
		"logged out",
		"login required",
		"cookie",
		"xsrf",
		"csrf",
		"qianwen upstream status 401",
		"\u672a\u767b\u5f55",
		"\u767b\u5f55",
		"\u8bf7\u5148\u767b\u5f55",
		"\u8eab\u4efd\u9a8c\u8bc1",
		"\u8ba4\u8bc1",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
