package internal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

const qwenAIBaseURL = "https://chat.qwen.ai"

var (
	qwenAITaskIDPattern = regexp.MustCompile(`(?i)"(?:task_id|taskId|wanx_task_id|wanxTaskId)"\s*:\s*"([^"]+)"`)
	qwenAIJWTLike       = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
)

type qwenAIClient struct {
	account    AccountRecord
	token      string
	httpClient *http.Client
	userAgent  string
}

type qwenAIMediaResult struct {
	URLs              []string                 `json:"urls"`
	ChatID            string                   `json:"chat_id,omitempty"`
	TaskID            string                   `json:"task_id,omitempty"`
	RequestJSON       string                   `json:"request_json,omitempty"`
	ResponseJSON      string                   `json:"response_json,omitempty"`
	UpstreamFiles     []map[string]interface{} `json:"upstream_files,omitempty"`
	Resources         []qwenImageResource      `json:"resources,omitempty"`
	InputResources    []qwenImageResource      `json:"input_resources,omitempty"`
	InputRoute        string                   `json:"input_route,omitempty"`
	PrimaryRouteError string                   `json:"primary_route_error,omitempty"`
}

type qwenAITokenCandidate struct {
	Value string
	Score int
	Key   string
}

type qwenAIInputFile struct {
	Filename    string
	ContentType string
	Bytes       []byte
	RemoteRef   map[string]interface{}
}

func newQwenAIClient(account AccountRecord) (*qwenAIClient, error) {
	token := qwenAIAccessToken(account)
	if token == "" {
		return nil, fmt.Errorf("qianwen media account is missing chat.qwen.ai Bearer token; please re-login from Admin so QIANWEN-WEB-01 can capture qianwen.com and chat.qwen.ai storage")
	}
	userAgent := strings.TrimSpace(account.UserAgent)
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
	}
	return &qwenAIClient{
		account:    account,
		token:      token,
		httpClient: &http.Client{Timeout: 180 * time.Second},
		userAgent:  userAgent,
	}, nil
}

func accountHasQwenAIToken(account AccountRecord) bool {
	return qwenAIAccessToken(account) != ""
}

func qwenAIAccessToken(account AccountRecord) string {
	candidates := []qwenAITokenCandidate{}
	addTokenCandidatesFromJSON(account.LocalStorageJSON, "storage", &candidates)
	addTokenCandidatesFromCookieHeader(account.CookieString, &candidates)

	var cookieItems []map[string]interface{}
	if strings.TrimSpace(account.CookieJSON) != "" && json.Unmarshal([]byte(account.CookieJSON), &cookieItems) == nil {
		for _, item := range cookieItems {
			name, _ := item["name"].(string)
			value, _ := item["value"].(string)
			addQwenAITokenCandidate(name, value, &candidates)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	return candidates[0].Value
}

func addTokenCandidatesFromCookieHeader(header string, candidates *[]qwenAITokenCandidate) {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		addQwenAITokenCandidate(part[:idx], part[idx+1:], candidates)
	}
}

func addTokenCandidatesFromJSON(raw, key string, candidates *[]qwenAITokenCandidate) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		addQwenAITokenCandidate(key, raw, candidates)
		return
	}
	collectQwenAITokens(value, key, candidates, 0)
}

func collectQwenAITokens(value interface{}, key string, candidates *[]qwenAITokenCandidate, depth int) {
	if depth > 8 {
		return
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		for k, v := range typed {
			collectQwenAITokens(v, joinTokenKey(key, k), candidates, depth+1)
		}
	case []interface{}:
		for idx, v := range typed {
			collectQwenAITokens(v, key+"["+strconv.Itoa(idx)+"]", candidates, depth+1)
		}
	case string:
		trimmed := strings.TrimSpace(typed)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			var nested interface{}
			if json.Unmarshal([]byte(trimmed), &nested) == nil {
				collectQwenAITokens(nested, key, candidates, depth+1)
			}
		}
		addQwenAITokenCandidate(key, trimmed, candidates)
	}
}

