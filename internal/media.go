package internal

import (
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

	account, err := AppStore.SelectAccountForCapability("image")
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusFailedDependency, "no_valid_image_account", "No valid image-capable qianwen.com login account is available. Add an account in Admin and complete a real image test first.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "account_select_failed", err.Error())
		return
	}

	task, err := createProtocolRequiredTask("image", req.Model, account.ID, req, "qianwen_image_protocol_required", "qianwen.com image generation protocol has not been captured yet. Use Admin account login/protocol capture before enabling image generation.")
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "task_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"error": ErrorDetail{
			Code:    task.ErrorCode,
			Message: task.ErrorMessage,
			Type:    "qianwen_web_error",
		},
		"task_id": task.ID,
		"status":  task.Status,
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
		req.Duration = 5
	}

	account, err := AppStore.SelectAccountForCapability("video")
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusFailedDependency, "no_valid_video_account", "No valid video-capable qianwen.com login account is available. Add an account in Admin and complete a real video test first.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "account_select_failed", err.Error())
		return
	}

	task, err := createProtocolRequiredTask("video", req.Model, account.ID, req, "qianwen_video_protocol_required", "qianwen.com video generation protocol has not been captured yet. Use Admin account login/protocol capture before enabling video generation.")
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "task_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"id":      task.ID,
		"object":  "video.generation",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"status":  task.Status,
		"error": ErrorDetail{
			Code:    task.ErrorCode,
			Message: task.ErrorMessage,
			Type:    "qianwen_web_error",
		},
	})
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

func parseUnix(value string) int64 {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().Unix()
	}
	return t.Unix()
}
