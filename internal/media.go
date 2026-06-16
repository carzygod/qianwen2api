package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ImageGenerationRequest struct {
	Model          string      `json:"model"`
	Prompt         string      `json:"prompt"`
	N              int         `json:"n"`
	Size           string      `json:"size"`
	Quality        string      `json:"quality"`
	Style          string      `json:"style"`
	ResponseFormat string      `json:"response_format"`
	AspectRatio    string      `json:"aspect_ratio"`
	Resolution     string      `json:"resolution"`
	Seed           interface{} `json:"seed"`
	NegativePrompt string      `json:"negative_prompt"`
	ReferenceImage string      `json:"reference_image"`
}

type VideoGenerationRequest struct {
	Model           string      `json:"model"`
	Prompt          string      `json:"prompt"`
	Duration        int         `json:"duration"`
	AspectRatio     string      `json:"aspect_ratio"`
	Resolution      string      `json:"resolution"`
	FirstFrameImage string      `json:"first_frame_image"`
	ReferenceImages []string    `json:"reference_images"`
	Seed            interface{} `json:"seed"`
	NegativePrompt  string      `json:"negative_prompt"`
	Metadata        interface{} `json:"metadata"`
}

func HandleImageGenerations(w http.ResponseWriter, r *http.Request) {
	if !requireAPIAuth(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is supported.")
		return
	}
	var req ImageGenerationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt_required", "prompt is required.")
		return
	}
	if req.Model == "" {
		req.Model = Cfg.DefaultImageModel
	}
	if req.N == 0 {
		req.N = 1
	}

	account, err := AppStore.SelectRunnableAccountForCapability("image")
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusFailedDependency, "no_image_account", "No image-capable qianwen.com login account is available. Add an account in Admin first.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "account_select_failed", err.Error())
		return
	}

	client, err := newQwenWebClient(*account)
	if err != nil {
		writeAPIError(w, http.StatusFailedDependency, "login_account_invalid", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
	defer cancel()
	state, events, err := client.submitImage(ctx, req)
	if err != nil {
		_ = AppStore.UpdateAccountStatus(account.ID, "unknown", err.Error(), false)
		writeAPIError(w, http.StatusBadGateway, "qianwen_image_submit_failed", err.Error())
		return
	}
	urls := filterMediaURLs(extractURLs(events), "image")
	if len(urls) == 0 {
		result, err := client.pollMedia(ctx, state, "image", 130*time.Second)
		if err != nil {
			_ = AppStore.UpdateAccountStatus(account.ID, "unknown", err.Error(), false)
			writeAPIError(w, http.StatusGatewayTimeout, "qianwen_image_poll_failed", err.Error())
			return
		}
		urls = result.URLs
	}
	urls = limitURLs(urls, req.N)
	if len(urls) == 0 {
		writeAPIError(w, http.StatusBadGateway, "qianwen_image_empty_result", "Qianwen image generation completed without a media URL.")
		return
	}
	_ = AppStore.UpdateAccountStatus(account.ID, "valid", "", true)
	data := make([]map[string]string, 0, len(urls))
	for _, url := range urls {
		data = append(data, map[string]string{"url": url})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"created": time.Now().Unix(),
		"data":    data,
	})
}