func joinTokenKey(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func addQwenAITokenCandidate(key, value string, candidates *[]qwenAITokenCandidate) {
	keyLower := strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "Bearer ")
	value = strings.TrimPrefix(value, "bearer ")
	value = strings.Trim(value, `"`)
	if value == "" || len(value) < 24 || len(value) > 8192 {
		return
	}
	if strings.Contains(keyLower, "csrf") || strings.Contains(keyLower, "xsrf") || strings.Contains(keyLower, "refresh") {
		return
	}
	score := 0
	switch {
	case strings.Contains(keyLower, "authorization") || strings.Contains(keyLower, "bearer"):
		score += 120
	case strings.Contains(keyLower, "access_token") || strings.Contains(keyLower, "accesstoken"):
		score += 110
	case strings.Contains(keyLower, "token"):
		score += 80
	}
	if score == 0 {
		return
	}
	if qwenAIJWTLike.MatchString(value) {
		score += 40
	}
	if strings.HasPrefix(value, "eyJ") {
		score += 20
	}
	*candidates = append(*candidates, qwenAITokenCandidate{Value: value, Score: score, Key: key})
}

func (c *qwenAIClient) headers() http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json, text/event-stream")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", c.userAgent)
	headers.Set("Authorization", "Bearer "+c.token)
	headers.Set("x-request-id", uuid.New().String())
	return headers
}

func (c *qwenAIClient) requestJSON(ctx context.Context, method, endpoint string, body interface{}, timeout time.Duration) (int, string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, "", err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, qwenAIBaseURL+endpoint, reader)
	if err != nil {
		return 0, "", err
	}
	req.Header = c.headers()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

func (c *qwenAIClient) createChat(ctx context.Context, model, chatType string) (string, error) {
	chatType = qwenAINormalizeChatType(chatType)
	ts := time.Now().Unix()
	status, text, err := c.requestJSON(ctx, http.MethodPost, "/api/v2/chats/new", map[string]interface{}{
		"title":     fmt.Sprintf("api_%d", ts),
		"models":    []string{model},
		"chat_mode": "normal",
		"chat_type": chatType,
		"timestamp": ts,
	}, 30*time.Second)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("create_chat HTTP %d: %s", status, truncateQwenAIText(text, 500))
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return "", fmt.Errorf("create_chat parse failed: %w", err)
	}
	data, _ := payload["data"].(map[string]interface{})
	id, _ := data["id"].(string)
	if id == "" {
		return "", fmt.Errorf("create_chat missing id: %s", truncateQwenAIText(text, 500))
	}
	return id, nil
}

func (c *qwenAIClient) deleteChat(chatID string) {
	if chatID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _, _ = c.requestJSON(ctx, http.MethodDelete, "/api/v2/chats/"+chatID, nil, 15*time.Second)
}

func (c *qwenAIClient) streamChatRaw(ctx context.Context, chatID string, payload map[string]interface{}, timeout time.Duration) (string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenAIBaseURL+"/api/v2/chat/completions?chat_id="+chatID, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header = c.headers()
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("chat_completion HTTP %d: %s", resp.StatusCode, truncateQwenAIText(string(body), 500))
	}
	return string(body), nil
}

