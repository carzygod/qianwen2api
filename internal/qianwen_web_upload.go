package internal

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	cdpRuntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

const qwenWorkspaceAPIURL = "https://workspace-res.qianwen.com"

type qwenWebOSSHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type qwenWebOSSTokenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Host          string             `json:"host"`
		Object        string             `json:"object"`
		Bucket        string             `json:"bucket"`
		Endpoint      string             `json:"endpoint"`
		Authorization string             `json:"authorization"`
		OSSHeaders    []qwenWebOSSHeader `json:"oss_headers"`
	} `json:"data"`
}

type qwenWebOSSCallbackResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		FileName       string `json:"file_name"`
		FileType       string `json:"file_type"`
		FileSize       int64  `json:"file_size"`
		WSGID          string `json:"ws_gid"`
		MaterialCDNURL string `json:"material_cdn_url"`
		MaterialURL    string `json:"material_url"`
	} `json:"data"`
}

type qwenWebRuntimeUploadResult struct {
	FileName   string `json:"file_name"`
	FileFormat string `json:"file_format"`
	FileSize   int64  `json:"file_size"`
	ID         string `json:"id"`
	URL        string `json:"url"`
}

func (c *qwenWebClient) uploadImageMaterial(ctx context.Context, source string) (qwenImageResource, error) {
	input, err := qwenAIResolveInputFile(ctx, source)
	if err != nil {
		return qwenImageResource{}, err
	}
	if input.RemoteRef != nil {
		return qwenImageResource{}, fmt.Errorf("qianwen.com web video does not accept chat.qwen.ai remote file JSON; use image URL/data URI/base64 or qianwen material id")
	}
	if len(input.Bytes) == 0 {
		return qwenImageResource{}, fmt.Errorf("image material is empty")
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = http.DetectContentType(input.Bytes)
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return qwenImageResource{}, fmt.Errorf("qianwen.com video material must be an image, got %s", contentType)
	}
	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		filename = "input" + qwenAIExtForContentType(contentType)
	}
	if filepath.Ext(filename) == "" {
		filename += qwenAIExtForContentType(contentType)
	}
	input.Filename = filename
	input.ContentType = contentType

	resource, browserErr := c.uploadImageMaterialViaBrowser(ctx, input)
	if browserErr == nil {
		return resource, nil
	}
	LogWarn("Qianwen browser material upload failed, falling back to direct workspace upload: %v", browserErr)
	resource, directErr := c.uploadImageMaterialDirect(ctx, input)
	if directErr == nil {
		return resource, nil
	}
	return qwenImageResource{}, fmt.Errorf("qianwen browser material upload failed: %v; direct workspace fallback failed: %w", browserErr, directErr)
}

func (c *qwenWebClient) uploadImageMaterialDirect(ctx context.Context, input qwenAIInputFile) (qwenImageResource, error) {
	filename := strings.TrimSpace(input.Filename)
	md5Sum := md5.Sum(input.Bytes)
	contentMD5 := base64.StdEncoding.EncodeToString(md5Sum[:])
	tokenReq := map[string]interface{}{
		"file_name":    filename,
		"content_type": "application/octet-stream",
		"content_md5":  contentMD5,
		"size":         len(input.Bytes),
	}
	var tokenResp qwenWebOSSTokenResponse
	if err := c.postWorkspaceJSON(ctx, "/1/oss_token", nil, tokenReq, &tokenResp); err != nil {
		return qwenImageResource{}, err
	}
	if tokenResp.Code != 0 {
		return qwenImageResource{}, fmt.Errorf("qianwen oss_token failed code=%d msg=%s", tokenResp.Code, tokenResp.Msg)
	}
	if strings.TrimSpace(tokenResp.Data.Host) == "" || strings.TrimSpace(tokenResp.Data.Object) == "" || strings.TrimSpace(tokenResp.Data.Authorization) == "" {
		return qwenImageResource{}, fmt.Errorf("qianwen oss_token missing upload fields")
	}

	if err := c.putWorkspaceObject(ctx, tokenResp, input.Bytes, contentMD5); err != nil {
		return qwenImageResource{}, err
	}

	callbackReqID := uuid.New().String()
	callbackReq := map[string]interface{}{
		"file_md5":  contentMD5,
		"file_name": filename,
		"file_type": strings.ToUpper(strings.TrimPrefix(filepath.Ext(filename), ".")),
		"bucket":    tokenResp.Data.Bucket,
		"endpoint":  tokenResp.Data.Endpoint,
		"object":    tokenResp.Data.Object,
		"entry":     "qwen_pc",
	}
	var callbackResp qwenWebOSSCallbackResponse
	if err := c.postWorkspaceJSON(ctx, "/1/oss/callback", []queryPair{
		{Key: "req_id", Value: callbackReqID},
	}, callbackReq, &callbackResp); err != nil {
		return qwenImageResource{}, err
	}
	if callbackResp.Code != 0 {
		return qwenImageResource{}, fmt.Errorf("qianwen oss_callback failed code=%d msg=%s", callbackResp.Code, callbackResp.Msg)
	}
	if strings.TrimSpace(callbackResp.Data.WSGID) == "" {
		return qwenImageResource{}, fmt.Errorf("qianwen oss_callback returned empty material id")
	}
	width, height := imageDimensions(input.Bytes)
	return qwenImageResource{
		ID:         callbackResp.Data.WSGID,
		URL:        defaultString(callbackResp.Data.MaterialCDNURL, callbackResp.Data.MaterialURL),
		Width:      width,
		Height:     height,
		FileFormat: strings.ToLower(callbackResp.Data.FileType),
		FileName:   defaultString(callbackResp.Data.FileName, filename),
		FileSize:   callbackResp.Data.FileSize,
	}, nil
}

