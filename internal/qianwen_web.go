package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	qwenWebBaseURL = "https://www.qianwen.com"
	qwenChatAPIURL = "https://chat2.qianwen.com"
)

type qwenCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type qwenWebClient struct {
	account      AccountRecord
	httpClient   *http.Client
	cookieHeader string
	xsrfToken    string
	deviceID     string
	userAgent    string
}

type qwenWebEvent struct {
	ErrorMsg  string `json:"error_msg"`
	ErrorCode int    `json:"error_code"`
	Data      struct {
		Messages []struct {
			MimeType string                 `json:"mime_type"`
			Content  string                 `json:"content"`
			Status   string                 `json:"status"`
			MetaData map[string]interface{} `json:"meta_data"`
		} `json:"messages"`
	} `json:"data"`
	Communication struct {
		SessionID string `json:"sessionid"`
		ReqID     string `json:"reqid"`
	} `json:"communication"`
}

type qwenRequestState struct {
	ReqID          string                 `json:"req_id"`
	SessionID      string                 `json:"session_id"`
	DeviceID       string                 `json:"device_id"`
	Payload        map[string]interface{} `json:"payload"`
	InputResources []qwenImageResource    `json:"input_resources,omitempty"`
}

type mediaPollResult struct {
	URLs   []string       `json:"urls"`
	Events []qwenWebEvent `json:"events,omitempty"`
}

var httpURLPattern = regexp.MustCompile(`https?://[^\s"'<>\\)]+`)

func newQwenWebClient(account AccountRecord) (*qwenWebClient, error) {
	cookieHeader, xsrf := accountCookieHeader(account)
	if strings.TrimSpace(cookieHeader) == "" {
		return nil, fmt.Errorf("account has no qianwen cookie material")
	}
	userAgent := strings.TrimSpace(account.UserAgent)
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
	}
	deviceID := strings.TrimSpace(account.DeviceID)
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	return &qwenWebClient{
		account:      account,
		httpClient:   &http.Client{Timeout: 180 * time.Second},
		cookieHeader: cookieHeader,
		xsrfToken:    xsrf,
		deviceID:     deviceID,
		userAgent:    userAgent,
	}, nil
}

func accountCookieHeader(account AccountRecord) (string, string) {
	var cookies []qwenCookie
	if strings.TrimSpace(account.CookieJSON) != "" {
		_ = json.Unmarshal([]byte(account.CookieJSON), &cookies)
	}
	if len(cookies) == 0 {
		return strings.TrimSpace(account.CookieString), cookieValueFromHeader(account.CookieString, "XSRF-TOKEN")
	}
	pairs := make([]string, 0, len(cookies))
	xsrf := ""
	for _, cookie := range cookies {
		if cookie.Name == "" || cookie.Value == "" {
			continue
		}
		pairs = append(pairs, cookie.Name+"="+cookie.Value)
		if cookie.Name == "XSRF-TOKEN" {
			xsrf = cookie.Value
		}
	}
	return strings.Join(pairs, "; "), xsrf
}

func cookieValueFromHeader(header, name string) string {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, name+"=") {
			return strings.TrimPrefix(part, name+"=")
		}
	}
	return ""
}

func (c *qwenWebClient) chat(ctx context.Context, req *ChatRequest) (string, []qwenWebEvent, error) {
	state := c.buildBasePayload(chatPrompt(req.Messages))
	events, err := c.postPayload(ctx, "/api/v2/chat", state)
	if err != nil {
		return "", nil, err
	}
	text := extractQwenText(events)
	if strings.TrimSpace(text) == "" {
		return "", events, fmt.Errorf("qianwen chat returned empty assistant content")
	}
	return text, events, nil
}

func (c *qwenWebClient) submitImage(ctx context.Context, req ImageGenerationRequest) (*qwenRequestState, []qwenWebEvent, error) {
	state := c.buildBasePayload(req.Prompt)
	state.Payload["ai_tool_scene"] = "zaodian_generate_image"
	state.Payload["biz_data"] = mustJSONString(map[string]interface{}{
		"req": map[string]interface{}{
			"rootModel":      "qwen2",
			"prompt":         req.Prompt,
			"originPrompt":   req.Prompt,
			"negativePrompt": req.NegativePrompt,
			"params": map[string]interface{}{
				"size":     normalizeImageAspect(req),
				"features": []string{"qwen_image_2"},
			},
		},
		"bizScene": "genImage",
	})
	events, err := c.postPayload(ctx, "/api/v2/chat", state)
	return state, events, err
}

