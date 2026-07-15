package internal

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

type capturedCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite,omitempty"`
}

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
	Mode          string `json:"mode"`
	NoVNCURL      string `json:"novnc_url,omitempty"`

	userDataDir       string
	targetAccountID   string
	existingAccountID string
	profilePersistent bool
	leaseOwner        string
	ctx               context.Context
	cancel            context.CancelFunc
	screenshot        []byte
	runGeneration     uint64
	captureMu         sync.Mutex
	mu                sync.Mutex
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
	Mode          string `json:"mode"`
	NoVNCURL      string `json:"novnc_url,omitempty"`
}

type LoginSessionManager struct {
	mu       sync.Mutex
	sessions map[string]*LoginSession
}

func (m *LoginSessionManager) register(session *LoginSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if Cfg.NoVNCURL != "" {
		for _, existing := range m.sessions {
			existing.mu.Lock()
			status := existing.Status
			browserActive := existing.ctx != nil
			existing.mu.Unlock()
			if browserActive || (status != "captured" && status != "failed" && status != "expired") {
				return fmt.Errorf("another interactive login session is already active")
			}
		}
	}
	m.sessions[session.ID] = session
	return nil
}

var QianwenLoginSessions = &LoginSessionManager{sessions: map[string]*LoginSession{}}

func (m *LoginSessionManager) Start(name string) (*LoginSessionView, error) {
	if strings.TrimSpace(name) == "" {
		name = "qianwen-login-" + time.Now().Format("20060102-150405")
	}
	id := uuid.New().String()
	accountID := uuid.New().String()
	session := &LoginSession{
		ID:          id,
		Name:        name,
		Status:      "starting",
		Message:     "正在启动 qianwen.com 登录浏览器。",
		CreatedAt:   nowISO(),
		UpdatedAt:   nowISO(),
		Mode:            "new_account",
		NoVNCURL:        Cfg.NoVNCURL,
		userDataDir:     accountProfilePath(accountID),
		targetAccountID: accountID,
	}
	if err := m.register(session); err != nil {
		return nil, err
	}
	session.launch()
	return session.view(), nil
}

func (m *LoginSessionManager) StartMaintenance(accountID string) (*LoginSessionView, error) {
	account, err := AppStore.GetAccount(strings.TrimSpace(accountID))
	if err != nil {
		return nil, err
	}
	id := uuid.New().String()
	if _, err := AppStore.BeginAccountMaintenance(account.ID, id, defaultMaintenanceLease); err != nil {
		return nil, err
	}
	session := &LoginSession{
		ID:                id,
		Name:              account.Name,
		Status:            "starting",
		Message:           "Starting account maintenance browser.",
		CreatedAt:         nowISO(),
		UpdatedAt:         nowISO(),
		Mode:              "maintenance",
		NoVNCURL:          Cfg.NoVNCURL,
		userDataDir:       accountProfilePath(account.ID),
		targetAccountID:   account.ID,
		existingAccountID: account.ID,
		profilePersistent: true,
		leaseOwner:        id,
	}
	if err := m.register(session); err != nil {
		_ = AppStore.EndAccountMaintenance(account.ID, id, err.Error())
		return nil, err
	}
	session.launch()
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

func (m *LoginSessionManager) LatestActive() (*LoginSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *LoginSession
	for _, session := range m.sessions {
		session.mu.Lock()
		status := session.Status
		createdAt := session.CreatedAt
		session.mu.Unlock()
		if status == "captured" || status == "failed" || status == "expired" {
			continue
		}
		if latest == nil {
			latest = session
			continue
		}
		latest.mu.Lock()
		latestCreatedAt := latest.CreatedAt
		latest.mu.Unlock()
		if createdAt > latestCreatedAt {
			latest = session
		}
	}
	return latest, latest != nil
}

func (m *LoginSessionManager) LatestOrStart() (*LoginSession, error) {
	if session, ok := m.LatestActive(); ok {
		return session, nil
	}
	view, err := m.Start("qianwen-auth-" + time.Now().Format("20060102-150405"))
	if err != nil {
		return nil, err
	}
	session, ok := m.Get(view.ID)
	if !ok {
		return nil, fmt.Errorf("login session disappeared after creation")
	}
	return session, nil
}

func (m *LoginSessionManager) Delete(id string) bool {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if ok {
		session.releaseBrowser()
	}
	return ok
}

func (m *LoginSessionManager) DeleteByAccountID(accountID string) []string {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil
	}

	type target struct {
		id      string
		session *LoginSession
	}
	targets := []target{}

	m.mu.Lock()
	for id, session := range m.sessions {
		session.mu.Lock()
		matches := session.AccountID == accountID || session.targetAccountID == accountID
		session.mu.Unlock()
		if matches {
			delete(m.sessions, id)
			targets = append(targets, target{id: id, session: session})
		}
	}
	m.mu.Unlock()

	ids := make([]string, 0, len(targets))
	for _, item := range targets {
		item.session.releaseBrowser()
		ids = append(ids, item.id)
	}
	return ids
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
		Mode:          s.Mode,
		NoVNCURL:      s.NoVNCURL,
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
	s.ctx = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *LoginSession) cleanupProfile() {
	s.mu.Lock()
	persistent := s.profilePersistent
	profile := s.userDataDir
	s.mu.Unlock()
	if !persistent && profile != "" {
		_ = os.RemoveAll(profile)
	}
}