func (c *qwenWebClient) uploadImageMaterialViaBrowser(ctx context.Context, input qwenAIInputFile) (qwenImageResource, error) {
	uploadCtx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("single-process", true),
		chromedp.UserAgent(c.userAgent),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(uploadCtx, opts...)
	defer allocCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...interface{}) {
		LogDebug("[qianwen-upload] "+format, args...)
	}))
	defer browserCancel()

	arg := map[string]string{
		"name": strings.TrimSpace(input.Filename),
		"type": strings.TrimSpace(input.ContentType),
		"b64":  base64.StdEncoding.EncodeToString(input.Bytes),
	}
	argJSON, _ := json.Marshal(arg)
	expression := fmt.Sprintf(`(async () => {
		const arg = %s;
		if (!window.webpackChunk_ali_qianwen_web) {
			throw new Error("qianwen webpack runtime is not ready");
		}
		let __req;
		window.webpackChunk_ali_qianwen_web.push([[Math.random()], {}, (r) => { __req = r; }]);
		const mod = __req(96152);
		if (!mod || typeof mod.IF !== "function") {
			throw new Error("qianwen upload module is unavailable");
		}
		const raw = Uint8Array.from(atob(arg.b64), c => c.charCodeAt(0));
		const file = new File([raw], arg.name || "input.png", { type: arg.type || "image/png" });
		const result = await mod.IF({ file });
		return JSON.stringify(result || {});
	})()`, string(argJSON))

	var resultJSON string
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(qwenWebBaseURL+"/"),
		chromedp.WaitReady("body"),
		chromedp.Sleep(4*time.Second),
		chromedp.Evaluate(expression, &resultJSON, func(p *cdpRuntime.EvaluateParams) *cdpRuntime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	); err != nil {
		return qwenImageResource{}, err
	}
	var result qwenWebRuntimeUploadResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return qwenImageResource{}, fmt.Errorf("decode qianwen runtime upload result: %w: %s", err, truncateQwenAIText(resultJSON, 500))
	}
	if strings.TrimSpace(result.ID) == "" {
		return qwenImageResource{}, fmt.Errorf("qianwen runtime upload returned empty material id: %s", truncateQwenAIText(resultJSON, 500))
	}
	width, height := imageDimensions(input.Bytes)
	return qwenImageResource{
		ID:         result.ID,
		URL:        result.URL,
		Width:      width,
		Height:     height,
		FileFormat: strings.ToLower(result.FileFormat),
		FileName:   defaultString(result.FileName, input.Filename),
		FileSize:   result.FileSize,
	}, nil
}

func (c *qwenWebClient) postWorkspaceJSON(ctx context.Context, path string, extraPairs []queryPair, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	signedURL, _, headers, err := loginACSAuth.signedURLAndHeadersForScene(c.account, qwenWorkspaceAPIURL, path, extraPairs, body, qwenWebAuthWorkspaceScene)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signedURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", qwenWebBaseURL)
	req.Header.Set("Referer", qwenWebBaseURL+"/")
	req.Header.Set("Cookie", c.cookieHeader)
	req.Header.Set("x-platform", "pc_tongyi")
	if c.xsrfToken != "" {
		req.Header.Set("x-csrf-token", c.xsrfToken)
	}
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qianwen workspace %s HTTP %d: %s", path, resp.StatusCode, truncateQwenAIText(string(raw), 800))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode qianwen workspace %s response: %w: %s", path, err, truncateQwenAIText(string(raw), 800))
	}
	return nil
}

func (c *qwenWebClient) putWorkspaceObject(ctx context.Context, token qwenWebOSSTokenResponse, raw []byte, contentMD5 string) error {
	host := strings.TrimSpace(token.Data.Host)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	uploadURL := strings.TrimRight(host, "/") + "/" + url.PathEscape(strings.TrimLeft(token.Data.Object, "/"))
	params := url.Values{}
	params.Set("req_id", uuid.New().String())
	params.Set("biz_id", "ai_qwen")
	uploadURL += "?" + params.Encode()

	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Md5", contentMD5)
	for _, header := range token.Data.OSSHeaders {
		if header.Key != "" && header.Value != "" {
			req.Header.Set(header.Key, header.Value)
		}
	}
	if signedMD5 := tokenContentMD5(token); signedMD5 != "" {
		req.Header.Set("Content-Md5", signedMD5)
	}
	req.Header.Set("authorization", token.Data.Authorization)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("qianwen oss PUT HTTP %d: %s", resp.StatusCode, truncateQwenAIText(string(rawResp), 500))
	}
	return nil
}

func tokenContentMD5(token qwenWebOSSTokenResponse) string {
	for _, header := range token.Data.OSSHeaders {
		if strings.EqualFold(header.Key, "Content-Md5") {
			return header.Value
		}
	}
	return ""
}

func imageDimensions(raw []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}