func (c *qwenAIClient) postChatCompletionOnce(ctx context.Context, chatID string, payload map[string]interface{}, timeout time.Duration) (int, string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenAIBaseURL+"/api/v2/chat/completions?chat_id="+chatID, bytes.NewReader(raw))
	if err != nil {
		return 0, "", err
	}
	req.Header = c.headers()
	req.Header.Set("X-Accel-Buffering", "no")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func (c *qwenAIClient) generateImage(ctx context.Context, req ImageGenerationRequest) (*qwenAIMediaResult, error) {
	model := qwenAIUpstreamModel(req.Model)
	files, err := c.prepareMediaFiles(ctx, []string{req.ReferenceImage})
	if err != nil {
		return nil, err
	}
	aspect := normalizeImageAspect(req)
	prompt := req.Prompt
	if len(files) > 0 {
		prompt = "Use the uploaded image(s) as visual reference. Generate image content for this request only.\nUser request: " + req.Prompt
	} else {
		prompt = "Generate image content for this request only.\nUser request: " + req.Prompt
	}
	if req.NegativePrompt != "" {
		prompt += "\nNegative prompt: " + req.NegativePrompt
	}
	prompt += "\nAspect ratio: " + aspect

	chatID, err := c.createChat(ctx, model, "image_gen")
	if err != nil {
		return nil, err
	}
	defer c.deleteChat(chatID)

	payload := qwenAIChatPayload(chatID, model, prompt, files, "image_gen", map[string]interface{}{"ratio": aspect})
	raw, err := c.streamChatRaw(ctx, chatID, payload, 4*time.Minute)
	result := &qwenAIMediaResult{
		ChatID:        chatID,
		RequestJSON:   marshalCompact(payload),
		ResponseJSON:  raw,
		UpstreamFiles: files,
	}
	if err != nil {
		return result, err
	}
	urls := filterMediaURLs(extractURLs(raw), "image")
	if len(urls) == 0 {
		return result, fmt.Errorf("qwen ai image generation produced no image URL: %s", summarizeQwenAIRaw(raw))
	}
	result.URLs = urls
	return result, nil
}

func (c *qwenAIClient) generateVideo(ctx context.Context, req VideoGenerationRequest) (*qwenAIMediaResult, error) {
	model := qwenAIUpstreamModel(req.Model)
	sources := []string{req.FirstFrameImage}
	sources = append(sources, req.ReferenceImages...)
	files, err := c.prepareMediaFiles(ctx, sources)
	if err != nil {
		return nil, err
	}
	aspect := strings.TrimSpace(req.AspectRatio)
	if aspect == "" {
		aspect = "16:9"
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 5
	}
	resolution := strings.TrimSpace(req.Resolution)
	if resolution == "" {
		resolution = "720P"
	}
	prompt := req.Prompt
	if len(files) > 0 {
		prompt = "Use the uploaded image(s) as first-frame or visual reference for the video.\nUser request: " + req.Prompt
	}
	if req.NegativePrompt != "" {
		prompt += "\nNegative prompt: " + req.NegativePrompt
	}
	prompt += fmt.Sprintf("\nVideo parameters: %d seconds, %s, %s.", duration, resolution, aspect)

	chatID, err := c.createChat(ctx, model, "t2v")
	if err != nil {
		return nil, err
	}
	defer c.deleteChat(chatID)

	options := map[string]interface{}{"ratio": aspect, "duration": duration, "resolution": resolution}
	payload := qwenAIChatPayload(chatID, model, prompt, files, "t2v", options)
	payload["stream"] = false
	status, raw, err := c.postChatCompletionOnce(ctx, chatID, payload, 120*time.Second)
	result := &qwenAIMediaResult{
		ChatID:        chatID,
		RequestJSON:   marshalCompact(payload),
		ResponseJSON:  raw,
		UpstreamFiles: files,
	}
	if err != nil {
		return result, err
	}
	if status != http.StatusOK {
		return result, fmt.Errorf("qwen ai video completion HTTP %d: %s", status, truncateQwenAIText(raw, 800))
	}
	urls := filterMediaURLs(extractURLs(raw), "video")
	taskIDs := extractQwenAITaskIDs(raw)
	if len(urls) == 0 && len(taskIDs) > 0 {
		result.TaskID = taskIDs[0]
		polled, err := c.pollVideoTask(ctx, taskIDs[0], 8*time.Minute)
		result.ResponseJSON += "\n" + polled
		if err != nil {
			return result, err
		}
		urls = filterMediaURLs(extractURLs(result.ResponseJSON), "video")
	}
	if len(urls) == 0 {
		detail, detailErr := c.getChatDetail(ctx, chatID, 30*time.Second)
		if detailErr == nil && detail != "" {
			result.ResponseJSON += "\n" + detail
			urls = filterMediaURLs(extractURLs(result.ResponseJSON), "video")
		}
	}
	if len(urls) == 0 {
		return result, fmt.Errorf("qwen ai video generation produced no video URL: %s", summarizeQwenAIRaw(result.ResponseJSON))
	}
	result.URLs = urls
	return result, nil
}