func (s *LoginSession) releaseBrowser() {
	s.stop()
	s.cleanupProfile()
	s.mu.Lock()
	accountID := s.existingAccountID
	owner := s.leaseOwner
	s.leaseOwner = ""
	s.mu.Unlock()
	if accountID != "" && owner != "" {
		_ = AppStore.EndAccountMaintenance(accountID, owner, "")
	}
}

func (s *LoginSession) Restart() error {
	s.mu.Lock()
	if s.Mode == "maintenance" {
		s.mu.Unlock()
		return fmt.Errorf("maintenance sessions cannot be refreshed; stop and start maintenance again")
	}
	if s.Status == "captured" {
		s.mu.Unlock()
		return fmt.Errorf("captured login sessions cannot be refreshed; start a new account login instead")
	}
	s.mu.Unlock()

	s.releaseBrowser()

	s.mu.Lock()
	s.Status = "refreshing"
	s.Message = "正在刷新扫码登录会话，并创建新的 Chromium profile。"
	s.CookieCount = 0
	s.AccountID = ""
	s.screenshot = nil
	s.UpdatedAt = nowISO()
	s.mu.Unlock()

	s.launch()
	return nil
}

func (s *LoginSession) launch() {
	s.mu.Lock()
	s.runGeneration++
	generation := s.runGeneration
	s.mu.Unlock()
	go s.run(generation)
}

