package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

type LoginSession struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	AccountID     string `json:"account_id,omitempty"`
	CookieCount   int    `json:"cookie_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ScreenshotURL string `json:"screenshot_url,omitempty"`

	userDataDir string
	ctx         context.Context
	cancel      context.CancelFunc
	screenshot  []byte
	mu          sync.Mutex
}

type LoginSessionView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	AccountID     string `json:"account_id,omitempty"`
	CookieCount   int    `json:"cookie_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ScreenshotURL string `json:"screenshot_url,omitempty"`
}

type LoginSessionManager struct {
	mu       sync.Mutex
	sessions map[string]*LoginSession
}

var QianwenLoginSessions = &LoginSessionManager{sessions: map[string]*LoginSession{}}

func (m *LoginSessionManager) Start(name string) (*LoginSessionView, error) {
	if strings.TrimSpace(name) == "" {
		name = "qianwen-login-" + time.Now().Format("20060102-150405")
	}
	id := uuid.New().String()
	session := &LoginSession{
		ID:          id,
		Name:        name,
		Status:      "starting",
		Message:     "Starting qianwen.com login browser.",
		CreatedAt:   nowISO(),
		UpdatedAt:   nowISO(),
		userDataDir: filepath.Join(Cfg.DataDir, "login-sessions", id),
	}
	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()
	go session.run()
	return session.view(), nil
}

func (m *LoginSessionManager) List() []LoginSessionView {
	m.mu.Lock()
	defer m.mu.Unlock()
	views := make([]LoginSessionView, 0, len(m.sessions))
	for _, session := range m.sessions {
		views = append(views, *session.view())
	}
	return views
}

func (m *LoginSessionManager) Get(id string) (*LoginSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *LoginSessionManager) Delete(id string) bool {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if ok {
		session.stop()
	}
	return ok
}

func (s *LoginSession) view() *LoginSessionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &LoginSessionView{
		ID:            s.ID,
		Name:          s.Name,
		Status:        s.Status,
		Message:       s.Message,
		AccountID:     s.AccountID,
		CookieCount:   s.CookieCount,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		ScreenshotURL: "/api/login-sessions/" + s.ID + "/screenshot",
	}
}

func (s *LoginSession) setStatus(status, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	s.Message = message
	s.UpdatedAt = nowISO()
}

func (s *LoginSession) stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *LoginSession) run() {
	if err := os.MkdirAll(s.userDataDir, 0700); err != nil {
		s.setStatus("failed", "Failed to create login profile directory: "+err.Error())
		return
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.UserDataDir(s.userDataDir),
		chromedp.WindowSize(1280, 980),
		chromedp.UserAgent(generateRandomUserAgent()),
	)
	if runtime.GOOS != "windows" {
		opts = append(opts, chromedp.Flag("single-process", true))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...interface{}) {
		LogDebug("[qianwen-login] "+format, args...)
	}))
	cancel := func() {
		ctxCancel()
		allocCancel()
	}
	s.mu.Lock()
	s.ctx = ctx
	s.cancel = cancel
	s.mu.Unlock()

	s.setStatus("opening", "Opening qianwen.com. If a QR code appears, scan it with the qianwen/Taobao/Alipay login flow shown on the page.")
	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate("https://www.qianwen.com/"),
		chromedp.Sleep(4*time.Second),
	); err != nil {
		s.setStatus("failed", "Failed to open qianwen.com: "+err.Error())
		return
	}

	_ = clickVisibleLogin(ctx)
	_ = s.RefreshScreenshot()
	s.setStatus("waiting_scan", "Scan the QR code in the screenshot, then click Capture Login in Admin after the page changes to a logged-in state.")

	ticker := time.NewTicker(6 * time.Second)
	expire := time.NewTimer(10 * time.Minute)
	defer ticker.Stop()
	defer expire.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.RefreshScreenshot()
			count, _ := s.countCookies()
			s.mu.Lock()
			s.CookieCount = count
			if s.Status != "captured" {
				s.UpdatedAt = nowISO()
			}
			s.mu.Unlock()
		case <-expire.C:
			s.setStatus("expired", "Login session expired. Start a new QR login session.")
			s.stop()
			return
		case <-ctx.Done():
			return
		}
	}
}

func clickVisibleLogin(ctx context.Context) error {
	var clicked bool
	script := `(() => {
  const textRe = /(登录|登陆|Sign in|Log in)/i;
  const nodes = Array.from(document.querySelectorAll('button,a,div,span'));
  const isVisible = (el) => {
    const rect = el.getBoundingClientRect();
    const style = getComputedStyle(el);
    return rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
  };
  const el = nodes.find((node) => isVisible(node) && textRe.test((node.innerText || node.textContent || '').trim()));
  if (el) {
    el.click();
    return true;
  }
  return false;
})()`
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(script, &clicked),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	if clicked {
		LogInfo("Clicked a visible qianwen login trigger")
	}
	return nil
}