func (c *qwenAIClient) getChatDetail(ctx context.Context, chatID string, timeout time.Duration) (string, error) {
	status, text, err := c.requestJSON(ctx, http.MethodGet, "/api/v2/chats/"+chatID, nil, timeout)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("get chat detail HTTP %d: %s", status, truncateQwenAIText(text, 300))
	}
	return text, nil
}

func (c *qwenAIClient) pollVideoTask(ctx context.Context, taskID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	snapshots := []string{}
	lastStatus := ""
	for time.Now().Before(deadline) {
		status, body, err := c.requestJSON(ctx, http.MethodGet, "/api/v1/tasks/status/"+taskID, nil, 30*time.Second)
		if body != "" {
			snapshots = append(snapshots, body)
		}
		if err == nil && status == http.StatusOK {
			taskStatus := qwenAITaskStatus(body)
			if taskStatus != "" {
				lastStatus = taskStatus
			}
			if qwenAITaskStatusIsSuccess(taskStatus) {
				return strings.Join(snapshots, "\n"), nil
			}
			if taskStatus != "" && !qwenAITaskStatusIsRunning(taskStatus) {
				return strings.Join(snapshots, "\n"), fmt.Errorf("qwen ai video task failed status=%s body=%s", taskStatus, truncateQwenAIText(body, 800))
			}
		}
		select {
		case <-ctx.Done():
			return strings.Join(snapshots, "\n"), ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	return strings.Join(snapshots, "\n"), fmt.Errorf("qwen ai video task timed out task_id=%s last_status=%s", taskID, defaultString(lastStatus, "-"))
}

func (c *qwenAIClient) prepareMediaFiles(ctx context.Context, sources []string) ([]map[string]interface{}, error) {
	files := []map[string]interface{}{}
	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		input, err := qwenAIResolveInputFile(ctx, source)
		if err != nil {
			return nil, err
		}
		if input.RemoteRef != nil {
			files = append(files, input.RemoteRef)
			continue
		}
		remote, err := c.uploadInputFile(ctx, input)
		if err != nil {
			return nil, err
		}
		files = append(files, remote)
	}
	return dedupeQwenAIFiles(files), nil
}

func qwenAIResolveInputFile(ctx context.Context, source string) (qwenAIInputFile, error) {
	if remote, ok := parseQwenAIRemoteRef(source); ok {
		return qwenAIInputFile{RemoteRef: remote}, nil
	}
	if strings.HasPrefix(source, "data:") {
		contentType, raw, err := decodeQwenAIDataURI(source)
		if err != nil {
			return qwenAIInputFile{}, err
		}
		return qwenAIInputFile{Filename: "input" + qwenAIExtForContentType(contentType), ContentType: contentType, Bytes: raw}, nil
	}
	if looksLikeBase64Image(source) {
		raw, err := base64.StdEncoding.DecodeString(source)
		if err == nil {
			contentType := http.DetectContentType(raw)
			if strings.HasPrefix(contentType, "image/") {
				return qwenAIInputFile{Filename: "input" + qwenAIExtForContentType(contentType), ContentType: contentType, Bytes: raw}, nil
			}
		}
	}
	if strings.HasPrefix(strings.ToLower(source), "http://") || strings.HasPrefix(strings.ToLower(source), "https://") {
		return downloadQwenAIInputFile(ctx, source)
	}
	return qwenAIInputFile{}, fmt.Errorf("unsupported image material; use http(s) URL, data URI/base64, or Qwen remote file JSON")
}

func parseQwenAIRemoteRef(source string) (map[string]interface{}, bool) {
	var raw map[string]interface{}
	if !strings.HasPrefix(strings.TrimSpace(source), "{") {
		return nil, false
	}
	if json.Unmarshal([]byte(source), &raw) != nil {
		return nil, false
	}
	if _, ok := raw["file"].(map[string]interface{}); ok {
		return raw, true
	}
	if remote, ok := raw["remote_ref"].(map[string]interface{}); ok {
		return remote, true
	}
	return nil, false
}

func decodeQwenAIDataURI(uri string) (string, []byte, error) {
	head, body, ok := strings.Cut(uri, ",")
	if !ok {
		return "", nil, fmt.Errorf("invalid data URI")
	}
	contentType := "application/octet-stream"
	if strings.HasPrefix(head, "data:") {
		contentType = strings.TrimPrefix(head, "data:")
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = contentType[:idx]
		}
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", nil, err
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(raw)
	}
	return contentType, raw, nil
}