func (s *LoginSession) run(generation uint64) {
	defer func() {
		s.mu.Lock()
		currentGeneration := s.runGeneration
		s.mu.Unlock()
		if currentGeneration == generation {
			s.releaseBrowser()
		}
	}()
	s.mu.Lock()
	currentGeneration := s.runGeneration
	s.mu.Unlock()
	if currentGeneration != generation {
		return
	}
	if err := os.MkdirAll(s.userDataDir, 0700); err != nil {
		s.setStatus("failed", "创建登录 profile 目录失败："+err.Error())
		return
	}
	if err := ensureChromiumRuntimeDirs(); err != nil {
		s.setStatus("failed", "Chromium runtime is unavailable: "+err.Error())
		return
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", Cfg.BrowserHeadless),
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
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...interface{}) {
		LogDebug("[qianwen-login] "+format, args...)
	}))
	cancel := func() {
		ctxCancel()
		allocCancel()
	}
	s.mu.Lock()
	if s.runGeneration != generation {
		s.mu.Unlock()
		cancel()
		return
	}
	s.ctx = ctx
	s.cancel = cancel
	s.mu.Unlock()

	s.setStatus("opening", "正在打开 qianwen.com。如果页面出现二维码，请按页面提示使用千问 / 淘宝 / 支付宝完成登录。")
	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate("https://www.qianwen.com/"),
		chromedp.Sleep(4*time.Second),
	); err != nil {
		if ctx.Err() != nil {
			return
		}
		s.setStatus("failed", "打开 qianwen.com 失败："+err.Error())
		return
	}
	_ = clickVisibleLogin(ctx)
	if ctx.Err() != nil {
		return
	}
	_ = clickLikelyTopRightLogin(ctx)
	if ctx.Err() != nil {
		return
	}
	_ = s.RefreshScreenshot()
	if ctx.Err() != nil {
		return
	}
	s.setStatus("waiting_scan", "请在截图中扫码登录；页面进入已登录状态后，点击“确认扫码”。")

	s.mu.Lock()
	mode := s.Mode
	leaseOwner := s.leaseOwner
	maintenanceAccountID := s.existingAccountID
	s.mu.Unlock()

	ticker := time.NewTicker(6 * time.Second)
	expireAfter := 10 * time.Minute
	if mode == "maintenance" {
		expireAfter = time.Hour
	}
	expire := time.NewTimer(expireAfter)
	var heartbeat <-chan time.Time
	var heartbeatTicker *time.Ticker
	if leaseOwner != "" {
		heartbeatTicker = time.NewTicker(5 * time.Minute)
		heartbeat = heartbeatTicker.C
	}
	defer ticker.Stop()
	defer expire.Stop()
	if heartbeatTicker != nil {
		defer heartbeatTicker.Stop()
	}

	for {
		select {
		case <-ticker.C:
			_ = s.RefreshScreenshot()
			count, cookies := s.cookieSnapshot()
			likelyLogin := hasLikelyLoginCookie(cookies)
			s.mu.Lock()
			s.CookieCount = count
			alreadyCaptured := s.AccountID != ""
			if s.Status != "captured" && likelyLogin {
				s.Status = "login_detected"
				if mode == "maintenance" {
					s.Message = "已检测到登录 Cookie，请确认捕获账号材料。"
				} else {
					s.Message = "已检测到登录 Cookie，正在自动捕获账号材料。"
				}
			}
			if s.Status != "captured" && !alreadyCaptured {
				s.UpdatedAt = nowISO()
			}
			s.mu.Unlock()
			if count > 0 && !alreadyCaptured && likelyLogin && mode != "maintenance" {
				if _, err := s.CaptureAccount(); err != nil {
					s.setStatus("capture_failed", "已检测到 Cookie，但捕获账号失败："+err.Error())
				}
			}
		case <-heartbeat:
			if _, err := AppStore.HeartbeatAccountMaintenance(maintenanceAccountID, leaseOwner, defaultMaintenanceLease); err != nil {
				s.setStatus("failed", "Maintenance lease heartbeat failed: "+err.Error())
				return
			}
		case <-expire.C:
			s.setStatus("expired", "登录会话已过期，请重新生成扫码会话。")
			s.releaseBrowser()
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

func clickLikelyTopRightLogin(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.MouseClickXY(1242, 30),
		chromedp.Sleep(2*time.Second),
	)
}

func (s *LoginSession) TriggerLogin() error {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return fmt.Errorf("login browser is not ready")
	}
	if err := clickVisibleLogin(ctx); err != nil {
		LogWarn("Text login click failed: %v", err)
	}
	if err := clickLikelyTopRightLogin(ctx); err != nil {
		return err
	}
	if err := s.RefreshScreenshot(); err != nil {
		return err
	}
	s.setStatus("waiting_scan", "已点击可能的登录入口；如果看到二维码，请扫码后再确认捕获登录态。")
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
	s.captureMu.Lock()
	defer s.captureMu.Unlock()
	s.mu.Lock()
	ctx := s.ctx
	existingAccountID := s.existingAccountID
	targetAccountID := s.targetAccountID
	leaseOwner := s.leaseOwner
	s.mu.Unlock()
	if existingAccountID != "" && leaseOwner != "" {
		if _, err := AppStore.HeartbeatAccountMaintenance(existingAccountID, leaseOwner, defaultMaintenanceLease); err != nil {
			return nil, err
		}
	}

	cookies, source, err := s.captureLoginCookies()
	if err != nil {
		return nil, err
	}
	cookieJSON, cookieString, err := serializeCapturedCookies(cookies)
	if err != nil {
		return nil, err
	}
	var localStorageJSON string
	var userAgent string
	if ctx != nil {
		localStorageJSON = captureQwenBrowserStorage(ctx)
		_ = chromedp.Run(ctx, chromedp.Evaluate(`navigator.userAgent`, &userAgent))
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = generateRandomUserAgent()
	}

	account := &AccountRecord{
		ID:               targetAccountID,
		Name:             s.Name,
		Type:             "qianwen_qr",
		Status:           "unknown",
		Enabled:          true,
		CookieJSON:       cookieJSON,
		CookieString:     cookieString,
		LocalStorageJSON: localStorageJSON,
		UserAgent:        userAgent,
		CapabilitiesJSON: `{"chat":true,"image":true,"video":true}`,
		LastError:        "已从 " + source + " 捕获扫码登录 Cookie。标记为可用前仍需要执行真实模型测活。",
	}
	if existingAccountID != "" {
		if _, err := AppStore.HeartbeatAccountMaintenance(existingAccountID, leaseOwner, defaultMaintenanceLease); err != nil {
			return nil, err
		}
		if err := AppStore.UpdateAccountSessionSnapshot(existingAccountID, cookieJSON, cookieString, localStorageJSON, userAgent); err != nil {
			return nil, err
		}
		account, err = AppStore.GetAccount(existingAccountID)
		if err != nil {
			return nil, err
		}
	} else if err := AppStore.CreateAccount(account); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.Status = "captured"
	s.Message = "已从 " + source + " 捕获浏览器 Cookie 并写入账号池。参与调度前请先执行真实模型测活。"
	s.AccountID = account.ID
	s.profilePersistent = true
	s.CookieCount = len(cookies)
	s.UpdatedAt = nowISO()
	s.mu.Unlock()
	go s.releaseBrowser()
	return account, nil
}

