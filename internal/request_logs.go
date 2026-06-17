package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RequestLogRecord struct {
	TS        int64  `json:"ts"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Model     string `json:"model,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Status    int    `json:"status"`
	MS        int64  `json:"ms"`
}

var requestLogs = struct {
	sync.Mutex
	items []RequestLogRecord
}{items: []RequestLogRecord{}}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func WithRequestLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		model, accountID := requestLogFields(r)
		recorder := &statusRecorder{ResponseWriter: w}
		next(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		AddRequestLog(RequestLogRecord{
			TS:        start.Unix(),
			Method:    r.Method,
			Path:      r.URL.Path,
			Model:     model,
			AccountID: accountID,
			Status:    status,
			MS:        time.Since(start).Milliseconds(),
		})
	}
}

func AddRequestLog(record RequestLogRecord) {
	requestLogs.Lock()
	defer requestLogs.Unlock()
	requestLogs.items = append(requestLogs.items, record)
	if len(requestLogs.items) > 200 {
		requestLogs.items = requestLogs.items[len(requestLogs.items)-200:]
	}
}

func ListRequestLogs() []RequestLogRecord {
	requestLogs.Lock()
	defer requestLogs.Unlock()
	out := make([]RequestLogRecord, len(requestLogs.items))
	copy(out, requestLogs.items)
	return out
}

func requestLogFields(r *http.Request) (string, string) {
	if r.Body == nil {
		return "", ""
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
		return "", ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 {
		return "", ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", ""
	}
	return stringField(payload, "model"), stringField(payload, "account_id")
}

func stringField(payload map[string]interface{}, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(toString(value))
}

func toString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}
