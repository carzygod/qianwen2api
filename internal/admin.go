package internal

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func HandleAdminPage(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

func HandleAdminAPI(w http.ResponseWriter, r *http.Request) {
	if !requireAdminAuth(w, r) {
		return
	}
	if AppStore == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "store_not_ready", "SQLite store is not initialized.")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api")
	switch {
	case path == "/admin/summary" && r.Method == http.MethodGet:
		handleAdminSummary(w, r)
	case path == "/login-sessions" || strings.HasPrefix(path, "/login-sessions/"):
		handleLoginSessions(w, r, path)
	case path == "/accounts" && r.Method == http.MethodGet:
		handleListAccounts(w, r)
	case path == "/accounts" && r.Method == http.MethodPost:
		handleCreateAccount(w, r)
	case strings.HasPrefix(path, "/accounts/"):
		handleAccountAction(w, r, strings.TrimPrefix(path, "/accounts/"))
	case path == "/tasks" && r.Method == http.MethodGet:
		handleListTasks(w, r)
	case strings.HasPrefix(path, "/tasks/") && r.Method == http.MethodGet:
		handleGetTask(w, r, strings.TrimPrefix(path, "/tasks/"))
	case path == "/models" && r.Method == http.MethodGet:
		models, err := AppStore.ListModels()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "model_list_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": models})
	default:
		writeAPIError(w, http.StatusNotFound, "admin_route_not_found", "Admin API route not found.")
	}
}

func handleAdminSummary(w http.ResponseWriter, r *http.Request) {
	accounts, _ := AppStore.ListAccounts()
	tasks, _ := AppStore.ListTasks(200)
	accountStatus := map[string]int{}
	taskStatus := map[string]int{}
	for _, a := range accounts {
		accountStatus[a.Status]++
	}
	for _, t := range tasks {
		taskStatus[t.Status]++
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service": map[string]interface{}{
			"name":            "QIANWEN-WEB-01",
			"host":            Cfg.Host,
			"port":            Cfg.Port,
			"data_dir":        Cfg.DataDir,
			"database_path":   Cfg.DatabasePath,
			"public_base_url": Cfg.PublicBaseURL,
			"guest_pool_size": Cfg.PoolSize,
		},
		"accounts": map[string]interface{}{
			"total":  len(accounts),
			"status": accountStatus,
		},
		"tasks": map[string]interface{}{
			"total":  len(tasks),
			"status": taskStatus,
		},
	})
}

func handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := AppStore.ListAccounts()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "account_list_failed", err.Error())
		return
	}
	if accounts == nil {
		accounts = []AccountRecord{}
	}
	for i := range accounts {
		accounts[i] = maskAccount(accounts[i])
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": accounts})
}

func handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req AccountRecord
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Type == "" {
		req.Type = "login_cookie"
	}
	if !req.Enabled {
		req.Enabled = true
	}
	if err := AppStore.CreateAccount(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "account_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"data": maskAccount(req)})
}