func captureQwenBrowserStorage(ctx context.Context) string {
	type storageCapture struct {
		URL            string                 `json:"url"`
		Origin         string                 `json:"origin"`
		LocalStorage   map[string]interface{} `json:"localStorage"`
		SessionStorage map[string]interface{} `json:"sessionStorage"`
	}
	captures := []storageCapture{}
	collect := func() {
		var raw string
		if err := chromedp.Run(ctx, chromedp.Evaluate(`JSON.stringify({
			url: location.href,
			origin: location.origin,
			localStorage: Object.fromEntries(Object.entries(localStorage)),
			sessionStorage: Object.fromEntries(Object.entries(sessionStorage))
		})`, &raw)); err != nil || strings.TrimSpace(raw) == "" {
			return
		}
		var capture storageCapture
		if json.Unmarshal([]byte(raw), &capture) == nil {
			captures = append(captures, capture)
		}
	}
	collect()
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://chat.qwen.ai/"),
		chromedp.Sleep(4*time.Second),
	); err == nil {
		collect()
	}
	if len(captures) == 0 {
		return ""
	}
	body, _ := json.Marshal(map[string]interface{}{"captures": captures})
	return string(body)
}

func (s *LoginSession) cookieSnapshot() (int, []capturedCookie) {
	cookies, err := s.readCDPCookies()
	if err == nil && len(cookies) > 0 && hasLikelyLoginCookie(cookies) {
		return len(cookies), cookies
	}
	profileCookies, profileErr := s.readProfileCookies()
	if profileErr == nil && len(profileCookies) > 0 {
		return len(profileCookies), profileCookies
	}
	if err == nil {
		return len(cookies), cookies
	}
	return 0, nil
}

