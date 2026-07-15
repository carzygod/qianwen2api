package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	qianwenUserBlockPattern = regexp.MustCompile(`(?s)window\._USER_\s*=\s*\{(.*?)\}`)
	qianwenUserIDPattern    = regexp.MustCompile(`(?:userId|aliyunUid)\s*:\s*["']([^"']+)["']`)
)

func (c *qwenWebClient) probeLoginIdentity(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qwenWebBaseURL+"/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Cookie", c.cookieHeader)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("qianwen identity request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("qianwen identity status %d: login required", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("qianwen identity status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read qianwen identity response: %w", err)
	}
	userID, found := parseQianwenIdentity(string(body))
	if !found {
		return "", fmt.Errorf("qianwen identity response format changed: window._USER_ was not found")
	}
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("qianwen identity indicates account is logged out")
	}
	return userID, nil
}

func parseQianwenIdentity(page string) (string, bool) {
	block := qianwenUserBlockPattern.FindStringSubmatch(page)
	if len(block) < 2 {
		return "", false
	}
	for _, match := range qianwenUserIDPattern.FindAllStringSubmatch(block[1], -1) {
		if len(match) >= 2 && strings.TrimSpace(match[1]) != "" {
			return strings.TrimSpace(match[1]), true
		}
	}
	return "", true
}