func (c *qwenWebClient) submitVideo(ctx context.Context, req VideoGenerationRequest) (*qwenRequestState, []qwenWebEvent, error) {
	state := c.buildBasePayload(req.Prompt)
	duration := req.Duration
	if duration <= 0 {
		duration = 10
	}
	resolution := strings.TrimSpace(req.Resolution)
	if resolution == "" {
		resolution = "720P"
	}
	resolution = strings.ToUpper(resolution)
	aspect := strings.TrimSpace(req.AspectRatio)
	if aspect == "" {
		aspect = "16:9"
	}
	params := map[string]interface{}{
		"duration":   duration,
		"resolution": resolution,
		"size":       aspect,
	}
	resources, err := resolveQwenVideoInputResources(req)
	if err != nil {
		return state, nil, err
	}
	if len(resources) > 0 {
		state.InputResources = resources
		params["attachments"] = qwenVideoAttachments(resources)
		addQwenImageMessagesToPayload(state.Payload, resources)
	}
	state.Payload["ai_tool_scene"] = "zaodian_generate_video"
	state.Payload["biz_data"] = mustJSONString(map[string]interface{}{
		"req": map[string]interface{}{
			"rootModel":    QianwenVideoProviderModel,
			"prompt":       req.Prompt,
			"originPrompt": req.Prompt,
			"genMode":      "vid_gen",
			"params":       params,
		},
		"bizScene": "genVideo",
		"videoReportParams": map[string]interface{}{
			"scene_agent":      "ai_video",
			"quota_use":        "2",
			"video_duration":   fmt.Sprintf("%ds", duration),
			"model":            QianwenVideoModelID,
			"video_ratio":      aspect,
			"video_resolution": resolution,
		},
	})
	events, err := c.postPayload(ctx, "/api/v2/chat", state)
	return state, events, err
}

func (c *qwenWebClient) pollSnap(ctx context.Context, state *qwenRequestState) ([]qwenWebEvent, error) {
	return c.postPayload(ctx, "/api/v1/chat/snap", state)
}