func looksLikeBase64Image(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 128 || strings.Contains(value, "://") {
		return false
	}
	for _, ch := range value {
		if !(ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '+' || ch == '/' || ch == '=' || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func downloadQwenAIInputFile(ctx context.Context, source string) (qwenAIInputFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return qwenAIInputFile{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return qwenAIInputFile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return qwenAIInputFile{}, fmt.Errorf("download image material HTTP %d", resp.StatusCode)
	}
	const maxBytes = int64(32 << 20)
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return qwenAIInputFile{}, err
	}
	if int64(len(raw)) > maxBytes {
		return qwenAIInputFile{}, fmt.Errorf("image material exceeds %d bytes", maxBytes)
	}
	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(raw)
	}
	filename := path.Base(resp.Request.URL.Path)
	if filename == "." || filename == "/" || filename == "" {
		filename = "input" + qwenAIExtForContentType(contentType)
	}
	if filepath.Ext(filename) == "" {
		filename += qwenAIExtForContentType(contentType)
	}
	return qwenAIInputFile{Filename: filename, ContentType: contentType, Bytes: raw}, nil
}

func (c *qwenAIClient) uploadInputFile(ctx context.Context, input qwenAIInputFile) (map[string]interface{}, error) {
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = http.DetectContentType(input.Bytes)
	}
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		filename = "input" + qwenAIExtForContentType(contentType)
	}
	status, text, err := c.requestJSON(ctx, http.MethodPost, "/api/v2/files/getstsToken", map[string]interface{}{
		"filename": filename,
		"filesize": len(input.Bytes),
		"filetype": "file",
	}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("getstsToken failed HTTP %d: %s", status, truncateQwenAIText(text, 500))
	}
	var stsPayload map[string]interface{}
	if err := json.Unmarshal([]byte(text), &stsPayload); err != nil {
		return nil, err
	}
	stsData, _ := stsPayload["data"].(map[string]interface{})
	fileID := qwenAIString(stsData["file_id"])
	filePathRemote := qwenAIString(stsData["file_path"])
	bucketName := qwenAIString(stsData["bucketname"])
	endpoint := strings.TrimPrefix(strings.TrimPrefix(qwenAIString(stsData["endpoint"]), "https://"), "http://")
	region := qwenAINormalizeOSSRegion(qwenAIString(stsData["region"]))
	accessKeyID := qwenAIString(stsData["access_key_id"])
	accessKeySecret := qwenAIString(stsData["access_key_secret"])
	securityToken := qwenAIString(stsData["security_token"])
	if fileID == "" || filePathRemote == "" || bucketName == "" || endpoint == "" || accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("getstsToken missing required fields: %s", truncateQwenAIText(text, 500))
	}

	options := []oss.ClientOption{oss.SecurityToken(securityToken), oss.AuthVersion(oss.AuthV4)}
	if region != "" {
		options = append(options, oss.Region(region))
	}
	ossClient, err := oss.New("https://"+endpoint, accessKeyID, accessKeySecret, options...)
	if err != nil {
		return nil, err
	}
	bucket, err := ossClient.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	if err := bucket.PutObject(strings.TrimLeft(filePathRemote, "/"), bytes.NewReader(input.Bytes), oss.ContentType(contentType)); err != nil {
		return nil, err
	}

	status, text, err = c.requestJSON(ctx, http.MethodPost, "/api/v2/files/parse", map[string]interface{}{"file_id": fileID}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("files/parse failed HTTP %d: %s", status, truncateQwenAIText(text, 500))
	}
	parseStatus, err := c.waitFileParsed(ctx, fileID, 60*time.Second)
	if err != nil {
		return nil, err
	}

	nowMillis := time.Now().UnixMilli()
	userID := ""
	if parts := strings.SplitN(strings.TrimLeft(filePathRemote, "/"), "/", 2); len(parts) >= 2 {
		userID = parts[0]
	}
	putURL := "https://" + bucketName + "." + endpoint + "/" + strings.TrimLeft(filePathRemote, "/")
	return map[string]interface{}{
		"type": "file",
		"file": map[string]interface{}{
			"created_at": nowMillis,
			"data":       map[string]interface{}{},
			"filename":   filename,
			"hash":       nil,
			"id":         fileID,
			"user_id":    userID,
			"meta": map[string]interface{}{
				"name":         filename,
				"size":         len(input.Bytes),
				"content_type": contentType,
				"parse_meta":   map[string]interface{}{"parse_status": parseStatus},
			},
			"update_at": nowMillis,
		},
		"id":              fileID,
		"url":             putURL,
		"name":            filename,
		"collection_name": "",
		"progress":        0,
		"status":          "uploaded",
		"greenNet":        "success",
		"size":            len(input.Bytes),
		"error":           "",
		"itemId":          qwenAIRandomID(),
		"file_type":       contentType,
		"showType":        "file",
		"file_class":      qwenAIFileClass(contentType),
		"uploadTaskId":    qwenAIRandomID(),
	}, nil
}