func (s *LoginSession) RefreshScreenshot() error {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return fmt.Errorf("login browser is not ready")
	}
	var image []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&image, 90)); err != nil {
		return err
	}
	s.mu.Lock()
	s.screenshot = image
	s.UpdatedAt = nowISO()
	s.mu.Unlock()
	return nil
}

func (s *LoginSession) Screenshot() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, len(s.screenshot))
	copy(out, s.screenshot)
	return out
}

func (s *LoginSession) CaptureAccount() (*AccountRecord, error) {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return nil, fmt.Errorf("login browser is not ready")
	}

	cookies, err := network.GetCookies().WithUrls([]string{
		"https://www.qianwen.com/",
		"https://qianwen.com/",
		"https://api.qianwen.com/",
		"https://passport.aliyun.com/",
		"https://login.taobao.com/",
	}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("read cookies: %w", err)
	}
	cookieJSON, cookieString, err := serializeCookies(cookies)
	if err != nil {
		return nil, err
	}
	var localStorageJSON string
	_ = chromedp.Run(ctx, chromedp.Evaluate(`JSON.stringify(Object.fromEntries(Object.entries(localStorage)))`, &localStorageJSON))
	var userAgent string
	_ = chromedp.Run(ctx, chromedp.Evaluate(`navigator.userAgent`, &userAgent))

	account := &AccountRecord{
		Name:             s.Name,
		Type:             "login_cookie",
		Status:           "unknown",
		Enabled:          true,
		CookieJSON:       cookieJSON,
		CookieString:     cookieString,
		LocalStorageJSON: localStorageJSON,
		UserAgent:        userAgent,
		CapabilitiesJSON: `{"chat":true,"image":true,"video":true}`,
		LastError:        "QR login cookies captured. Real model probe is still required before this account is marked valid.",
	}
	if err := AppStore.CreateAccount(account); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.Status = "captured"
	s.Message = "Captured browser cookies into account pool. Run a real model test after the qianwen.com protocol adapter is implemented."
	s.AccountID = account.ID
	s.CookieCount = len(cookies)
	s.UpdatedAt = nowISO()
	s.mu.Unlock()
	return account, nil
}

func (s *LoginSession) countCookies() (int, error) {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return 0, fmt.Errorf("login browser is not ready")
	}
	cookies, err := network.GetCookies().WithUrls([]string{"https://www.qianwen.com/", "https://api.qianwen.com/"}).Do(ctx)
	if err != nil {
		return 0, err
	}
	return len(cookies), nil
}

func serializeCookies(cookies []*network.Cookie) (string, string, error) {
	type cookieItem struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires,omitempty"`
		HTTPOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite,omitempty"`
	}
	items := make([]cookieItem, 0, len(cookies))
	pairs := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" {
			continue
		}
		item := cookieItem{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  float64(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			SameSite: string(cookie.SameSite),
		}
		items = append(items, item)
		pairs = append(pairs, cookie.Name+"="+cookie.Value)
	}
	body, err := json.Marshal(items)
	if err != nil {
		return "", "", err
	}
	return string(body), strings.Join(pairs, "; "), nil
}

func handleLoginSessions(w http.ResponseWriter, r *http.Request, path string) {
	if path == "/login-sessions" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]interface{}{"data": QianwenLoginSessions.List()})
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			session, err := QianwenLoginSessions.Start(body.Name)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "login_session_start_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, map[string]interface{}{"data": session})
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		}
		return
	}

	suffix := strings.TrimPrefix(path, "/login-sessions/")
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, "login_session_not_found", "Login session not found.")
		return
	}
	session, ok := QianwenLoginSessions.Get(parts[0])
	if !ok {
		writeAPIError(w, http.StatusNotFound, "login_session_not_found", "Login session not found.")
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]interface{}{"data": session.view()})
		case http.MethodDelete:
			QianwenLoginSessions.Delete(parts[0])
			writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		}
		return
	}

	switch parts[1] {
	case "screenshot":
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
			return
		}
		image := session.Screenshot()
		if len(image) == 0 {
			writeAPIError(w, http.StatusNotFound, "screenshot_not_ready", "Screenshot is not ready yet.")
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(image)
	case "refresh":
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
			return
		}
		if err := session.RefreshScreenshot(); err != nil {
			writeAPIError(w, http.StatusFailedDependency, "screenshot_refresh_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": session.view()})
	case "capture":
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
			return
		}
		account, err := session.CaptureAccount()
		if err != nil {
			writeAPIError(w, http.StatusFailedDependency, "login_capture_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": maskAccount(*account), "session": session.view()})
	default:
		writeAPIError(w, http.StatusNotFound, "login_session_route_not_found", "Login session route not found.")
	}
}