func (c *qwenWebClient) pollMedia(ctx context.Context, state *qwenRequestState, mediaType string, timeout time.Duration) (*mediaPollResult, error) {
	deadline := time.Now().Add(timeout)
	var lastEvents []qwenWebEvent
	for {
		events, err := c.pollSnap(ctx, state)
		if err != nil {
			return nil, err
		}
		lastEvents = events
		urls := filterMediaURLs(extractURLs(events), mediaType)
		if len(urls) > 0 {
			return &mediaPollResult{URLs: urls, Events: events}, nil
		}
		if time.Now().After(deadline) {
			return &mediaPollResult{Events: lastEvents}, fmt.Errorf("qianwen %s generation did not return media url before timeout: %s", mediaType, summarizeMediaEvents(lastEvents, mediaType))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *qwenWebClient) buildBasePayload(prompt string) *qwenRequestState {
	reqID := generateRandomHex(16)
	sessionID := strings.ReplaceAll(uuid.New().String(), "-", "")
	payload := map[string]interface{}{
		"req_id":        reqID,
		"parent_req_id": "0",
		"messages": []map[string]interface{}{
			{
				"mime_type": "text/plain",
				"content":   prompt,
				"meta_data": map[string]interface{}{"ori_query": prompt},
				"status":    "complete",
			},
		},
		"scene":            "chat",
		"sub_scene":        "",
		"scene_param":      "first_turn",
		"session_id":       sessionID,
		"biz_id":           "ai_qwen",
		"topic_id":         generateRandomToken(32, ""),
		"model":            "Qwen",
		"from":             "default",
		"protocol_version": "v2",
		"messages_merge":   false,
		"chat_client":      "h5",
		"deep_search":      "0",
		"temporary":        false,
	}
	return &qwenRequestState{ReqID: reqID, SessionID: sessionID, DeviceID: c.deviceID, Payload: payload}
}

func (c *qwenWebClient) postPayload(ctx context.Context, path string, state *qwenRequestState) ([]qwenWebEvent, error) {
	body, _ := json.Marshal(state.Payload)
	signedURL, acsDeviceID, acsHeaders, err := loginACSAuth.signedURLAndHeaders(c.account, path, body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, signedURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, state, acsDeviceID, acsHeaders)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qianwen upstream status %d: %s", resp.StatusCode, string(raw))
	}
	events, err := parseQwenSSE(resp.Body)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (c *qwenWebClient) setHeaders(req *http.Request, state *qwenRequestState, acsDeviceID string, acsHeaders map[string]string) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream, text/plain, */*")
	req.Header.Set("Origin", qwenWebBaseURL)
	req.Header.Set("Referer", qwenWebBaseURL+"/chat")
	req.Header.Set("Cookie", c.cookieHeader)
	req.Header.Set("x-platform", "pc_tongyi")
	if acsDeviceID != "" {
		req.Header.Set("x-device-id", acsDeviceID)
	} else {
		req.Header.Set("x-device-id", state.DeviceID)
	}
	req.Header.Set("x-chat-id", state.ReqID)
	req.Header.Set("x-wpk-reqid", state.ReqID)
	req.Header.Set("x-chat-biz", mustJSONString(map[string]interface{}{"chatId": state.ReqID, "agentId": "", "enableWebp": ""}))
	req.Header.Set("x-wpk-bid", "66ur41cs-cntu1744")
	req.Header.Set("x-wpk-rel", "")
	req.Header.Set("x-wpk-traceid", state.ReqID)
	if c.xsrfToken != "" {
		req.Header.Set("x-csrf-token", c.xsrfToken)
	}
	for key, value := range acsHeaders {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
}

func summarizeMediaEvents(events []qwenWebEvent, mediaType string) string {
	urls := extractURLs(events)
	filtered := filterMediaURLs(urls, mediaType)
	text := extractQwenText(events)
	if len(text) > 240 {
		text = text[:240]
	}
	summary := map[string]interface{}{
		"events":        len(events),
		"urls":          urls,
		"filtered_urls": filtered,
		"text":          strings.TrimSpace(text),
	}
	return marshalCompact(summary)
}

func parseQwenSSE(body io.Reader) ([]qwenWebEvent, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	events := []qwenWebEvent{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event qwenWebEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			LogWarn("Failed to parse qianwen SSE event: %v", err)
			continue
		}
		if event.ErrorCode != 0 {
			return events, fmt.Errorf("qianwen error %d: %s", event.ErrorCode, event.ErrorMsg)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}

func extractQwenText(events []qwenWebEvent) string {
	text := ""
	for _, event := range events {
		for _, msg := range event.Data.Messages {
			if msg.Content == "" {
				continue
			}
			if strings.Contains(msg.Content, "<div") || strings.Contains(msg.Content, "<style") {
				continue
			}
			if msg.Status == "complete" {
				text = msg.Content
			} else if len(msg.Content) > len(text) {
				text = msg.Content
			}
		}
	}
	return strings.TrimSpace(text)
}

func extractURLs(value interface{}) []string {
	raw, _ := json.Marshal(value)
	matches := httpURLPattern.FindAllString(string(raw), -1)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		match = html.UnescapeString(strings.TrimRight(match, ".,;"))
		match = strings.ReplaceAll(match, `\/`, `/`)
		if seen[match] {
			continue
		}
		seen[match] = true
		out = append(out, match)
	}
	return out
}

func filterMediaURLs(urls []string, mediaType string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, u := range urls {
		lower := strings.ToLower(u)
		if strings.Contains(lower, "g.alicdn.com") || strings.Contains(lower, "w3.org") || strings.Contains(lower, "images.quark.cn/s/uae/g/1y/fea/prod/file/20260109") {
			continue
		}
		if mediaType == "video" {
			if !strings.Contains(lower, ".mp4") {
				continue
			}
		} else {
			if strings.Contains(lower, ".mp4") || !(strings.Contains(lower, ".png") || strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".webp")) {
				continue
			}
		}
		if seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

func chatPrompt(messages []Message) string {
	if len(messages) == 0 {
		return "Hello"
	}
	if len(messages) == 1 {
		return messages[0].Content
	}
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		parts = append(parts, strings.ToUpper(role)+": "+msg.Content)
	}
	return strings.Join(parts, "\n")
}

func normalizeImageAspect(req ImageGenerationRequest) string {
	value := strings.TrimSpace(req.AspectRatio)
	if value == "" {
		value = strings.TrimSpace(req.Size)
	}
	if strings.Contains(value, ":") {
		return value
	}
	switch value {
	case "1024x1024", "1x1":
		return "1:1"
	case "1024x1792", "9x16":
		return "9:16"
	case "1792x1024", "16x9":
		return "16:9"
	default:
		return "3:4"
	}
}

func mustJSONString(value interface{}) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func marshalCompact(value interface{}) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