func (c *qwenAIClient) waitFileParsed(ctx context.Context, fileID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, text, err := c.requestJSON(ctx, http.MethodPost, "/api/v2/files/parse/status", map[string]interface{}{
			"file_id_list": []string{fileID},
		}, 30*time.Second)
		if err != nil {
			return "", err
		}
		if status != http.StatusOK {
			return "", fmt.Errorf("files/parse/status failed HTTP %d: %s", status, truncateQwenAIText(text, 500))
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			return "", err
		}
		rows, _ := payload["data"].([]interface{})
		row := map[string]interface{}{}
		if len(rows) > 0 {
			row, _ = rows[0].(map[string]interface{})
		}
		parseStatus := qwenAIString(row["status"])
		if parseStatus == "" {
			parseStatus = "pending"
		}
		switch strings.ToLower(parseStatus) {
		case "success", "succeeded", "completed":
			return parseStatus, nil
		case "failed", "error":
			return "", fmt.Errorf("file parse failed: %s", truncateQwenAIText(marshalCompact(row), 500))
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return "", fmt.Errorf("file parse timeout: %s", fileID)
}

func qwenAIChatPayload(chatID, model, content string, files []map[string]interface{}, chatType string, mediaOptions map[string]interface{}) map[string]interface{} {
	if chatType == "" {
		chatType = "t2t"
	}
	ts := time.Now().Unix()
	isImage := chatType == "image_gen" || chatType == "t2i"
	isVideo := chatType == "t2v"
	featureConfig := map[string]interface{}{}
	messageChatType := chatType
	subChatType := chatType
	ratio := qwenAIImageRatio(mediaOptions)
	messageMeta := map[string]interface{}{"subChatType": chatType}
	if isImage {
		featureConfig = map[string]interface{}{
			"thinking_enabled":     false,
			"output_schema":        "phase",
			"auto_thinking":        false,
			"thinking_mode":        "off",
			"auto_search":          false,
			"code_interpreter":     false,
			"function_calling":     false,
			"plugins_enabled":      true,
			"image_generation":     true,
			"default_aspect_ratio": ratio,
		}
		messageChatType = "t2t"
		subChatType = "t2i"
		messageMeta = map[string]interface{}{"subChatType": "t2i", "mode": "image_generation", "aspectRatio": ratio, "size": ratio}
	} else if isVideo {
		featureConfig = map[string]interface{}{
			"thinking_enabled":     false,
			"output_schema":        "phase",
			"auto_thinking":        false,
			"thinking_mode":        "off",
			"auto_search":          false,
			"code_interpreter":     false,
			"function_calling":     false,
			"plugins_enabled":      true,
			"video_generation":     true,
			"default_aspect_ratio": ratio,
		}
		messageChatType = "t2v"
		subChatType = "t2v"
		messageMeta = map[string]interface{}{"subChatType": "t2v", "mode": "video_generation", "aspectRatio": ratio, "size": ratio}
	} else {
		featureConfig = map[string]interface{}{
			"thinking_enabled":     true,
			"output_schema":        "phase",
			"research_mode":        "normal",
			"auto_thinking":        true,
			"thinking_mode":        "Auto",
			"thinking_format":      "summary",
			"auto_search":          false,
			"code_interpreter":     false,
			"plugins_enabled":      false,
			"function_calling":     false,
			"enable_tools":         false,
			"enable_function_call": false,
			"tool_choice":          "none",
		}
	}
	if files == nil {
		files = []map[string]interface{}{}
	}
	payload := map[string]interface{}{
		"stream":             true,
		"version":            "2.1",
		"incremental_output": true,
		"chat_id":            chatID,
		"chat_mode":          "normal",
		"model":              model,
		"parent_id":          nil,
		"messages": []map[string]interface{}{{
			"fid":            qwenAIRandomID(),
			"parentId":       nil,
			"childrenIds":    []string{qwenAIRandomID()},
			"role":           "user",
			"content":        content,
			"user_action":    "chat",
			"files":          files,
			"timestamp":      ts,
			"models":         []string{model},
			"chat_type":      messageChatType,
			"feature_config": featureConfig,
			"extra":          map[string]interface{}{"meta": messageMeta},
			"sub_chat_type":  subChatType,
			"parent_id":      nil,
		}},
		"timestamp": ts,
	}
	if isImage || isVideo {
		payload["size"] = ratio
	}
	return payload
}

func qwenAINormalizeChatType(chatType string) string {
	switch strings.TrimSpace(chatType) {
	case "image_gen", "t2i":
		return "t2i"
	case "t2v":
		return "t2v"
	default:
		if strings.TrimSpace(chatType) == "" {
			return "t2t"
		}
		return strings.TrimSpace(chatType)
	}
}

func qwenAIUpstreamModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "qwen3.6-plus"
	}
	lower := strings.ToLower(model)
	aliases := map[string]string{
		"qwen-image":                "qwen3.6-plus",
		"qwen-image-2":              "qwen3.6-plus",
		"qwen-image-2.0":            "qwen3.6-plus",
		"qwen-image-plus":           "qwen3.6-plus",
		"qwen-video":                "qwen3.6-plus",
		"qwen-video-plus":           "qwen3.6-plus",
		"happyhorse":                "qwen3.6-plus",
		"happyhorse 1.0":            "qwen3.6-plus",
		"tongyi-qwen3-max-model":    "qwen3.6-plus",
		"tongyi-qwen3-max-thinking": "qwen3.6-plus",
	}
	if mapped, ok := aliases[lower]; ok {
		return mapped
	}
	if strings.HasPrefix(lower, "qwen3.") || strings.HasPrefix(lower, "qwen2.") {
		return model
	}
	return "qwen3.6-plus"
}