func (s *LoginSession) captureLoginCookies() ([]capturedCookie, string, error) {
	failures := make([]string, 0, 2)
	if cookies, err := s.readCDPCookies(); err == nil {
		if len(cookies) == 0 {
			failures = append(failures, "cdp: 没有读取到 Cookie")
		} else if !hasLikelyLoginCookie(cookies) {
			failures = append(failures, "cdp: 已读取到 Cookie，但不像已登录态："+strings.Join(cookieNames(cookies), ","))
		} else {
			return cookies, "cdp", nil
		}
	} else {
		failures = append(failures, "cdp: "+err.Error())
	}

	if cookies, err := s.readProfileCookies(); err == nil {
		if len(cookies) == 0 {
			failures = append(failures, "profile: 没有读取到 Cookie")
		} else if !hasLikelyLoginCookie(cookies) {
			failures = append(failures, "profile: 已读取到 Cookie，但不像已登录态："+strings.Join(cookieNames(cookies), ","))
		} else {
			return cookies, "chromium profile", nil
		}
	} else {
		failures = append(failures, "profile: "+err.Error())
	}

	return nil, "", fmt.Errorf("暂未检测到 qianwen.com 登录 Cookie（%s）", strings.Join(failures, "；"))
}

func (s *LoginSession) readCDPCookies() ([]capturedCookie, error) {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return nil, fmt.Errorf("login browser is not ready")
	}
	raw, err := network.GetCookies().WithUrls([]string{
		"https://www.qianwen.com/",
		"https://qianwen.com/",
		"https://api.qianwen.com/",
		"https://chat.qwen.ai/",
		"https://passport.aliyun.com/",
		"https://login.taobao.com/",
	}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("read cdp cookies: %w", err)
	}
	cookies := make([]capturedCookie, 0, len(raw))
	for _, cookie := range raw {
		if cookie == nil || cookie.Name == "" || cookie.Value == "" {
			continue
		}
		cookies = append(cookies, capturedCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  float64(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			SameSite: string(cookie.SameSite),
		})
	}
	return cookies, nil
}

func (s *LoginSession) readProfileCookies() ([]capturedCookie, error) {
	cookieDB := filepath.Join(s.userDataDir, "Default", "Cookies")
	if _, err := os.Stat(cookieDB); err != nil {
		return nil, fmt.Errorf("read chromium profile cookies: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+cookieDB+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open chromium profile cookie db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.Query(`SELECT host_key, name, value, encrypted_value, path, expires_utc, is_httponly, is_secure, samesite FROM cookies`)
	if err != nil {
		return nil, fmt.Errorf("query chromium profile cookie db: %w", err)
	}
	defer rows.Close()

	cookies := []capturedCookie{}
	for rows.Next() {
		var host, name, value, path string
		var encrypted []byte
		var expires int64
		var httpOnly, secure, sameSite int
		if err := rows.Scan(&host, &name, &value, &encrypted, &path, &expires, &httpOnly, &secure, &sameSite); err != nil {
			return nil, fmt.Errorf("scan chromium profile cookie: %w", err)
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		if value == "" && len(encrypted) > 0 {
			decrypted, err := decryptChromiumCookie(host, encrypted)
			if err != nil {
				LogWarn("Failed to decrypt chromium profile cookie %s/%s: %v", host, name, err)
				continue
			}
			value = decrypted
		}
		if value == "" {
			continue
		}
		cookies = append(cookies, capturedCookie{
			Name:     name,
			Value:    value,
			Domain:   host,
			Path:     defaultString(path, "/"),
			Expires:  chromeCookieExpiresToUnix(expires),
			HTTPOnly: httpOnly == 1,
			Secure:   secure == 1,
			SameSite: chromeCookieSameSite(sameSite),
		})
	}
	return cookies, rows.Err()
}

func cookieNames(cookies []capturedCookie) []string {
	names := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		names = append(names, cookie.Domain+"/"+cookie.Name)
	}
	return names
}

func serializeCapturedCookies(cookies []capturedCookie) (string, string, error) {
	items := make([]capturedCookie, 0, len(cookies))
	pairs := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Name == "" || cookie.Value == "" {
			continue
		}
		items = append(items, cookie)
		pairs = append(pairs, cookie.Name+"="+cookie.Value)
	}
	body, err := json.Marshal(items)
	if err != nil {
		return "", "", err
	}
	return string(body), strings.Join(pairs, "; "), nil
}

func decryptChromiumCookie(host string, encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	payload := encrypted
	if len(payload) >= 3 {
		prefix := string(payload[:3])
		if prefix == "v10" || prefix == "v11" {
			payload = payload[3:]
		}
	}
	if len(payload) == 0 || len(payload)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid chromium encrypted cookie length")
	}
	block, err := aes.NewCipher(pbkdf2SHA1([]byte("peanuts"), []byte("saltysalt"), 1, 16))
	if err != nil {
		return "", err
	}
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}
	plain := make([]byte, len(payload))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, payload)
	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", err
	}
	if len(plain) >= sha256.Size {
		digest := sha256.Sum256([]byte(host))
		if hmac.Equal(plain[:sha256.Size], digest[:]) {
			plain = plain[sha256.Size:]
		}
	}
	return string(plain), nil
}

