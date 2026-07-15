package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
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
		Status:     accountStatusUnknown,
	}
	if err := AppStore.BeginAccountTest(accountID); err != nil {
		return nil, fmt.Errorf("begin account test: %w", err)
	}

	if account.Type == "guest" {
		if capability != "chat" {
			result.Status = "invalid"
			result.Message = "游客池账号目前只支持对话适配器。"
			_ = AppStore.UpdateAccountStatus(accountID, "invalid", result.Message, false)
			return result, nil
		}
		text, err := runGuestChatProbe()
		if err != nil {
			return finishFailedAccountTest(result, err), nil
		}
		result.OK = true
		result.Status = "valid"
		result.Message = "游客池对话测活已拿到非空模型回复。"
		result.ResponseText = text
		_ = AppStore.UpdateAccountStatus(accountID, "valid", "", true)
		return result, nil
	}

	if strings.TrimSpace(account.CookieJSON) == "" && strings.TrimSpace(account.CookieString) == "" {
		result.Status = "invalid"
		result.Message = "登录账号缺少 Cookie 材料，请先通过扫码登录捕获 qianwen.com 登录态。"
		_ = AppStore.UpdateAccountStatus(accountID, "invalid", result.Message, false)
		return result, nil
	}

	if !accountHasQianwenLoginMaterial(*account) {
		result.Status = accountStatusInvalid
		result.Message = "账号当前只有 qianwen.com 游客态 Cookie，缺少真实登录票据；游客回复不能作为登录账号测活结果。请重新扫码登录。"
		_ = AppStore.UpdateAccountStatus(accountID, accountStatusInvalid, result.Message, false)
		return result, nil
	}

	client, err := newQwenWebClient(*account)
	if err != nil {
		return finishFailedAccountTest(result, err), nil
	}
	identityCtx, identityCancel := context.WithTimeout(context.Background(), 20*time.Second)
	_, err = client.probeLoginIdentity(identityCtx)
	identityCancel()
	if err != nil {
		return finishFailedAccountTest(result, err), nil
	}

	if capability == "image" || capability == "video" {
		ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
		defer cancel()
		if capability == "image" {
			media, err := generateImageWithAccount(ctx, *account, ImageGenerationRequest{
				Model:  Cfg.DefaultImageModel,
				Prompt: "生成一张简单的蓝色圆形图标，用于账号可用性测试。",
				N:      1,
				Size:   "1024x1024",
			})
			if err != nil {
				return finishFailedAccountTest(result, err), nil
			}
			result.OK = len(media.URLs) > 0
			result.Status = "valid"
			result.Message = "image 测试已拿到真实图片 URL。"
			result.ResponseText = strings.Join(media.URLs, "\n")
			_ = AppStore.UpdateAccountStatus(accountID, "valid", "", true)
			return result, nil
		}
		media, err := generateVideoWithAccount(ctx, *account, VideoGenerationRequest{
			Model:       Cfg.DefaultVideoModel,
			Prompt:      "生成一个白色立方体在桌面缓慢旋转的 1 秒短视频。",
			Duration:    1,
			AspectRatio: "16:9",
			Resolution:  "720P",
		})
		if err != nil {
			return finishFailedAccountTest(result, err), nil
		}
		result.OK = len(media.URLs) > 0
		result.Status = "valid"
		result.Message = "video 测试已拿到真实视频 URL。"
		result.ResponseText = strings.Join(media.URLs, "\n")
		_ = AppStore.UpdateAccountStatus(accountID, "valid", "", true)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	text, _, err := client.chat(ctx, &ChatRequest{
		Model: Cfg.DefaultChatModel,
		Messages: []Message{
			{Role: "user", Content: "请只回复：ok"},
		},
	})
	if err != nil {
		return finishFailedAccountTest(result, err), nil
	}
	result.OK = true
	result.Status = "valid"
	result.Message = "登录账号已从 qianwen.com 拿到非空模型回复。"
	result.ResponseText = text
	_ = AppStore.UpdateAccountStatus(accountID, "valid", "", true)
	return result, nil
}

func finishFailedAccountTest(result *AccountTestResult, err error) *AccountTestResult {
	result.Status = accountStatusUnknown
	result.Message = err.Error()
	if looksQianwenAccountStateError(result.Message) {
		result.Status = accountStatusInvalid
	}
	_ = AppStore.UpdateAccountStatus(result.AccountID, result.Status, result.Message, false)
	return result
}

func runGuestChatProbe() (string, error) {
	if GlobalPool == nil {
		return "", fmt.Errorf("游客池未启用")
	}
	account, err := GlobalPool.AcquireAccount()
	if err != nil {
		return "", fmt.Errorf("没有可用的游客池账号：%w", err)
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
		return "", fmt.Errorf("游客池测活返回了空回复")
	}
	return fullContent, nil
}