func qwenAIImageRatio(options map[string]interface{}) string {
	for _, key := range []string{"ratio", "aspect_ratio", "aspectRatio", "size"} {
		if value, ok := options[key].(string); ok && strings.TrimSpace(value) != "" {
			return normalizeQwenAIRatio(value)
		}
	}
	return "1:1"
}

func normalizeQwenAIRatio(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "*", "x")
	if strings.Contains(value, ":") {
		return value
	}
	value = strings.ReplaceAll(value, "x", ":")
	switch value {
	case "1:1", "1024:1024", "1328:1328":
		return "1:1"
	case "16:9", "1792:1024", "1664:928", "1280:720":
		return "16:9"
	case "9:16", "1024:1792", "928:1664", "720:1280":
		return "9:16"
	case "4:3", "1472:1140":
		return "4:3"
	case "3:4", "1140:1472":
		return "3:4"
	default:
		return value
	}
}

func extractQwenAITaskIDs(text string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, match := range qwenAITaskIDPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			out = append(out, match[1])
		}
	}
	var value interface{}
	if json.Unmarshal([]byte(text), &value) == nil {
		collectQwenAITaskIDs(value, &out, seen)
	}
	return out
}

func collectQwenAITaskIDs(value interface{}, out *[]string, seen map[string]bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, v := range typed {
			lower := strings.ToLower(key)
			if lower == "task_id" || lower == "taskid" || lower == "wanx_task_id" || lower == "wanxtaskid" {
				id := qwenAIString(v)
				if id != "" && !seen[id] {
					seen[id] = true
					*out = append(*out, id)
				}
			}
			collectQwenAITaskIDs(v, out, seen)
		}
	case []interface{}:
		for _, item := range typed {
			collectQwenAITaskIDs(item, out, seen)
		}
	}
}