func pbkdf2SHA1(password, salt []byte, iterations, keyLen int) []byte {
	const hashLen = 20
	blockCount := (keyLen + hashLen - 1) / hashLen
	out := make([]byte, 0, blockCount*hashLen)
	for block := 1; block <= blockCount; block++ {
		mac := hmac.New(sha1.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha1.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, fmt.Errorf("invalid pkcs7 payload length")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, fmt.Errorf("invalid pkcs7 padding")
	}
	for _, b := range value[len(value)-padding:] {
		if int(b) != padding {
			return nil, fmt.Errorf("invalid pkcs7 padding bytes")
		}
	}
	return value[:len(value)-padding], nil
}

func chromeCookieExpiresToUnix(expires int64) float64 {
	if expires <= 0 {
		return 0
	}
	unixSeconds := float64(expires)/1000000 - 11644473600
	if unixSeconds <= 0 {
		return 0
	}
	return unixSeconds
}

func chromeCookieSameSite(value int) string {
	switch value {
	case 0:
		return "None"
	case 1:
		return "Lax"
	case 2:
		return "Strict"
	default:
		return ""
	}
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
		if err := session.Restart(); err != nil {
			writeAPIError(w, http.StatusFailedDependency, "login_session_refresh_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": session.view()})
	case "click-login":
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
			return
		}
		if err := session.TriggerLogin(); err != nil {
			writeAPIError(w, http.StatusFailedDependency, "login_click_failed", err.Error())
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

func HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	accounts, err := AppStore.ListAccounts()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "account_list_failed", err.Error())
		return
	}
	if accounts == nil {
		accounts = []AccountRecord{}
	}
	masked := make([]AccountRecord, 0, len(accounts))
	validCount := 0
	for _, account := range accounts {
		if account.Status == "valid" {
			validCount++
		}
		masked = append(masked, maskAccount(account))
	}
	var latest interface{} = nil
	if session, ok := QianwenLoginSessions.LatestActive(); ok {
		latest = session.view()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                  true,
		"provider":            "QIANWEN-WEB-01",
		"logged_in":           len(accounts) > 0,
		"valid_account_count": validCount,
		"account_count":       len(accounts),
		"accounts":            masked,
		"login_session":       latest,
	})
}

func HandleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or POST.")
		return
	}
	session, err := QianwenLoginSessions.LatestOrStart()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "login_session_start_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": session.view(),
		"qr":   "/auth/qr?key=" + Cfg.AdminKey,
	})
}

func HandleAuthQR(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	session, err := QianwenLoginSessions.LatestOrStart()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "login_session_start_failed", err.Error())
		return
	}
	for i := 0; i < 12; i++ {
		image := session.Screenshot()
		if len(image) > 0 {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(image)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	writeAPIError(w, http.StatusAccepted, "screenshot_not_ready", "Login screenshot is not ready yet. Refresh this URL in a few seconds.")
}

func HandleAuthCapture(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST.")
		return
	}
	session, ok := QianwenLoginSessions.LatestActive()
	if !ok {
		writeAPIError(w, http.StatusNotFound, "login_session_not_found", "No active qianwen login session.")
		return
	}
	account, err := session.CaptureAccount()
	if err != nil {
		writeAPIError(w, http.StatusFailedDependency, "login_capture_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":    maskAccount(*account),
		"session": session.view(),
	})
}
