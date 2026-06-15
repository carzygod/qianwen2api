package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type AccountTestResult struct {
	OK           bool   `json:"ok"`
	AccountID    string `json:"account_id"`
	Capability   string `json:"capability"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	ResponseText string `json:"response_text,omitempty"`
}

func TestAccount(accountID, capability string) (*AccountTestResult, error) {
	account, err := AppStore.GetAccount(accountID)
	if err != nil {
		return nil, err
	}
	if capability == "" {
		capability = "chat"
	}

	result := &AccountTestResult{
		AccountID:  accountID,
		Capability: capability,
		Status:     "unknown",
	}

	if account.Type == "guest" {
		if capability != "chat" {
			result.Status = "invalid"
			result.Message = "Guest accounts only support the current chat adapter."
			_ = AppStore.UpdateAccountStatus(accountID, "invalid", result.Message, false)
			return result, nil
		}
		text, err := runGuestChatProbe()
		if err != nil {
			result.Status = "invalid"
			result.Message = err.Error()
			_ = AppStore.UpdateAccountStatus(accountID, "invalid", result.Message, false)
			return result, nil
		}
		result.OK = true
		result.Status = "valid"
		result.Message = "Guest chat probe returned a non-empty assistant response."
		result.ResponseText = text
		_ = AppStore.UpdateAccountStatus(accountID, "valid", "", true)
		return result, nil
	}

	if strings.TrimSpace(account.CookieJSON) == "" && strings.TrimSpace(account.CookieString) == "" {
		result.Status = "invalid"
		result.Message = "Login account has no cookie material. Paste qianwen.com Cookie JSON or request Cookie header first."
		_ = AppStore.UpdateAccountStatus(accountID, "invalid", result.Message, false)
		return result, nil
	}

	result.Status = "unknown"
	result.Message = "Login-cookie probe requires qianwen.com logged-in protocol capture. The account is stored, but it will not be marked valid until a real model call succeeds."
	_ = AppStore.UpdateAccountStatus(accountID, "unknown", result.Message, false)
	return result, nil
}

func runGuestChatProbe() (string, error) {
	if GlobalPool == nil {
		return "", fmt.Errorf("guest pool is disabled")
	}
	account, err := GlobalPool.AcquireAccount()
	if err != nil {
		return "", fmt.Errorf("no guest account available: %w", err)
	}
	defer GlobalPool.ReleaseAccount(account)

	req := &ChatRequest{
		Model: Cfg.DefaultChatModel,
		Messages: []Message{
			{Role: "user", Content: "你好"},
		},
		Stream: false,
	}
	resp, err := sendUpstreamRequest(req, account)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upstream status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var fullContent string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var qwResp QianwenResponse
		if err := json.Unmarshal([]byte(payload), &qwResp); err != nil {
			continue
		}
		for _, content := range qwResp.Contents {
			if content.ContentType != "text" {
				continue
			}
			if content.Incremental {
				fullContent += content.Content
			} else {
				fullContent = content.Content
			}
		}
		if qwResp.MsgStatus == "finished" || qwResp.StopReason == "stop" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(fullContent) == "" {
		return "", fmt.Errorf("guest probe returned empty assistant content")
	}
	return fullContent, nil
}