func qwenAITaskStatus(text string) string {
	var value interface{}
	if json.Unmarshal([]byte(text), &value) != nil {
		return ""
	}
	return findQwenAITaskStatus(value)
}

func findQwenAITaskStatus(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"status", "task_status", "taskStatus", "state"} {
			if status := qwenAIString(typed[key]); status != "" {
				return strings.ToLower(status)
			}
		}
		for _, v := range typed {
			if status := findQwenAITaskStatus(v); status != "" {
				return status
			}
		}
	case []interface{}:
		for _, item := range typed {
			if status := findQwenAITaskStatus(item); status != "" {
				return status
			}
		}
	}
	return ""
}

func qwenAITaskStatusIsRunning(status string) bool {
	switch strings.ToLower(status) {
	case "", "running", "pending", "queued", "processing", "created":
		return true
	default:
		return false
	}
}

func qwenAITaskStatusIsSuccess(status string) bool {
	switch strings.ToLower(status) {
	case "success", "succeeded", "finished", "completed":
		return true
	default:
		return false
	}
}

func dedupeQwenAIFiles(files []map[string]interface{}) []map[string]interface{} {
	out := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, file := range files {
		key := qwenAIString(file["id"])
		if key == "" {
			key = marshalCompact(file)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, file)
	}
	return out
}

func qwenAIString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case json.Number:
		return typed.String()
	}
	return ""
}

func qwenAIExtForContentType(contentType string) string {
	if ext, err := mime.ExtensionsByType(contentType); err == nil && len(ext) > 0 {
		return ext[0]
	}
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func qwenAIFileClass(contentType string) string {
	lower := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(lower, "image/"):
		return "image"
	case strings.HasPrefix(lower, "audio/"):
		return "audio"
	case strings.HasPrefix(lower, "video/"):
		return "video"
	default:
		return "document"
	}
}

func qwenAINormalizeOSSRegion(region string) string {
	region = strings.TrimSpace(region)
	region = strings.TrimPrefix(region, "oss-")
	return region
}

func qwenAIRandomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return generateRandomHex(16)
	}
	return hex.EncodeToString(buf)
}

func summarizeQwenAIRaw(raw string) string {
	raw = html.UnescapeString(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, `\/`, `/`)
	raw = strings.Join(strings.Fields(raw), " ")
	return truncateQwenAIText(raw, 500)
}

func truncateQwenAIText(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
