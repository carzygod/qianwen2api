package internal

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>QIANWEN-WEB-01 Account Pool</title>
<script src="https://unpkg.com/vue@3/dist/vue.global.prod.js"></script>
<style>
:root{
  color:#f4f8ff;
  background:#020409;
  color-scheme:dark;
  font-family:Inter,"Segoe UI","Microsoft YaHei",Arial,sans-serif;
  --bg:#020409;
  --panel:#08111b;
  --panel-2:#0d1825;
  --panel-3:#060d16;
  --line:#182638;
  --line-strong:#294057;
  --text:#f4f8ff;
  --muted:#9cafc7;
  --dim:#667891;
  --cyan:#36e7ff;
  --blue:#3e7dff;
  --violet:#9a66ff;
  --green:#3fe59a;
  --red:#ff6675;
  --amber:#ffd166;
  --grad:linear-gradient(135deg,#36e7ff 0%,#3e7dff 48%,#9a66ff 100%);
  --grad-soft:linear-gradient(135deg,rgba(54,231,255,.15),rgba(62,125,255,.1) 46%,rgba(154,102,255,.13));
  --shadow:0 22px 56px rgba(0,0,0,.44);
  --glow:0 0 26px rgba(54,231,255,.16),0 0 42px rgba(154,102,255,.08);
  --radius-sm:14px;
  --radius-md:18px;
  --radius-lg:24px;
  --radius-xl:30px;
  --ease:cubic-bezier(.2,.8,.2,1);
}
*{box-sizing:border-box}
*{scrollbar-color:rgba(54,231,255,.32) rgba(6,12,21,.8);scrollbar-width:thin}
*::-webkit-scrollbar{width:10px;height:10px}
*::-webkit-scrollbar-track{background:rgba(6,12,21,.8)}
*::-webkit-scrollbar-thumb{background:linear-gradient(180deg,rgba(54,231,255,.5),rgba(154,102,255,.45));border:2px solid rgba(6,12,21,.9);border-radius:999px}
body{
  margin:0;
  min-width:320px;
  min-height:100vh;
  background:
    linear-gradient(90deg,rgba(144,189,255,.018) 1px,transparent 1px),
    linear-gradient(rgba(144,189,255,.014) 1px,transparent 1px),
    linear-gradient(135deg,#010208 0%,#040813 48%,#080815 100%);
  background-size:34px 34px,34px 34px,auto;
  color:var(--text);
}
button,input,select,textarea{font:inherit;letter-spacing:0}
button{cursor:pointer}
[v-cloak]{display:none}
.shell{display:grid;grid-template-columns:268px 1fr;min-height:100vh}
.side{
  background:linear-gradient(180deg,rgba(9,17,28,.99),rgba(2,4,9,.99)),var(--bg);
  border-right:1px solid var(--line);
  padding:22px 16px;
  display:flex;
  flex-direction:column;
  gap:22px;
  box-shadow:18px 0 48px rgba(0,0,0,.26);
}
.brand{display:flex;align-items:center;gap:12px;min-height:50px}
.brand-mark{
  width:44px;height:44px;border-radius:var(--radius-md);
  display:grid;place-items:center;
  color:#00131a;font-weight:900;
  background:var(--grad);
  box-shadow:var(--glow);
  border:1px solid rgba(54,231,255,.24);
}
.brand-title{font-size:20px;font-weight:900;letter-spacing:0}
.brand-sub,.muted,.metric-label,.eyebrow{color:var(--muted);font-size:12px}
.nav{display:grid;gap:8px}
.nav button{
  min-height:42px;border:1px solid transparent;color:#c9d6e6;background:transparent;
  display:flex;align-items:center;gap:10px;padding:10px 12px;border-radius:var(--radius-md);
  transition:color 180ms var(--ease),border-color 180ms var(--ease),background 180ms var(--ease),transform 180ms var(--ease),box-shadow 180ms var(--ease);
}
.nav button:hover,.nav button.active{
  color:var(--text);background:rgba(54,231,255,.065);border-color:rgba(54,231,255,.22);
  box-shadow:inset 0 1px 0 rgba(255,255,255,.045);
}
.nav button.active{
  background:linear-gradient(90deg,rgba(54,231,255,.15),rgba(62,125,255,.09) 46%,rgba(154,102,255,.08)),rgba(255,255,255,.018);
  border-color:rgba(54,231,255,.34);
  box-shadow:var(--glow),inset 0 1px 0 rgba(255,255,255,.06);
}
.nav button:hover{transform:translateX(2px)}
.side-foot{margin-top:auto;border:1px solid var(--line);border-radius:var(--radius-lg);background:rgba(8,17,27,.72);padding:14px}
.main{min-width:0;padding:24px 28px 42px}
.topbar{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:22px}
h1,h2,h3,p{margin:0}
h1{font-size:28px;line-height:1.12}
h2{font-size:18px}
h3{font-size:15px}
.subline{margin-top:8px;color:var(--muted);font-size:13px}
.toolbar{display:flex;gap:10px;align-items:center;flex-wrap:wrap}
.tab-panel{animation:tabReveal 360ms var(--ease)}
@keyframes tabReveal{from{opacity:0;transform:translateY(10px) scale(.995)}to{opacity:1;transform:none}}
.grid{display:grid;gap:16px}
.stats{grid-template-columns:repeat(4,minmax(0,1fr));margin-bottom:18px}
.two{grid-template-columns:minmax(360px,1.1fr) minmax(340px,.9fr)}
.card,.table-wrap{
  background:linear-gradient(180deg,rgba(12,24,38,.94),rgba(7,15,25,.94)),var(--panel);
  border:1px solid var(--line);
  border-radius:var(--radius-lg);
  box-shadow:var(--shadow);
  transition:border-color 220ms var(--ease),box-shadow 220ms var(--ease),transform 220ms var(--ease);
}
.card{padding:18px}
.card:hover,.table-wrap:hover{border-color:rgba(54,231,255,.24);box-shadow:var(--shadow),0 0 36px rgba(54,231,255,.06)}
.metric{
  min-height:110px;
  display:flex;
  flex-direction:column;
  justify-content:space-between;
  background:linear-gradient(135deg,rgba(54,231,255,.08),rgba(154,102,255,.06)),rgba(8,17,27,.92);
}
.metric-value{font-size:26px;font-weight:900}
.metric-meta{font-size:12px;color:var(--dim);word-break:break-word}
.account-list{display:grid;gap:12px;max-height:calc(100vh - 255px);overflow:auto;padding-right:3px}
.account-card{
  border:1px solid var(--line);
  background:rgba(255,255,255,.026);
  border-radius:var(--radius-md);
  padding:14px;
  display:grid;
  gap:11px;
  transition:transform 180ms var(--ease),border-color 180ms var(--ease),background 180ms var(--ease),box-shadow 180ms var(--ease);
}
.account-card:hover{transform:translateY(-1px);border-color:rgba(54,231,255,.24)}
.account-card.active{border-color:rgba(54,231,255,.48);background:var(--grad-soft);box-shadow:var(--glow)}
.account-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}
.account-name{font-weight:800}
.account-id{font-size:12px;color:var(--dim);font-family:"SFMono-Regular",Consolas,monospace;margin-top:3px}
.badges{display:flex;gap:6px;flex-wrap:wrap}
.badge{
  display:inline-flex;align-items:center;gap:6px;
  min-height:24px;padding:3px 9px;border-radius:999px;
  font-size:12px;border:1px solid transparent;background:rgba(142,160,182,.12);color:var(--muted);
}
.badge.valid,.badge.succeeded,.badge.captured,.badge.login_detected{background:rgba(63,229,154,.14);color:var(--green)}
.badge.hot,.badge.qianwen_qr{background:rgba(54,231,255,.12);color:var(--cyan)}
.badge.invalid,.badge.failed,.badge.capture_failed,.badge.expired,.badge.error{background:rgba(255,102,117,.14);color:var(--red)}
.badge.unknown,.badge.starting,.badge.opening,.badge.waiting_scan,.badge.disabled{background:rgba(255,209,102,.13);color:var(--amber)}
.detail-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:16px}
.detail-title{display:flex;align-items:center;gap:12px}
.avatar{
  width:46px;height:46px;border-radius:var(--radius-md);
  display:grid;place-items:center;background:var(--grad);color:#00131a;font-weight:900;
}
.actions{display:flex;gap:8px;flex-wrap:wrap}
.btn{
  border:1px solid var(--line-strong);background:rgba(11,19,30,.94);color:var(--text);
  border-radius:var(--radius-sm);min-height:38px;padding:8px 13px;
  display:inline-flex;align-items:center;justify-content:center;gap:8px;
  transition:transform 180ms var(--ease),border-color 180ms var(--ease),color 180ms var(--ease),background 180ms var(--ease),box-shadow 180ms var(--ease);
}
.btn:hover{transform:translateY(-1px);border-color:var(--cyan);color:var(--cyan);box-shadow:var(--glow)}
.btn.primary{border:0;background:var(--grad);color:#00131a;font-weight:900;box-shadow:0 14px 34px rgba(54,231,255,.18)}
.btn.danger{color:var(--red);border-color:rgba(255,102,117,.35)}
.btn.ghost{background:transparent}
.btn:disabled{opacity:.48;cursor:not-allowed;transform:none;box-shadow:none}
.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}
label{display:grid;gap:7px;color:var(--muted);font-size:12px}
input,select,textarea{
  width:100%;
  border:1px solid var(--line-strong);
  border-radius:var(--radius-sm);
  background:rgba(5,11,18,.92);
  color:var(--text);
  padding:10px 12px;
  outline:none;
  transition:border-color 160ms var(--ease),box-shadow 160ms var(--ease),background 160ms var(--ease);
}
textarea{resize:vertical;min-height:96px}
input:focus,select:focus,textarea:focus{border-color:var(--cyan);box-shadow:0 0 0 3px rgba(54,231,255,.12);background:#07111d}
.kv{display:grid;grid-template-columns:130px 1fr;gap:10px;padding:9px 0;border-bottom:1px solid rgba(41,64,87,.42);font-size:13px}
.kv:last-child{border-bottom:0}
.kv span:first-child{color:var(--muted)}
.mono{font-family:"SFMono-Regular",Consolas,monospace;font-size:12px;word-break:break-all}
.table-wrap{overflow:hidden}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{padding:12px 14px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
th{color:var(--muted);font-size:12px;background:#070f19}
tr:hover td{background:rgba(54,231,255,.025)}
.shot{
  width:100%;
  min-height:360px;
  border-radius:var(--radius-md);
  border:1px solid var(--line-strong);
  background:#03070c;
  object-fit:contain;
}
.overlay{
  position:fixed;inset:0;z-index:60;
  background:rgba(2,4,9,.72);
  backdrop-filter:blur(18px);
  display:grid;place-items:center;
  padding:24px;
  animation:fadeIn 180ms var(--ease);
}
.modal{
  width:min(640px,100%);
  background:linear-gradient(180deg,rgba(12,24,38,.98),rgba(5,11,18,.98));
  border:1px solid rgba(54,231,255,.24);
  border-radius:var(--radius-xl);
  box-shadow:var(--shadow),var(--glow);
  padding:22px;
  animation:modalUp 240ms var(--ease);
}
.modal-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:16px}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes modalUp{from{opacity:0;transform:translateY(12px) scale(.98)}to{opacity:1;transform:none}}
.guide{display:grid;gap:9px;margin:14px 0;padding:14px;border:1px solid rgba(41,64,87,.72);border-radius:var(--radius-md);background:rgba(54,231,255,.055);color:var(--muted);font-size:13px}
.empty{padding:38px;text-align:center;color:var(--muted)}
.split{display:flex;align-items:center;justify-content:space-between;gap:12px}
.hint{font-size:12px;color:var(--dim);line-height:1.6}
pre.out{max-height:420px;overflow:auto;white-space:pre-wrap;word-break:break-word;background:#050b12;border:1px solid var(--line);border-radius:var(--radius-md);padding:14px;color:#adf5ff}
.loading-ribbon{position:fixed;inset:0 0 auto;z-index:80;height:3px;overflow:hidden;background:rgba(54,231,255,.08)}
.loading-ribbon span{display:block;width:42%;height:100%;background:linear-gradient(90deg,transparent,var(--cyan),var(--blue),var(--violet),transparent);box-shadow:0 0 18px rgba(54,231,255,.62);animation:loadingSweep 1.1s var(--ease) infinite}
@keyframes loadingSweep{from{transform:translateX(-100%)}to{transform:translateX(240%)}}
.toast{
  position:fixed;left:50%;bottom:24px;z-index:90;
  transform:translateX(-50%) translateY(16px);
  opacity:0;pointer-events:none;
  max-width:min(720px,calc(100vw - 32px));
  padding:12px 14px;border-radius:var(--radius-sm);
  border:1px solid rgba(54,231,255,.28);
  background:rgba(7,15,25,.96);
  box-shadow:var(--shadow),var(--glow);
  color:var(--text);
  transition:opacity 180ms var(--ease),transform 180ms var(--ease);
}
.toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
@media (max-width:1100px){
  .shell{grid-template-columns:1fr}
  .side{position:sticky;top:0;z-index:20;display:block;padding:14px}
  .brand{margin-bottom:12px}
  .nav{grid-template-columns:repeat(4,minmax(0,1fr))}
  .side-foot{display:none}
  .stats,.two{grid-template-columns:1fr}
}
@media (max-width:720px){
  .main{padding:18px 14px 32px}
  .topbar,.detail-head,.split{display:grid}
  .stats{grid-template-columns:repeat(2,minmax(0,1fr))}
  .form-grid{grid-template-columns:1fr}
  .nav{grid-template-columns:repeat(2,minmax(0,1fr))}
}
</style>
</head>
<body>
<div id="app" v-cloak>
  <div v-if="busy" class="loading-ribbon"><span></span></div>
  <div class="shell">
    <aside class="side">
      <div class="brand">
        <div class="brand-mark">Q</div>
        <div>
          <div class="brand-title">Qianwen Pool</div>
          <div class="brand-sub">gen2api style console</div>
        </div>
      </div>
      <nav class="nav">
        <button v-for="item in tabs" :key="item.key" :class="{active:tab===item.key}" @click="tab=item.key">
          <span>{{item.icon}}</span><span>{{item.name}}</span>
        </button>
      </nav>
      <div class="side-foot">
        <div class="split">
          <span class="muted">Default chat</span>
          <span class="badge hot">{{defaultChatModel}}</span>
        </div>
        <p class="hint" style="margin-top:10px">Only accounts that pass a real model test should be routed by external API requests.</p>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div>
          <div class="eyebrow">QIANWEN WEB REVERSE PROXY</div>
          <h1>{{title}}</h1>
          <p class="subline">QR login account pool, SQLite storage, chat/image/video routing, no Redis.</p>
        </div>
        <div class="toolbar">
          <button class="btn" @click="refreshAll">Refresh</button>
          <button class="btn primary" @click="openAdd">Add account</button>
        </div>
      </header>

      <section v-show="tab==='accounts'" class="tab-panel">
        <div class="grid stats">
          <div class="card metric"><div class="metric-label">Accounts</div><div class="metric-value">{{accounts.length}}</div><div class="metric-meta">SQLite account pool</div></div>
          <div class="card metric"><div class="metric-label">Valid</div><div class="metric-value">{{validCount}}</div><div class="metric-meta">Passed real chat probe</div></div>
          <div class="card metric"><div class="metric-label">QR sessions</div><div class="metric-value">{{sessions.length}}</div><div class="metric-meta">{{activeSessionCount}} active</div></div>
          <div class="card metric"><div class="metric-label">Tasks</div><div class="metric-value">{{tasks.length}}</div><div class="metric-meta">{{taskBreakdown}}</div></div>
        </div>

        <div class="grid two">
          <div class="card">
            <div class="split" style="margin-bottom:14px">
              <h2>Account pool</h2>
              <button class="btn ghost" @click="loadAccounts">Sync state</button>
            </div>
            <div class="account-list">
              <div v-for="account in accounts" :key="account.id" :class="['account-card',{active:selectedId===account.id}]" @click="selectAccount(account.id)">
                <div class="account-head">
                  <div>
                    <div class="account-name">{{account.name}}</div>
                    <div class="account-id">{{account.id}}</div>
                  </div>
                  <span :class="['badge',statusClass(account.status)]">{{account.status || 'unknown'}}</span>
                </div>
                <div class="badges">
                  <span v-if="account.enabled" class="badge valid">enabled</span>
                  <span v-else class="badge disabled">disabled</span>
                  <span :class="['badge',account.type]">{{account.type}}</span>
                  <span v-for="cap in capabilityList(account)" :key="cap" class="badge hot">{{cap}}</span>
                </div>
                <div class="hint mono">{{account.last_error || account.cookie_string || 'Waiting for QR login capture'}}</div>
              </div>
              <div v-if="!accounts.length" class="empty">No accounts yet. Click Add account to start QR login.</div>
            </div>
          </div>

          <div class="card" v-if="selectedAccount">
            <div class="detail-head">
              <div class="detail-title">
                <div class="avatar">{{initial(selectedAccount.name)}}</div>
                <div>
                  <h2>{{selectedAccount.name}}</h2>
                  <div class="account-id">{{selectedAccount.id}}</div>
                </div>
              </div>
              <span :class="['badge',statusClass(selectedAccount.status)]">{{selectedAccount.status || 'unknown'}}</span>
            </div>
            <div class="actions" style="margin-bottom:16px">
              <button class="btn primary" @click="testAccount(selectedAccount.id)" :disabled="accountProbe.loading">Test</button>
              <button class="btn" @click="syncQuota(selectedAccount.id)" :disabled="accountProbe.loading">Sync quota</button>
              <button class="btn danger" @click="deleteAccount(selectedAccount.id)">Delete account</button>
            </div>
            <div class="grid" style="gap:0">
              <div class="kv"><span>Type</span><strong>{{selectedAccount.type}}</strong></div>
              <div class="kv"><span>Capabilities</span><span class="mono">{{formatCaps(selectedAccount.capabilities_json)}}</span></div>
              <div class="kv"><span>Last test</span><span>{{selectedAccount.last_test_at || '-'}}</span></div>
              <div class="kv"><span>Last success</span><span>{{selectedAccount.last_success_at || '-'}}</span></div>
              <div class="kv"><span>Last quota sync</span><span>{{selectedAccount.last_quota_sync_at || '-'}}</span></div>
              <div class="kv"><span>Error</span><span>{{selectedAccount.last_error || '-'}}</span></div>
            </div>
            <div v-if="accountProbe.message" style="margin-top:14px">
              <span :class="['badge',statusClass(accountProbe.status)]">{{accountProbe.status}}</span>
              <span class="hint" style="margin-left:8px">{{accountProbe.message}}</span>
            </div>
          </div>
          <div class="card" v-else>
            <div class="empty">Select an account to view details.</div>
          </div>
        </div>
      </section>

      <section v-show="tab==='sessions'" class="tab-panel">
        <div class="grid two">
          <div class="card">
            <div class="split" style="margin-bottom:14px">
              <h2>QR login sessions</h2>
              <button class="btn ghost" @click="loadSessions">Refresh sessions</button>
            </div>
            <div class="account-list">
              <div v-for="session in sessions" :key="session.id" :class="['account-card',{active:selectedSessionId===session.id}]" @click="selectSession(session.id)">
                <div class="account-head">
                  <div>
                    <div class="account-name">{{session.name}}</div>
                    <div class="account-id">{{session.id}}</div>
                  </div>
                  <span :class="['badge',statusClass(session.status)]">{{session.status}}</span>
                </div>
                <div class="badges">
                  <span class="badge hot">cookies {{session.cookie_count || 0}}</span>
                  <span v-if="session.account_id" class="badge valid">captured</span>
                </div>
                <div class="hint mono">{{session.message || '-'}}</div>
              </div>
              <div v-if="!sessions.length" class="empty">No QR sessions. Add an account to create one.</div>
            </div>
          </div>
          <div class="card" v-if="selectedSession">
            <div class="detail-head">
              <div class="detail-title">
                <div class="avatar">QR</div>
                <div>
                  <h2>{{selectedSession.name}}</h2>
                  <div class="account-id">{{selectedSession.id}}</div>
                </div>
              </div>
              <span :class="['badge',statusClass(selectedSession.status)]">{{selectedSession.status}}</span>
            </div>
            <div class="actions" style="margin-bottom:16px">
              <button class="btn" @click="clickLoginEntry(selectedSession.id)">Click login entry</button>
              <button class="btn" @click="refreshSession(selectedSession.id)">Refresh QR</button>
              <button class="btn primary" @click="captureSession(selectedSession.id)">Confirm scan</button>
              <button class="btn danger" @click="deleteSession(selectedSession.id)">Delete session</button>
            </div>
            <img class="shot" :src="screenshotUrl(selectedSession.id)" alt="Qianwen login screenshot">
            <div class="grid" style="gap:0;margin-top:14px">
              <div class="kv"><span>Message</span><span>{{selectedSession.message || '-'}}</span></div>
              <div class="kv"><span>Account</span><span class="mono">{{selectedSession.account_id || '-'}}</span></div>
              <div class="kv"><span>Updated</span><span>{{selectedSession.updated_at || '-'}}</span></div>
            </div>
          </div>
          <div class="card" v-else>
            <div class="empty">Select a QR session to view the live screenshot.</div>
          </div>
        </div>
      </section>

      <section v-show="tab==='tasks'" class="tab-panel">
        <div class="table-wrap">
          <table>
            <thead><tr><th>ID</th><th>Type</th><th>Model</th><th>Status</th><th>Account</th><th>Error</th><th>Created</th></tr></thead>
            <tbody>
              <tr v-for="task in tasks" :key="task.id">
                <td class="mono">{{task.id}}</td>
                <td>{{task.type}}</td>
                <td>{{task.model || '-'}}</td>
                <td><span :class="['badge',statusClass(task.status)]">{{task.status}}</span></td>
                <td class="mono">{{task.provider_account_id || '-'}}</td>
                <td>{{task.error_message || '-'}}</td>
                <td>{{task.created_at || '-'}}</td>
              </tr>
              <tr v-if="!tasks.length"><td colspan="7" class="empty">No tasks recorded yet.</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-show="tab==='system'" class="tab-panel">
        <div class="grid two">
          <div class="card">
            <h2 style="margin-bottom:12px">Runtime</h2>
            <div class="kv"><span>Service</span><span>{{summary.service.name || 'QIANWEN-WEB-01'}}</span></div>
            <div class="kv"><span>Listen</span><span class="mono">{{summary.service.host || '0.0.0.0'}}:{{summary.service.port || '-'}}</span></div>
            <div class="kv"><span>Data dir</span><span class="mono">{{summary.service.data_dir || '-'}}</span></div>
            <div class="kv"><span>SQLite</span><span class="mono">{{summary.service.database_path || '-'}}</span></div>
            <div class="kv"><span>Public URL</span><span class="mono">{{summary.service.public_base_url || locationOrigin}}</span></div>
            <div class="kv"><span>Guest pool</span><span>{{summary.service.guest_pool_size}}</span></div>
          </div>
          <div class="card">
            <h2 style="margin-bottom:12px">Models</h2>
            <div class="badges">
              <span v-for="model in models" :key="model.id" class="badge hot">{{model.id}}</span>
            </div>
            <h2 style="margin:18px 0 12px">Admin key</h2>
            <p class="hint">The key from the URL is saved into localStorage so refreshes keep the same authenticated console.</p>
            <pre class="out" style="margin-top:12px">{{adminKey ? 'stored in localStorage:qianwenAdminKey' : 'missing key'}}</pre>
          </div>
        </div>
      </section>
    </main>
  </div>

  <div v-if="addModal" class="overlay" @click.self="closeAdd">
    <div class="modal">
      <div class="modal-head">
        <div>
          <h2>Add Qianwen account</h2>
          <p class="subline">Create a server-side Chromium QR login session, then capture the logged-in qianwen.com account into SQLite.</p>
        </div>
        <button class="btn ghost" @click="closeAdd">Close</button>
      </div>
      <div class="guide">
        <div>1. Enter a readable account name.</div>
        <div>2. The server opens Chromium and loads qianwen.com.</div>
        <div>3. Scan the QR or complete the page login flow shown in the screenshot.</div>
        <div>4. Click Confirm scan only after the screenshot shows a logged-in page.</div>
        <div>5. Run Test; real model output is required before the account should receive traffic.</div>
      </div>
      <label>Account name
        <input v-model.trim="newAccount.name" placeholder="Example: qianwen-main-01" @keyup.enter="createAccount">
      </label>
      <div class="actions" style="margin-top:16px">
        <button class="btn ghost" @click="closeAdd">Cancel</button>
        <button class="btn primary" @click="createAccount" :disabled="!newAccount.name">Generate QR</button>
      </div>
    </div>
  </div>

  <div class="toast" :class="{show:toast.show}">{{toast.text}}</div>
</div>

<script>
const {createApp,ref,reactive,computed,onMounted,onBeforeUnmount}=Vue;
createApp({
  setup(){
    const initialKey=new URLSearchParams(window.location.search).get("key") || window.localStorage.getItem("qianwenAdminKey") || "";
    if(initialKey) window.localStorage.setItem("qianwenAdminKey", initialKey);
    const adminKey=initialKey;
    const locationOrigin=window.location.origin;
    const tabs=[
      {key:"accounts",name:"Accounts",icon:"A"},
      {key:"sessions",name:"QR sessions",icon:"Q"},
      {key:"tasks",name:"Tasks",icon:"T"},
      {key:"system",name:"System",icon:"S"}
    ];
    const tab=ref("accounts");
    const busy=ref(false);
    const accounts=ref([]);
    const sessions=ref([]);
    const tasks=ref([]);
    const models=ref([]);
    const selectedId=ref("");
    const selectedSessionId=ref("");
    const screenshotTick=ref(0);
    const addModal=ref(false);
    const newAccount=reactive({name:""});
    const accountProbe=reactive({loading:false,status:"",message:""});
    const toast=reactive({show:false,text:"",timer:0});
    const summary=reactive({service:{},accounts:{},tasks:{}});
    let pollTimer=0;

    const title=computed(function(){
      const found=tabs.find(function(item){return item.key===tab.value});
      return found ? found.name : "Accounts";
    });
    const selectedAccount=computed(function(){
      return accounts.value.find(function(account){return account.id===selectedId.value}) || null;
    });
    const selectedSession=computed(function(){
      return sessions.value.find(function(session){return session.id===selectedSessionId.value}) || null;
    });
    const validCount=computed(function(){
      return accounts.value.filter(function(account){return account.status==="valid"}).length;
    });
    const activeSessionCount=computed(function(){
      return sessions.value.filter(function(session){
        return ["captured","failed","expired"].indexOf(session.status)===-1;
      }).length;
    });
    const taskBreakdown=computed(function(){
      return breakdown(summary.tasks && summary.tasks.status);
    });
    const defaultChatModel=computed(function(){
      const chat=models.value.find(function(model){return model.type==="chat" && model.is_default});
      return chat ? chat.id : "tongyi-qwen3-max-model";
    });

    function headers(json){
      const h={};
      if(adminKey) h["X-Admin-Key"]=adminKey;
      if(json!==false) h["Content-Type"]="application/json";
      return h;
    }
    async function api(path,opts){
      const options=opts || {};
      busy.value=true;
      try{
        const resp=await fetch("/api"+path,Object.assign({},options,{headers:Object.assign(headers(!(options.body instanceof FormData)),options.headers || {})}));
        const text=await resp.text();
        let data={};
        try{data=text ? JSON.parse(text) : {}}catch(err){data={message:text}}
        if(!resp.ok){
          throw new Error(errorMessage(data) || resp.statusText || ("HTTP "+resp.status));
        }
        return data;
      }finally{
        busy.value=false;
      }
    }
    function errorMessage(data){
      if(!data) return "";
      if(data.error && data.error.message) return data.error.message;
      return data.message || data.detail || "";
    }
    function showToast(text){
      toast.text=text || "";
      toast.show=true;
      if(toast.timer) clearTimeout(toast.timer);
      toast.timer=setTimeout(function(){toast.show=false},3200);
    }
    async function refreshAll(){
      await Promise.all([loadSummary(),loadAccounts(),loadSessions(),loadTasks(),loadModels()]);
    }
    async function loadSummary(){
      try{
        const data=await api("/admin/summary");
        Object.assign(summary.service,data.service || {});
        summary.accounts=data.accounts || {};
        summary.tasks=data.tasks || {};
      }catch(err){
        showToast(err.message);
      }
    }
    async function loadAccounts(){
      const data=await api("/accounts");
      accounts.value=data.data || [];
      if(!selectedId.value && accounts.value.length) selectedId.value=accounts.value[0].id;
      if(selectedId.value && !accounts.value.some(function(account){return account.id===selectedId.value})){
        selectedId.value=accounts.value[0] ? accounts.value[0].id : "";
      }
    }
    async function loadSessions(){
      const data=await api("/login-sessions");
      sessions.value=data.data || [];
      if(!selectedSessionId.value && sessions.value.length) selectedSessionId.value=sessions.value[0].id;
      if(selectedSessionId.value && !sessions.value.some(function(session){return session.id===selectedSessionId.value})){
        selectedSessionId.value=sessions.value[0] ? sessions.value[0].id : "";
      }
      screenshotTick.value++;
    }
    async function loadTasks(){
      const data=await api("/tasks?limit=80");
      tasks.value=data.data || [];
    }
    async function loadModels(){
      const data=await api("/models");
      models.value=data.data || [];
    }
    function selectAccount(id){
      selectedId.value=id;
      accountProbe.status="";
      accountProbe.message="";
    }
    function selectSession(id){
      selectedSessionId.value=id;
      tab.value="sessions";
      screenshotTick.value++;
    }
    function openAdd(){
      newAccount.name="";
      addModal.value=true;
      setTimeout(function(){
        const input=document.querySelector(".modal input");
        if(input) input.focus();
      },80);
    }
    function closeAdd(){
      addModal.value=false;
      newAccount.name="";
    }
    async function createAccount(){
      const name=(newAccount.name || "").trim();
      if(!name){
        showToast("Enter an account name first.");
        return;
      }
      try{
        const data=await api("/accounts",{method:"POST",body:JSON.stringify({name:name})});
        closeAdd();
        await refreshAll();
        if(data.data && data.data.id){
          selectedSessionId.value=data.data.id;
          tab.value="sessions";
          screenshotTick.value++;
        }
        showToast("QR login session created.");
      }catch(err){
        showToast(err.message);
      }
    }
    async function testAccount(id){
      accountProbe.loading=true;
      accountProbe.status="";
      accountProbe.message="";
      try{
        const result=await api("/accounts/"+encodeURIComponent(id)+"/test",{method:"POST",body:JSON.stringify({capability:"chat"})});
        accountProbe.status=result.ok ? "valid" : "invalid";
        accountProbe.message=result.message || "Account test completed.";
        showToast(accountProbe.message);
      }catch(err){
        accountProbe.status="error";
        accountProbe.message=err.message;
        showToast(err.message);
      }finally{
        accountProbe.loading=false;
        await refreshAll();
      }
    }
    async function syncQuota(id){
      accountProbe.loading=true;
      accountProbe.status="";
      accountProbe.message="";
      try{
        const data=await api("/accounts/"+encodeURIComponent(id)+"/quota/sync",{method:"POST"});
        accountProbe.status=data.ok ? "valid" : "unknown";
        accountProbe.message=data.message || "Quota sync completed.";
        showToast(accountProbe.message);
      }catch(err){
        accountProbe.status="error";
        accountProbe.message=err.message;
        showToast(err.message);
      }finally{
        accountProbe.loading=false;
        await refreshAll();
      }
    }
    async function deleteAccount(id){
      const account=accounts.value.find(function(item){return item.id===id});
      const label=account ? account.name+" / "+account.id : id;
      const ok=window.confirm("Delete account "+label+"?\n\nThis will remove the SQLite account row, account events, detach historical task ownership, and close captured QR sessions for this account.\n\nThis action cannot be undone.");
      if(!ok) return;
      try{
        const result=await api("/accounts/"+encodeURIComponent(id),{method:"DELETE"});
        selectedId.value="";
        await refreshAll();
        const data=result.data || {};
        showToast("Account deleted. events="+(data.account_events_deleted || 0)+", tasks_detached="+(data.tasks_detached || 0)+".");
      }catch(err){
        showToast(err.message);
      }
    }
    async function clickLoginEntry(id){
      try{
        await api("/login-sessions/"+encodeURIComponent(id)+"/click-login",{method:"POST"});
        screenshotTick.value++;
        await loadSessions();
        showToast("Clicked login entry.");
      }catch(err){
        showToast(err.message);
      }
    }
    async function refreshSession(id){
      try{
        await api("/login-sessions/"+encodeURIComponent(id)+"/refresh",{method:"POST"});
        screenshotTick.value++;
        await loadSessions();
        showToast("QR session refreshed.");
      }catch(err){
        showToast(err.message);
      }
    }
    async function captureSession(id){
      try{
        const result=await api("/login-sessions/"+encodeURIComponent(id)+"/capture",{method:"POST"});
        await refreshAll();
        if(result.data && result.data.id){
          selectedId.value=result.data.id;
          tab.value="accounts";
        }
        showToast("Account captured. Run Test before routing traffic.");
      }catch(err){
        showToast(err.message);
      }
    }
    async function deleteSession(id){
      const session=sessions.value.find(function(item){return item.id===id});
      const label=session ? session.name+" / "+session.id : id;
      const ok=window.confirm("Delete QR session "+label+"?\n\nThis closes the Chromium process and removes its temporary browser profile. Expired QR sessions should be deleted to avoid memory buildup.");
      if(!ok) return;
      try{
        await api("/login-sessions/"+encodeURIComponent(id),{method:"DELETE"});
        if(selectedSessionId.value===id) selectedSessionId.value="";
        await loadSessions();
        showToast("QR session deleted.");
      }catch(err){
        showToast(err.message);
      }
    }
    function screenshotUrl(id){
      screenshotTick.value;
      return "/api/login-sessions/"+encodeURIComponent(id)+"/screenshot?key="+encodeURIComponent(adminKey)+"&t="+Date.now()+"-"+screenshotTick.value;
    }
    function capabilityList(account){
      try{
        const caps=JSON.parse(account.capabilities_json || "{}");
        return Object.keys(caps).filter(function(key){return !!caps[key]});
      }catch(err){
        return [];
      }
    }
    function formatCaps(value){
      try{
        const caps=JSON.parse(value || "{}");
        const keys=Object.keys(caps).filter(function(key){return !!caps[key]});
        return keys.length ? keys.join(" / ") : "{}";
      }catch(err){
        return value || "-";
      }
    }
    function statusClass(value){
      const v=value || "unknown";
      if(["valid","succeeded","captured","login_detected"].indexOf(v)>=0) return v;
      if(["invalid","failed","capture_failed","expired","error"].indexOf(v)>=0) return v;
      return v;
    }
    function breakdown(obj){
      const keys=Object.keys(obj || {});
      if(!keys.length) return "none";
      return keys.map(function(key){return key+":"+obj[key]}).join(" / ");
    }
    function initial(name){
      return String(name || "Q").slice(0,1).toUpperCase();
    }
    onMounted(function(){
      refreshAll();
      pollTimer=setInterval(function(){
        if(tab.value==="sessions"){
          loadSessions().catch(function(){});
        }
      },5000);
    });
    onBeforeUnmount(function(){
      if(pollTimer) clearInterval(pollTimer);
      if(toast.timer) clearTimeout(toast.timer);
    });
    return{
      tabs,tab,title,busy,accounts,sessions,tasks,models,summary,selectedId,selectedSessionId,selectedAccount,selectedSession,
      validCount,activeSessionCount,taskBreakdown,defaultChatModel,addModal,newAccount,accountProbe,toast,adminKey,locationOrigin,
      refreshAll,loadAccounts,loadSessions,selectAccount,selectSession,openAdd,closeAdd,createAccount,testAccount,syncQuota,deleteAccount,
      clickLoginEntry,refreshSession,captureSession,deleteSession,screenshotUrl,capabilityList,formatCaps,statusClass,initial
    };
  }
}).mount("#app");
</script>
</body>
</html>`