func HandleVideoGenerations(w http.ResponseWriter, r *http.Request) {
	if !requireAPIAuth(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is supported.")
		return
	}
	var req VideoGenerationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeAPIError(w, http.StatusBadRequest, "prompt_required", "prompt is required.")
		return
	}
	if req.Model == "" {
		req.Model = Cfg.DefaultVideoModel
	}
	if req.Duration == 0 {
		req.Duration = 10
	}

	account, err := AppStore.SelectRunnableAccountForCapability("video")
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusFailedDependency, "no_video_account", "No video-capable qianwen.com login account is available. Add an account in Admin first.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "account_select_failed", err.Error())
		return
	}

	client, err := newQwenWebClient(*account)
	if err != nil {
		writeAPIError(w, http.StatusFailedDependency, "login_account_invalid", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	state, events, err := client.submitVideo(ctx, req)
	if err != nil {
		_ = AppStore.UpdateAccountStatus(account.ID, "unknown", err.Error(), false)
		writeAPIError(w, http.StatusBadGateway, "qianwen_video_submit_failed", err.Error())
		return
	}
	body, _ := json.Marshal(req)
	task := &TaskRecord{
		Type:                   "video",
		Status:                 "processing",
		Model:                  req.Model,
		ProviderAccountID:      account.ID,
		UpstreamTaskID:         state.ReqID,
		UpstreamConversationID: state.SessionID,
		RequestJSON:            string(body),
		UpstreamRequestJSON:    marshalCompact(state),
		UpstreamResponseJSON:   marshalCompact(events),
		StartedAt:              nowISO(),
	}
	if err := AppStore.CreateTask(task); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "task_create_failed", err.Error())
		return
	}
	go pollVideoTask(task.ID, *account, state)
	_ = AppStore.UpdateAccountStatus(account.ID, "valid", "", true)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      task.ID,
		"object":  "video.generation",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"status":  task.Status,
	})
}

func pollVideoTask(taskID string, account AccountRecord, state *qwenRequestState) {
	client, err := newQwenWebClient(account)
	if err != nil {
		_ = AppStore.UpdateTaskFailed(taskID, "login_account_invalid", err.Error(), "")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	result, err := client.pollMedia(ctx, state, "video", 7*time.Minute)
	if err != nil {
		_ = AppStore.UpdateTaskFailed(taskID, "qianwen_video_poll_failed", err.Error(), marshalCompact(result))
		return
	}
	_ = AppStore.UpdateTaskCompleted(taskID, marshalCompact(map[string]interface{}{
		"urls": result.URLs,
	}), marshalCompact(result.Events))
}

func HandleVideoTask(w http.ResponseWriter, r *http.Request) {
	if !requireAPIAuth(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/video/generations/")
	id = strings.TrimPrefix(id, "/v1/videos/generations/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeAPIError(w, http.StatusNotFound, "task_id_required", "Task id is required.")
		return
	}
	task, err := AppStore.GetTask(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "task_not_found", "Task not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "task_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, normalizeVideoTaskResponse(task))
}

func HandleGenericTask(w http.ResponseWriter, r *http.Request) {
	if !requireAPIAuth(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	id = strings.Trim(id, "/")
	task, err := AppStore.GetTask(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "task_not_found", "Task not found.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "task_get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func createProtocolRequiredTask(taskType, model, accountID string, req interface{}, code, message string) (*TaskRecord, error) {
	body, _ := json.Marshal(req)
	task := &TaskRecord{
		Type:              taskType,
		Status:            "failed",
		Model:             model,
		ProviderAccountID: accountID,
		RequestJSON:       string(body),
		ErrorCode:         code,
		ErrorMessage:      message,
		CompletedAt:       nowISO(),
	}
	if err := AppStore.CreateTask(task); err != nil {
		return nil, err
	}
	return task, nil
}

func normalizeVideoTaskResponse(task *TaskRecord) map[string]interface{} {
	resp := map[string]interface{}{
		"id":      task.ID,
		"object":  "video.generation",
		"created": parseUnix(task.CreatedAt),
		"model":   task.Model,
		"status":  task.Status,
	}
	if task.ResultJSON != "" {
		var result interface{}
		if json.Unmarshal([]byte(task.ResultJSON), &result) == nil {
			resp["data"] = result
		}
	}
	if task.ErrorCode != "" || task.ErrorMessage != "" {
		resp["error"] = ErrorDetail{
			Code:    task.ErrorCode,
			Message: task.ErrorMessage,
			Type:    "qianwen_web_error",
		}
	}
	return resp
}

func limitURLs(urls []string, n int) []string {
	if n <= 0 {
		n = 1
	}
	if len(urls) <= n {
		return urls
	}
	return urls[:n]
}

func parseUnix(value string) int64 {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().Unix()
	}
	return t.Unix()
}