func handleAccountAction(w http.ResponseWriter, r *http.Request, suffix string) {
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, "account_route_not_found", "Account route not found.")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			account, err := AppStore.GetAccount(id)
			if err != nil {
				writeAccountLookupError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"data": maskAccount(*account)})
		case http.MethodDelete:
			if err := AppStore.DeleteAccount(id); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "account_delete_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		}
		return
	}
	if len(parts) == 2 && parts[1] == "test" && r.Method == http.MethodPost {
		var body struct {
			Capability string `json:"capability"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		result, err := TestAccount(id, body.Capability)
		if err != nil {
			writeAccountLookupError(w, err)
			return
		}
		status := http.StatusOK
		if !result.OK {
			status = http.StatusFailedDependency
		}
		writeJSON(w, status, result)
		return
	}
	if len(parts) == 3 && parts[1] == "quota" && parts[2] == "sync" && r.Method == http.MethodPost {
		msg := "Quota sync requires qianwen.com logged-in quota endpoint capture. No quota was changed."
		_ = AppStore.UpdateAccountStatus(id, "unknown", msg, false)
		writeJSON(w, http.StatusFailedDependency, map[string]interface{}{
			"ok":      false,
			"code":    "quota_protocol_required",
			"message": msg,
		})
		return
	}
	writeAPIError(w, http.StatusNotFound, "account_route_not_found", "Account route not found.")
}

func handleListTasks(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	tasks, err := AppStore.ListTasks(limit)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "task_list_failed", err.Error())
		return
	}
	if tasks == nil {
		tasks = []TaskRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": tasks})
}

func handleGetTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := AppStore.GetTask(strings.Trim(id, "/"))
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

func writeAccountLookupError(w http.ResponseWriter, err error) {
	if err == sql.ErrNoRows {
		writeAPIError(w, http.StatusNotFound, "account_not_found", "Account not found.")
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "account_lookup_failed", err.Error())
}

func maskAccount(a AccountRecord) AccountRecord {
	a.CookieJSON = maskSecret(a.CookieJSON)
	a.CookieString = maskSecret(a.CookieString)
	a.LocalStorageJSON = maskSecret(a.LocalStorageJSON)
	a.XsrfToken = maskSecret(a.XsrfToken)
	return a
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 12 {
		return "***"
	}
	return value[:6] + "..." + value[len(value)-4:]
}

const adminHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>QIANWEN-WEB-01 Admin</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #101418;
      --panel: #171d23;
      --panel-2: #1d252d;
      --text: #e8eef4;
      --muted: #96a2ae;
      --line: #2b3640;
      --accent: #36d399;
      --danger: #ff6b6b;
      --warn: #f8c14a;
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--bg); color: var(--text); font: 14px/1.5 Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    header { display: flex; align-items: center; justify-content: space-between; padding: 22px 28px; border-bottom: 1px solid var(--line); background: #12181e; position: sticky; top: 0; z-index: 2; }
    h1 { font-size: 18px; margin: 0; letter-spacing: 0; }
    main { padding: 24px 28px 42px; max-width: 1320px; margin: 0 auto; }
    .grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 18px; }
    .card, section { background: var(--panel); border: 1px solid var(--line); border-radius: 18px; box-shadow: 0 18px 48px rgba(0,0,0,.18); }
    .card { padding: 16px; min-height: 98px; }
    .label { color: var(--muted); font-size: 12px; }
    .metric { font-size: 24px; font-weight: 700; margin-top: 8px; }
    section { margin-top: 18px; overflow: hidden; }
    .section-head { display: flex; justify-content: space-between; align-items: center; padding: 16px 18px; border-bottom: 1px solid var(--line); }
    h2 { font-size: 15px; margin: 0; }
    button, input, textarea, select { border: 1px solid var(--line); border-radius: 12px; background: var(--panel-2); color: var(--text); padding: 9px 11px; }
    button { cursor: pointer; transition: transform .16s ease, border-color .16s ease, background .16s ease; }
    button:hover { transform: translateY(-1px); border-color: var(--accent); }
    button.primary { background: var(--accent); color: #04110b; border-color: var(--accent); font-weight: 700; }
    button.danger { border-color: rgba(255,107,107,.4); color: var(--danger); }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 12px 14px; border-bottom: 1px solid var(--line); vertical-align: top; }
    th { color: var(--muted); font-size: 12px; font-weight: 600; background: #141a20; }
    code { color: #b7f7d4; }
    .pill { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); border-radius: 999px; padding: 3px 8px; color: var(--muted); font-size: 12px; }
    .ok { color: var(--accent); }
    .bad { color: var(--danger); }
    .warn { color: var(--warn); }
    dialog { width: min(760px, calc(100vw - 32px)); border: 1px solid var(--line); border-radius: 22px; background: var(--panel); color: var(--text); box-shadow: 0 30px 100px rgba(0,0,0,.55); }
    dialog::backdrop { background: rgba(0,0,0,.62); backdrop-filter: blur(8px); }
    .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    textarea { width: 100%; min-height: 130px; resize: vertical; }
    .full { grid-column: 1 / -1; }
    .hint { color: var(--muted); font-size: 12px; margin: 8px 0 0; }
    @media (max-width: 900px) { .grid, .form-grid { grid-template-columns: 1fr; } header { align-items: flex-start; gap: 12px; flex-direction: column; } }
  </style>
</head>
<body>
  <header>
    <h1>QIANWEN-WEB-01 Admin</h1>
    <div class="pill">SQLite · Web reverse proxy · no Redis</div>
  </header>
  <main>
    <div class="grid">
      <div class="card"><div class="label">Accounts</div><div class="metric" id="accountTotal">-</div></div>
      <div class="card"><div class="label">Tasks</div><div class="metric" id="taskTotal">-</div></div>
      <div class="card"><div class="label">Guest Pool</div><div class="metric" id="guestPool">-</div></div>
      <div class="card"><div class="label">Port</div><div class="metric" id="port">-</div></div>
    </div>
    <section>
      <div class="section-head">
        <h2>账号池</h2>
        <div style="display:flex; gap:10px;">
          <button onclick="startLoginSession()">扫码登录</button>
          <button class="primary" onclick="openAccountDialog()">新增账号</button>
        </div>
      </div>
      <table>
        <thead><tr><th>名称</th><th>类型</th><th>状态</th><th>能力</th><th>最近错误</th><th>操作</th></tr></thead>
        <tbody id="accounts"></tbody>
      </table>
    </section>
    <section>
      <div class="section-head">
        <h2>扫码登录会话</h2>
        <button onclick="loadLoginSessions()">刷新</button>
      </div>
      <table>
        <thead><tr><th>ID</th><th>名称</th><th>状态</th><th>Cookie</th><th>消息</th><th>操作</th></tr></thead>
        <tbody id="loginSessions"></tbody>
      </table>
    </section>
    <section>
      <div class="section-head">
        <h2>任务</h2>
        <button onclick="loadAll()">刷新</button>
      </div>
      <table>
        <thead><tr><th>ID</th><th>类型</th><th>模型</th><th>状态</th><th>错误</th><th>创建时间</th></tr></thead>
        <tbody id="tasks"></tbody>
      </table>
    </section>
  </main>
  <dialog id="accountDialog">
    <form method="dialog" onsubmit="event.preventDefault(); createAccount();">
      <h2>新增 qianwen.com 账号材料</h2>
      <p class="hint">当前登录态真实测试需要完成 qianwen.com 协议抓包；保存 Cookie 后不会自动标记 valid，必须真实模型调用通过才会参与调度。</p>
      <div class="form-grid">
        <label>名称<br><input id="name" required placeholder="Qianwen account" /></label>
        <label>类型<br><select id="type"><option value="login_cookie">login_cookie</option><option value="guest">guest</option></select></label>
        <label class="full">Cookie 字符串<br><textarea id="cookieString" placeholder="从请求头复制 Cookie: a=b; c=d"></textarea></label>
        <label class="full">Cookie JSON<br><textarea id="cookieJSON" placeholder='[{"name":"...","value":"...","domain":".qianwen.com"}]'></textarea></label>
        <label class="full">能力 JSON<br><input id="capabilities" value='{"chat":true,"image":true,"video":true}' /></label>
      </div>
      <div style="display:flex; justify-content:flex-end; gap:10px; margin-top:16px;">
        <button type="button" onclick="accountDialog.close()">取消</button>
        <button class="primary" type="submit">保存账号</button>
      </div>
    </form>
  </dialog>
  <dialog id="loginDialog">
    <h2>qianwen.com 扫码登录</h2>
    <p class="hint">使用手机扫描下方截图中的 qianwen.com 登录二维码。扫码成功、页面变成已登录后，点击“保存当前登录态”。保存后账号会进入 SQLite 账号池，但仍需真实模型测试通过才会参与调度。</p>
    <div style="background:#0b1015; border:1px solid var(--line); border-radius:18px; min-height:360px; display:flex; align-items:center; justify-content:center; overflow:hidden;">
      <img id="loginShot" alt="qianwen login screenshot" style="max-width:100%; display:block;" />
    </div>
    <p class="hint" id="loginStatusText"></p>
    <div style="display:flex; justify-content:flex-end; gap:10px; margin-top:16px;">
      <button type="button" onclick="refreshLoginScreenshot()">刷新截图</button>
      <button type="button" onclick="captureLoginSession()">保存当前登录态</button>
      <button class="primary" type="button" onclick="loginDialog.close()">关闭</button>
    </div>
  </dialog>
  <script>
    const adminKey = new URLSearchParams(location.search).get('key') || '';
    let currentLoginSessionId = '';
    let loginPollTimer = 0;
    const headers = () => ({ 'Content-Type': 'application/json', 'X-Admin-Key': adminKey });
    async function api(path, options = {}) {
      const res = await fetch('/api' + path, { ...options, headers: { ...headers(), ...(options.headers || {}) } });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.message || data.error?.message || res.statusText);
      return data;
    }
    function openAccountDialog() { accountDialog.showModal(); }
    async function loadAll() {
      const summary = await api('/admin/summary');
      accountTotal.textContent = summary.accounts.total;
      taskTotal.textContent = summary.tasks.total;
      guestPool.textContent = summary.service.guest_pool_size;
      port.textContent = summary.service.port;
      const accountData = await api('/accounts');
      accounts.innerHTML = accountData.data.map(a => '<tr><td>' + esc(a.name) + '<br><code>' + esc(a.id) + '</code></td><td>' + esc(a.type) + '</td><td>' + status(a.status) + '</td><td><code>' + esc(a.capabilities_json || '') + '</code></td><td>' + esc(a.last_error || '') + '</td><td><button onclick="testAccount(\'' + a.id + '\')">测试</button> <button class="danger" onclick="deleteAccount(\'' + a.id + '\')">删除</button></td></tr>').join('');
      await loadLoginSessions();
      const taskData = await api('/tasks?limit=50');
      tasks.innerHTML = taskData.data.map(t => '<tr><td><code>' + esc(t.id) + '</code></td><td>' + esc(t.type) + '</td><td>' + esc(t.model || '') + '</td><td>' + status(t.status) + '</td><td>' + esc(t.error_message || '') + '</td><td>' + esc(t.created_at) + '</td></tr>').join('');
    }
    async function createAccount() {
      await api('/accounts', { method: 'POST', body: JSON.stringify({
        name: name.value, type: type.value, cookie_string: cookieString.value, cookie_json: cookieJSON.value,
        capabilities_json: capabilities.value, enabled: true
      }) });
      accountDialog.close();
      await loadAll();
    }
    async function testAccount(id) {
      try {
        const result = await api('/accounts/' + id + '/test', { method: 'POST', body: JSON.stringify({ capability: 'chat' }) });
        alert(result.message || 'ok');
      } catch (err) {
        alert(err.message);
      }
      await loadAll();
    }
    async function deleteAccount(id) {
      if (!confirm('删除该账号？')) return;
      await api('/accounts/' + id, { method: 'DELETE' });
      await loadAll();
    }
    async function loadLoginSessions() {
      const data = await api('/login-sessions');
      loginSessions.innerHTML = data.data.map(s => '<tr><td><code>' + esc(s.id) + '</code></td><td>' + esc(s.name) + '</td><td>' + status(s.status) + '</td><td>' + esc(s.cookie_count || 0) + '</td><td>' + esc(s.message || '') + '</td><td><button onclick="showLoginSession(\'' + s.id + '\')">打开</button> <button onclick="captureSpecificLoginSession(\'' + s.id + '\')">保存</button></td></tr>').join('');
    }
    async function startLoginSession() {
      const data = await api('/login-sessions', { method: 'POST', body: JSON.stringify({ name: 'qianwen-' + new Date().toISOString() }) });
      currentLoginSessionId = data.data.id;
      showLoginDialog(data.data);
      await loadLoginSessions();
    }
    async function showLoginSession(id) {
      const data = await api('/login-sessions/' + id);
      currentLoginSessionId = id;
      showLoginDialog(data.data);
    }
    function showLoginDialog(session) {
      currentLoginSessionId = session.id;
      loginShot.src = '/api/login-sessions/' + session.id + '/screenshot?key=' + encodeURIComponent(adminKey) + '&t=' + Date.now();
      loginStatusText.textContent = session.status + ' · ' + (session.message || '');
      loginDialog.showModal();
      clearInterval(loginPollTimer);
      loginPollTimer = setInterval(async () => {
        if (!currentLoginSessionId || !loginDialog.open) return;
        try {
          const latest = await api('/login-sessions/' + currentLoginSessionId);
          loginStatusText.textContent = latest.data.status + ' · ' + (latest.data.message || '') + ' · cookies=' + (latest.data.cookie_count || 0);
          loginShot.src = '/api/login-sessions/' + currentLoginSessionId + '/screenshot?key=' + encodeURIComponent(adminKey) + '&t=' + Date.now();
          await loadLoginSessions();
        } catch {}
      }, 6000);
    }
    async function refreshLoginScreenshot() {
      if (!currentLoginSessionId) return;
      await api('/login-sessions/' + currentLoginSessionId + '/refresh', { method: 'POST' });
      loginShot.src = '/api/login-sessions/' + currentLoginSessionId + '/screenshot?key=' + encodeURIComponent(adminKey) + '&t=' + Date.now();
      await loadLoginSessions();
    }
    async function captureSpecificLoginSession(id) {
      currentLoginSessionId = id;
      await captureLoginSession();
    }
    async function captureLoginSession() {
      if (!currentLoginSessionId) return;
      try {
        const result = await api('/login-sessions/' + currentLoginSessionId + '/capture', { method: 'POST' });
        alert('已保存账号：' + result.data.id);
      } catch (err) {
        alert(err.message);
      }
      await loadAll();
    }
    function status(value) {
      const cls = value === 'valid' || value === 'succeeded' ? 'ok' : (value === 'invalid' || value === 'failed' ? 'bad' : 'warn');
      return '<span class="' + cls + '">' + esc(value || '') + '</span>';
    }
    function esc(value) {
      return String(value ?? '').replace(/[&<>"']/g, s => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[s]));
    }
    loadAll().catch(err => alert(err.message));
  </script>
</body>
</html>`
