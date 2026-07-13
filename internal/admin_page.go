package internal

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>QIANWEN-WEB-01 账号池</title>
<script src="https://unpkg.com/vue@3/dist/vue.global.prod.js"></script>
<style>
:root{
  color:#f4f8ff;
  background:#020409;
  color-scheme:dark;
  font-family:Inter,"Segoe UI","Microsoft YaHei",Arial,sans-serif;
  --bg:#020409;
  --panel:#08111b;
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
  color:var(--text);
  background:
    linear-gradient(90deg,rgba(144,189,255,.018) 1px,transparent 1px),
    linear-gradient(rgba(144,189,255,.014) 1px,transparent 1px),
    linear-gradient(135deg,#010208 0%,#040813 48%,#080815 100%);
  background-size:34px 34px,34px 34px,auto;
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
.brand-title{font-size:20px;font-weight:900}
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
.toolbar,.actions{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
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
.badge.unknown,.badge.starting,.badge.opening,.badge.waiting_scan,.badge.disabled,.badge.processing,.badge.queued{background:rgba(255,209,102,.13);color:var(--amber)}
.detail-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:16px}
.detail-title{display:flex;align-items:center;gap:12px}
.avatar{width:46px;height:46px;border-radius:var(--radius-md);display:grid;place-items:center;background:var(--grad);color:#00131a;font-weight:900}
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
textarea{resize:vertical;min-height:118px}
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
.shot{width:100%;min-height:360px;border-radius:var(--radius-md);border:1px solid var(--line-strong);background:#03070c;object-fit:contain}
.overlay{position:fixed;inset:0;z-index:60;background:rgba(2,4,9,.72);backdrop-filter:blur(18px);display:grid;place-items:center;padding:24px;animation:fadeIn 180ms var(--ease)}
.modal{width:min(680px,100%);background:linear-gradient(180deg,rgba(12,24,38,.98),rgba(5,11,18,.98));border:1px solid rgba(54,231,255,.24);border-radius:var(--radius-xl);box-shadow:var(--shadow),var(--glow);padding:22px;animation:modalUp 240ms var(--ease)}
.modal.wide{width:min(980px,100%)}
.modal-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:16px}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
@keyframes modalUp{from{opacity:0;transform:translateY(12px) scale(.98)}to{opacity:1;transform:none}}
.guide{display:grid;gap:9px;margin:14px 0;padding:14px;border:1px solid rgba(41,64,87,.72);border-radius:var(--radius-md);background:rgba(54,231,255,.055);color:var(--muted);font-size:13px}
.scan-frame{margin-top:14px;border:1px solid var(--line-strong);border-radius:var(--radius-lg);background:#03070c;min-height:430px;display:grid;place-items:center;overflow:hidden}
.scan-frame img{width:100%;max-height:620px;object-fit:contain;display:block}
.scan-status{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-top:12px}
.empty{padding:38px;text-align:center;color:var(--muted)}
.split{display:flex;align-items:center;justify-content:space-between;gap:12px}
.hint{font-size:12px;color:var(--dim);line-height:1.6}
pre.out{max-height:420px;overflow:auto;white-space:pre-wrap;word-break:break-word;background:#050b12;border:1px solid var(--line);border-radius:var(--radius-md);padding:14px;color:#adf5ff}
.loading-ribbon{position:fixed;inset:0 0 auto;z-index:80;height:3px;overflow:hidden;background:rgba(54,231,255,.08)}
.loading-ribbon span{display:block;width:42%;height:100%;background:linear-gradient(90deg,transparent,var(--cyan),var(--blue),var(--violet),transparent);box-shadow:0 0 18px rgba(54,231,255,.62);animation:loadingSweep 1.1s var(--ease) infinite}
@keyframes loadingSweep{from{transform:translateX(-100%)}to{transform:translateX(240%)}}
.toast{position:fixed;left:50%;bottom:24px;z-index:90;transform:translateX(-50%) translateY(16px);opacity:0;pointer-events:none;max-width:min(720px,calc(100vw - 32px));padding:12px 14px;border-radius:var(--radius-sm);border:1px solid rgba(54,231,255,.28);background:rgba(7,15,25,.96);box-shadow:var(--shadow),var(--glow);color:var(--text);transition:opacity 180ms var(--ease),transform 180ms var(--ease)}
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
          <div class="brand-title">千问账号池</div>
          <div class="brand-sub">QIANWEN-WEB-01</div>
        </div>
      </div>
      <nav class="nav">
        <button v-for="item in tabs" :key="item.key" :class="{active:tab===item.key}" @click="tab=item.key">
          <span>{{item.icon}}</span><span>{{item.name}}</span>
        </button>
      </nav>
      <div class="side-foot">
        <div class="split"><span class="muted">默认对话模型</span><span class="badge hot">{{defaultChatModel}}</span></div>
        <p class="hint" style="margin-top:10px">只有测活通过的账号才建议参与接口调度；二维码会话过期后请及时删除。</p>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div>
          <div class="eyebrow">千问网页反代</div>
          <h1>{{title}}</h1>
          <p class="subline">扫码登录账号池、SQLite 存储、对话 / 生图 / 生视频接口测试与请求日志。</p>
        </div>
        <div class="toolbar">
          <button class="btn" @click="refreshAll">刷新</button>
          <button class="btn primary" @click="openAdd">新增账号</button>
        </div>
      </header>

      <section v-show="tab==='accounts'" class="tab-panel">
        <div class="grid stats">
          <div class="card metric"><div class="metric-label">账号总数</div><div class="metric-value">{{accounts.length}}</div><div class="metric-meta">SQLite 本地账号池</div></div>
          <div class="card metric"><div class="metric-label">有效账号</div><div class="metric-value">{{validCount}}</div><div class="metric-meta">真实模型测活通过</div></div>
          <div class="card metric"><div class="metric-label">扫码会话</div><div class="metric-value">{{sessions.length}}</div><div class="metric-meta">{{activeSessionCount}} 个仍在等待</div></div>
          <div class="card metric"><div class="metric-label">任务记录</div><div class="metric-value">{{tasks.length}}</div><div class="metric-meta">{{taskBreakdown}}</div></div>
        </div>

        <div class="grid two">
          <div class="card">
            <div class="split" style="margin-bottom:14px"><h2>账号池</h2><button class="btn ghost" @click="loadAccounts">同步状态</button></div>
            <div class="account-list">
              <div v-for="account in accounts" :key="account.id" :class="['account-card',{active:selectedId===account.id}]" @click="selectAccount(account.id)">
                <div class="account-head">
                  <div><div class="account-name">{{account.name}}</div><div class="account-id">{{account.id}}</div></div>
                  <span :class="['badge',statusClass(account.status)]">{{statusText(account.status)}}</span>
                </div>
                <div class="badges">
                  <span v-if="account.enabled" class="badge valid">启用</span>
                  <span v-else class="badge disabled">禁用</span>
                  <span :class="['badge',account.type]">{{typeText(account.type)}}</span>
                  <span v-for="cap in capabilityList(account)" :key="cap" class="badge hot">{{capText(cap)}}</span>
                </div>
                <div class="hint mono">{{account.last_error || account.cookie_string || '等待扫码捕获账号'}}</div>
              </div>
              <div v-if="!accounts.length" class="empty">暂无账号，点击“新增账号”开始扫码登录。</div>
            </div>
          </div>

          <div class="card" v-if="selectedAccount">
            <div class="detail-head">
              <div class="detail-title">
                <div class="avatar">{{initial(selectedAccount.name)}}</div>
                <div><h2>{{selectedAccount.name}}</h2><div class="account-id">{{selectedAccount.id}}</div></div>
              </div>
              <span :class="['badge',statusClass(selectedAccount.status)]">{{statusText(selectedAccount.status)}}</span>
            </div>
            <div class="actions" style="margin-bottom:16px">
              <button class="btn primary" @click="startMaintenance(selectedAccount.id)">检修</button>
              <button class="btn primary" @click="testAccount(selectedAccount.id,'chat')" :disabled="accountProbe.loading">测对话</button>
              <button class="btn" @click="testAccount(selectedAccount.id,'image')" :disabled="accountProbe.loading">测生图</button>
              <button class="btn" @click="testAccount(selectedAccount.id,'video')" :disabled="accountProbe.loading">测视频</button>
              <button class="btn" @click="syncQuota(selectedAccount.id)" :disabled="accountProbe.loading">同步额度</button>
              <button class="btn" @click="openLatestSessionModal">查看扫码会话</button>
              <button class="btn danger" v-if="!isProtectedAccount(selectedAccount.id)" @click="deleteAccount(selectedAccount.id)">删除账号</button>
            </div>
            <div class="grid" style="gap:0">
              <div class="kv"><span>账号类型</span><strong>{{typeText(selectedAccount.type)}}</strong></div>
              <div class="kv"><span>能力</span><span class="mono">{{formatCaps(selectedAccount.capabilities_json)}}</span></div>
              <div class="kv"><span>最后测活</span><span>{{selectedAccount.last_test_at || '-'}}</span></div>
              <div class="kv"><span>最后成功</span><span>{{selectedAccount.last_success_at || '-'}}</span></div>
              <div class="kv"><span>额度同步</span><span>{{selectedAccount.last_quota_sync_at || '-'}}</span></div>
              <div class="kv"><span>错误信息</span><span>{{selectedAccount.last_error || '-'}}</span></div>
            </div>
            <div v-if="accountProbe.message" style="margin-top:14px">
              <span :class="['badge',statusClass(accountProbe.status)]">{{statusText(accountProbe.status)}}</span>
              <span class="hint" style="margin-left:8px">{{capText(accountProbe.capability)}} · {{accountProbe.message}}</span>
              <pre v-if="accountProbe.response" class="out" style="margin-top:12px">{{accountProbe.response}}</pre>
            </div>
          </div>
          <div class="card" v-else><div class="empty">请选择一个账号查看详情。</div></div>
        </div>

        <div class="grid two" style="margin-top:16px">
          <div class="card">
            <div class="split" style="margin-bottom:14px"><h2>扫码会话</h2><button class="btn ghost" @click="loadSessions">刷新会话</button></div>
            <div class="account-list">
              <div v-for="session in sessions" :key="session.id" :class="['account-card',{active:selectedSessionId===session.id}]" @click="selectSession(session.id)">
                <div class="account-head">
                  <div><div class="account-name">{{session.name}}</div><div class="account-id">{{session.id}}</div></div>
                  <span :class="['badge',statusClass(session.status)]">{{statusText(session.status)}}</span>
                </div>
                <div class="badges">
                  <span class="badge hot">Cookie {{session.cookie_count || 0}}</span>
                  <span v-if="session.account_id" class="badge valid">已捕获</span>
                </div>
                <div class="hint mono">{{session.message || '-'}}</div>
              </div>
              <div v-if="!sessions.length" class="empty">暂无扫码会话。新增账号后会自动生成。</div>
            </div>
          </div>
          <div class="card" v-if="selectedSession">
            <div class="detail-head">
              <div class="detail-title">
                <div class="avatar">QR</div>
                <div><h2>{{selectedSession.name}}</h2><div class="account-id">{{selectedSession.id}}</div></div>
              </div>
              <span :class="['badge',statusClass(selectedSession.status)]">{{statusText(selectedSession.status)}}</span>
            </div>
            <div class="actions" style="margin-bottom:16px">
              <button v-if="selectedSession.novnc_url" class="btn" @click="openNoVNC(selectedSession.novnc_url)">noVNC</button>
              <button class="btn" @click="clickLoginEntry(selectedSession.id)">点击登录入口</button>
              <button class="btn" @click="openSessionModal(selectedSession.id)">打开扫码窗口</button>
              <button v-if="selectedSession.mode!=='maintenance'" class="btn" @click="refreshSession(selectedSession.id)">刷新二维码</button>
              <button class="btn primary" @click="captureSession(selectedSession.id)">确认扫码</button>
              <button class="btn danger" @click="deleteSession(selectedSession.id)">删除会话</button>
            </div>
            <img class="shot" :src="screenshotUrl(selectedSession.id)" alt="千问登录截图">
            <div class="grid" style="gap:0;margin-top:14px">
              <div class="kv"><span>提示</span><span>{{selectedSession.message || '-'}}</span></div>
              <div class="kv"><span>账号</span><span class="mono">{{selectedSession.account_id || '-'}}</span></div>
              <div class="kv"><span>更新时间</span><span>{{selectedSession.updated_at || '-'}}</span></div>
            </div>
          </div>
          <div class="card" v-else><div class="empty">请选择一个扫码会话查看截图。</div></div>
        </div>
      </section>

      <section v-show="tab==='test'" class="tab-panel">
        <div class="card">
          <div class="form-grid">
            <label>账号
              <select v-model="test.account_id">
                <option value="">自动调度</option>
                <option v-for="a in accounts" :key="a.id" :value="a.id">{{a.name}} / {{a.id}}</option>
              </select>
            </label>
            <label>模型
              <select v-model="test.model">
                <option v-for="m in models" :key="m.id" :value="m.id">{{m.id}}</option>
              </select>
            </label>
          </div>
          <div class="form-grid" style="margin-top:12px">
            <label>视频秒数
              <select v-model.number="test.duration">
                <option :value="5">5 秒</option>
                <option :value="10">10 秒</option>
              </select>
            </label>
            <label>画面比例
              <select v-model="test.ratio">
                <option value="16:9">16:9</option>
                <option value="9:16">9:16</option>
                <option value="1:1">1:1</option>
              </select>
            </label>
          </div>
          <label style="margin-top:12px">提示词
            <textarea v-model="test.prompt" placeholder="输入要发送到千问的测试内容"></textarea>
          </label>
          <div class="actions" style="margin-top:14px">
            <button class="btn primary" :disabled="test.loading" @click="runTest">{{test.loading ? '请求中' : '发送测试'}}</button>
            <button class="btn" @click="copy(test.output)">复制结果</button>
            <span class="badge hot">{{modelKind(test.model)}}</span>
          </div>
          <pre v-if="test.output" class="out" style="margin-top:14px">{{test.output}}</pre>
          <div v-if="test.error" class="card" style="margin-top:14px;border-color:rgba(255,102,117,.35);color:var(--red)">{{test.error}}</div>
        </div>
      </section>

      <section v-show="tab==='logs'" class="tab-panel">
        <div class="table-wrap">
          <table>
            <thead><tr><th>时间</th><th>方法</th><th>路径</th><th>模型</th><th>账号</th><th>状态</th><th>耗时</th></tr></thead>
            <tbody>
              <tr v-for="log in logs" :key="String(log.ts)+log.path+String(log.ms)">
                <td class="mono">{{fmtClock(log.ts)}}</td>
                <td><span class="badge hot">{{log.method}}</span></td>
                <td class="mono">{{log.path}}</td>
                <td>{{log.model || '-'}}</td>
                <td class="mono">{{log.account_id || '-'}}</td>
                <td :style="{color:log.status < 400 ? 'var(--green)' : 'var(--red)'}">{{log.status}}</td>
                <td>{{log.ms}}ms</td>
              </tr>
              <tr v-if="!logs.length"><td colspan="7" class="empty">暂无请求日志。使用“接口测试”发送一次请求后会出现记录。</td></tr>
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
            <div class="kv"><span>Public URL</span><span class="mono">{{summary.service.public_base_url || locationOrigin}}</span></div>
            <div class="kv"><span>Database</span><span class="mono">{{summary.service.database_path || '-'}}</span></div>
            <div class="kv"><span>Data Dir</span><span class="mono">{{summary.service.data_dir || '-'}}</span></div>
          </div>
          <div class="card">
            <h2 style="margin-bottom:12px">API Key Manager</h2>
            <div class="kv"><span>API Base</span><span class="mono">{{apiBase}}</span></div>
            <label style="margin-top:12px">API Key<input v-model="serviceKey.api_key" class="mono" autocomplete="off"></label>
            <div class="actions" style="margin-top:14px"><button class="btn" @click="copy(apiBase)">Copy API Base</button><button class="btn" @click="copy(serviceKey.api_key)">Copy API Key</button><button class="btn primary" @click="saveServiceKey">Save API Key</button></div>
            <p class="hint" style="margin-top:12px">This key is used by NewAPI channels. Saving takes effect immediately and persists to SQLite; the environment variable is only the initial fallback.</p>
            <span v-if="serviceKey.message" class="badge valid" style="margin-top:10px">{{serviceKey.message}}</span>
          </div>
          <div class="card">
            <h2 style="margin-bottom:12px">Models</h2>
            <div class="badges"><span v-for="model in models" :key="model.id" class="badge hot">{{model.id}}</span></div>
            <h2 style="margin:18px 0 12px">Admin Login Cache</h2>
            <p class="hint">The admin key from URL is stored in browser localStorage for this WebUI only.</p>
            <pre class="out" style="margin-top:12px">{{adminKey ? 'Saved in localStorage: qianwenAdminKey' : 'No admin key detected'}}</pre>
          </div>
        </div>
      </section>
    </main>
  </div>

  <div v-if="addModal" class="overlay" @click.self="closeAdd">
    <div class="modal">
      <div class="modal-head">
        <div>
          <h2>新增千问账号</h2>
          <p class="subline">创建服务端 Chromium 扫码会话，扫码后捕获 qianwen.com 登录态并写入 SQLite 账号池。</p>
        </div>
        <button class="btn ghost" @click="closeAdd">关闭</button>
      </div>
      <div class="guide">
        <div>1. 填写一个容易识别的账号名称。</div>
        <div>2. 点击生成二维码，服务器会打开独立 Chromium profile。</div>
        <div>3. 在下方截图中完成扫码或登录流程。</div>
        <div>4. 页面进入已登录状态后，点击“确认扫码”。</div>
        <div>5. 保存后点击“测活”，拿到真实模型返回才建议参与调度。</div>
      </div>
      <label>账号名称
        <input v-model.trim="newAccount.name" placeholder="例如：千问主号 01" @keyup.enter="createAccount">
      </label>
      <div class="actions" style="margin-top:16px">
        <button class="btn ghost" @click="closeAdd">取消</button>
        <button class="btn primary" @click="createAccount" :disabled="!newAccount.name">生成二维码</button>
      </div>
    </div>
  </div>

  <div v-if="scanModal" class="overlay" @click.self="closeScanModal">
    <div class="modal wide" v-if="selectedSession">
      <div class="modal-head">
        <div>
          <h2>扫码登录</h2>
          <p class="subline">{{selectedSession.name}} / {{selectedSession.id}}</p>
        </div>
        <button class="btn ghost" @click="closeScanModal">关闭</button>
      </div>
      <div class="guide">
        <div>1. 在截图中确认已经进入千问登录页；如果二维码过期，点击“刷新二维码”。</div>
        <div>2. 使用千问 / 淘宝 / 支付宝支持的方式完成扫码或授权登录。</div>
        <div>3. 截图显示已经进入已登录页面后，再点击“确认扫码”。</div>
        <div>4. 捕获成功后账号会写入 SQLite，随后请对账号执行“测活”。</div>
      </div>
      <div class="scan-frame">
        <img :src="screenshotUrl(selectedSession.id)" alt="千问登录截图">
      </div>
      <div class="scan-status">
        <span :class="['badge',statusClass(selectedSession.status)]">{{statusText(selectedSession.status)}}</span>
        <span class="badge hot">Cookie {{selectedSession.cookie_count || 0}}</span>
        <span v-if="selectedSession.account_id" class="badge valid">已捕获 {{selectedSession.account_id}}</span>
        <span class="hint">{{selectedSession.message || '等待扫码'}}</span>
      </div>
      <div class="actions" style="margin-top:16px">
        <button class="btn" @click="clickLoginEntry(selectedSession.id)">点击登录入口</button>
        <button class="btn" @click="refreshSession(selectedSession.id)">刷新二维码</button>
        <button class="btn primary" @click="captureSession(selectedSession.id)">确认扫码</button>
        <button class="btn danger" @click="deleteSession(selectedSession.id)">删除会话</button>
      </div>
    </div>
    <div class="modal" v-else>
      <div class="empty">当前没有可查看的扫码会话。</div>
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
      {key:"accounts",name:"账号池",icon:"◎"},
      {key:"test",name:"接口测试",icon:"↯"},
      {key:"logs",name:"请求日志",icon:"≡"},
      {key:"system",name:"系统",icon:"⚙"}
    ];
    const tab=ref("accounts");
    const busy=ref(false);
    const accounts=ref([]);
    const sessions=ref([]);
    const tasks=ref([]);
    const models=ref([]);
    const logs=ref([]);
    const selectedId=ref("");
    const selectedSessionId=ref("");
    const screenshotTick=ref(0);
    const addModal=ref(false);
    const scanModal=ref(false);
    const newAccount=reactive({name:""});
    const accountProbe=reactive({loading:false,status:"",message:"",capability:"",response:""});
    const test=reactive({account_id:"",model:"tongyi-qwen3-max-model",prompt:"你好，请只回复一句话确认你可用。",duration:5,ratio:"16:9",output:"",error:"",loading:false});
    const toast=reactive({show:false,text:"",timer:0});
    const summary=reactive({service:{},accounts:{},tasks:{}});
    const serviceKey=reactive({api_key:"",message:""});
    let pollTimer=0;

    const title=computed(function(){const found=tabs.find(function(item){return item.key===tab.value});return found ? found.name : "账号池";});
    const selectedAccount=computed(function(){return accounts.value.find(function(account){return account.id===selectedId.value}) || null;});
    const selectedSession=computed(function(){return sessions.value.find(function(session){return session.id===selectedSessionId.value}) || null;});
    const validCount=computed(function(){return accounts.value.filter(function(account){return account.status==="valid"}).length;});
    const activeSessionCount=computed(function(){return sessions.value.filter(function(session){return ["captured","failed","expired"].indexOf(session.status)===-1;}).length;});
    const taskBreakdown=computed(function(){return breakdown(summary.tasks && summary.tasks.status);});
    const defaultChatModel=computed(function(){const chat=models.value.find(function(model){return model.type==="chat" && model.is_default});return chat ? chat.id : "tongyi-qwen3-max-model";});
    const apiBase=computed(function(){return locationOrigin+"/v1";});

    function adminHeaders(json){const h={};if(adminKey) h["X-Admin-Key"]=adminKey;if(json!==false) h["Content-Type"]="application/json";return h;}
    function apiHeaders(json){const h={};if(adminKey) h["Authorization"]="Bearer "+adminKey;if(json!==false) h["Content-Type"]="application/json";return h;}
    async function api(path,opts){
      const options=opts || {};
      busy.value=true;
      try{
        const resp=await fetch("/api"+path,Object.assign({},options,{headers:Object.assign(adminHeaders(!(options.body instanceof FormData)),options.headers || {})}));
        const text=await resp.text();
        let data={};
        try{data=text ? JSON.parse(text) : {}}catch(err){data={message:text};}
        if(!resp.ok){throw new Error(errorMessage(data) || resp.statusText || ("HTTP "+resp.status));}
        return data;
      }finally{busy.value=false;}
    }
    async function callProvider(path,body){
      const resp=await fetch(path,{method:"POST",headers:apiHeaders(true),body:JSON.stringify(body)});
      const text=await resp.text();
      let data={};
      try{data=text ? JSON.parse(text) : {}}catch(err){data={message:text};}
      if(!resp.ok){throw new Error(errorMessage(data) || text || resp.statusText);}
      return data;
    }
    function errorMessage(data){if(!data) return "";if(data.error && data.error.message) return data.error.message;return data.message || data.detail || "";}
    function showToast(text){toast.text=text || "";toast.show=true;if(toast.timer) clearTimeout(toast.timer);toast.timer=setTimeout(function(){toast.show=false;},3200);}
    async function refreshAll(){await Promise.all([loadSummary(),loadAccounts(),loadSessions(),loadTasks(),loadModels(),loadLogs(),loadServiceKey()]);}
    async function loadSummary(){try{const data=await api("/admin/summary");Object.assign(summary.service,data.service || {});summary.accounts=data.accounts || {};summary.tasks=data.tasks || {};}catch(err){showToast(err.message);}}
    async function loadAccounts(){const data=await api("/accounts");accounts.value=data.data || [];if(!selectedId.value && accounts.value.length) selectedId.value=accounts.value[0].id;if(selectedId.value && !accounts.value.some(function(account){return account.id===selectedId.value;})){selectedId.value=accounts.value[0] ? accounts.value[0].id : "";}}
    async function loadSessions(){const data=await api("/login-sessions");sessions.value=data.data || [];if(!selectedSessionId.value && sessions.value.length) selectedSessionId.value=sessions.value[0].id;if(selectedSessionId.value && !sessions.value.some(function(session){return session.id===selectedSessionId.value;})){selectedSessionId.value=sessions.value[0] ? sessions.value[0].id : "";}screenshotTick.value++;}
    async function loadTasks(){const data=await api("/tasks?limit=80");tasks.value=data.data || [];}
    async function loadModels(){const data=await api("/models");models.value=data.data || [];if(!models.value.some(function(model){return model.id===test.model;}) && models.value.length){test.model=models.value[0].id;}}
    async function loadLogs(){try{const data=await api("/logs");logs.value=(Array.isArray(data) ? data : []).slice().reverse();}catch(err){}}
    async function loadServiceKey(){const data=await api("/service/api-key");serviceKey.api_key=data.api_key||"";}
    async function saveServiceKey(){const data=await api("/service/api-key",{method:"PUT",body:JSON.stringify({api_key:serviceKey.api_key})});serviceKey.api_key=data.api_key||serviceKey.api_key;serviceKey.message="Saved and active";showToast("API Key saved");setTimeout(function(){serviceKey.message="";},2400);}
    function selectAccount(id){selectedId.value=id;accountProbe.status="";accountProbe.message="";accountProbe.capability="";accountProbe.response="";}
    function selectSession(id){selectedSessionId.value=id;screenshotTick.value++;}
    function openSessionModal(id){if(id){selectedSessionId.value=id;}screenshotTick.value++;scanModal.value=true;}
    function openNoVNC(url){if(url) window.open(url,"_blank","noopener");}
    async function startMaintenance(id){
      try{
        const data=await api("/accounts/"+encodeURIComponent(id)+"/maintenance/start",{method:"POST",body:"{}"});
        await loadSessions();
        if(data.data && data.data.id){openSessionModal(data.data.id);openNoVNC(data.data.novnc_url);}
        showToast("账号检修浏览器已启动。");
      }catch(err){showToast(err.message);}
    }
    function closeScanModal(){scanModal.value=false;}
    function openLatestSessionModal(){if(sessions.value.length){openSessionModal(sessions.value[0].id);showToast("已打开最新扫码会话。");}else{showToast("暂无扫码会话，请先新增账号。");}}
    function openAdd(){newAccount.name="";addModal.value=true;setTimeout(function(){const input=document.querySelector(".modal input");if(input) input.focus();},80);}
    function closeAdd(){addModal.value=false;newAccount.name="";}
    async function createAccount(){
      const name=(newAccount.name || "").trim();
      if(!name){showToast("请先填写账号名称。");return;}
      try{
        const data=await api("/accounts",{method:"POST",body:JSON.stringify({name:name})});
        closeAdd();
        await refreshAll();
        if(data.data && data.data.id){openSessionModal(data.data.id);}
        showToast("扫码会话已创建，请在弹窗中完成扫码。");
      }catch(err){showToast(err.message);}
    }
    async function testAccount(id, capability){
      capability=capability || "chat";
      accountProbe.loading=true;accountProbe.status="";accountProbe.message="";accountProbe.capability=capability;accountProbe.response="";
      try{
        const result=await api("/accounts/"+encodeURIComponent(id)+"/test",{method:"POST",body:JSON.stringify({capability:capability})});
        accountProbe.status=result.status || (result.ok ? "valid" : "invalid");
        accountProbe.message=result.message || "账号测活完成。";
        accountProbe.response=result.response_text || "";
        showToast(accountProbe.message);
      }catch(err){accountProbe.status="error";accountProbe.message=err.message;showToast(err.message);}
      finally{accountProbe.loading=false;await refreshAll();}
    }
    async function syncQuota(id){
      accountProbe.loading=true;accountProbe.status="";accountProbe.message="";
      try{
        const data=await api("/accounts/"+encodeURIComponent(id)+"/quota/sync",{method:"POST"});
        accountProbe.status=data.ok ? "valid" : "unknown";
        accountProbe.message=data.message || "额度同步完成。";
        showToast(accountProbe.message);
      }catch(err){accountProbe.status="error";accountProbe.message=err.message;showToast(err.message);}
      finally{accountProbe.loading=false;await refreshAll();}
    }
    async function deleteAccount(id){
      const account=accounts.value.find(function(item){return item.id===id;});
      const label=account ? account.name+" / "+account.id : id;
      const ok=window.confirm("确认删除账号 "+label+"？\n\n这会删除 SQLite 账号记录、账号事件，解绑历史任务归属，并关闭该账号关联的扫码会话。\n\n该操作不可恢复。");
      if(!ok) return;
      try{
        const result=await api("/accounts/"+encodeURIComponent(id),{method:"DELETE"});
        selectedId.value="";
        await refreshAll();
        const data=result.data || {};
        showToast("账号已删除，事件 "+(data.account_events_deleted || 0)+" 条，解绑任务 "+(data.tasks_detached || 0)+" 条。");
      }catch(err){showToast(err.message);}
    }
    async function clickLoginEntry(id){try{await api("/login-sessions/"+encodeURIComponent(id)+"/click-login",{method:"POST"});screenshotTick.value++;await loadSessions();showToast("已点击登录入口。");}catch(err){showToast(err.message);}}
    async function refreshSession(id){try{await api("/login-sessions/"+encodeURIComponent(id)+"/refresh",{method:"POST"});screenshotTick.value++;await loadSessions();showToast("二维码会话已刷新。");}catch(err){showToast(err.message);}}
    async function captureSession(id){
      try{
        const result=await api("/login-sessions/"+encodeURIComponent(id)+"/capture",{method:"POST"});
        await refreshAll();
        if(result.data && result.data.id){selectedId.value=result.data.id;}
        scanModal.value=false;
        showToast("账号已捕获，请继续测活。");
      }catch(err){showToast(err.message);}
    }
    async function deleteSession(id){
      const session=sessions.value.find(function(item){return item.id===id;});
      const label=session ? session.name+" / "+session.id : id;
      const ok=window.confirm("确认删除扫码会话 "+label+"？\n\n这会关闭 Chromium 进程并删除临时浏览器 profile。");
      if(!ok) return;
      try{await api("/login-sessions/"+encodeURIComponent(id),{method:"DELETE"});if(selectedSessionId.value===id){selectedSessionId.value="";scanModal.value=false;}await loadSessions();showToast("扫码会话已删除。");}catch(err){showToast(err.message);}
    }
    async function runTest(){
      test.loading=true;test.output="";test.error="";
      try{
        const kind=modelKind(test.model);
        let data;
        if(kind==="生图"){
          data=await callProvider("/v1/images/generations",{model:test.model,prompt:test.prompt,account_id:test.account_id || undefined,n:1});
        }else if(kind==="生视频"){
          data=await callProvider("/v1/videos",{model:test.model,prompt:test.prompt,account_id:test.account_id || undefined,duration:test.duration,ratio:test.ratio,wait:false});
        }else{
          data=await callProvider("/v1/chat/completions",{model:test.model,account_id:test.account_id || undefined,messages:[{role:"user",content:test.prompt}],stream:false});
        }
        test.output=JSON.stringify(data,null,2);
        await Promise.all([loadAccounts(),loadTasks(),loadLogs()]);
      }catch(err){test.error=err.message;await loadLogs();}
      finally{test.loading=false;}
    }
    async function copy(text){if(!text)return;try{await navigator.clipboard.writeText(text);showToast("已复制到剪切板。");}catch(err){const ta=document.createElement("textarea");ta.value=text;document.body.appendChild(ta);ta.select();document.execCommand("copy");ta.remove();showToast("已复制到剪切板。");}}
    function screenshotUrl(id){screenshotTick.value;return "/api/login-sessions/"+encodeURIComponent(id)+"/screenshot?key="+encodeURIComponent(adminKey)+"&t="+Date.now()+"-"+screenshotTick.value;}
    function capabilityList(account){try{const caps=JSON.parse(account.capabilities_json || "{}");return Object.keys(caps).filter(function(key){return !!caps[key];});}catch(err){return [];}}
    function formatCaps(value){try{const caps=JSON.parse(value || "{}");const keys=Object.keys(caps).filter(function(key){return !!caps[key];});return keys.length ? keys.map(capText).join(" / ") : "{}";}catch(err){return value || "-";}}
    function modelKind(modelID){const found=models.value.find(function(model){return model.id===modelID;});if(found && found.type==="image") return "生图";if(found && found.type==="video") return "生视频";const lower=String(modelID || "").toLowerCase();if(lower.indexOf("image")>=0) return "生图";if(lower.indexOf("happyhorse")>=0 || lower.indexOf("video")>=0) return "生视频";return "对话";}
    function statusClass(value){const v=value || "unknown";if(["valid","succeeded","captured","login_detected"].indexOf(v)>=0) return v;if(["invalid","failed","capture_failed","expired","error"].indexOf(v)>=0) return v;return v;}
    function statusText(value){const map={valid:"可用",unknown:"未知",invalid:"无效",succeeded:"成功",failed:"失败",captured:"已捕获",login_detected:"检测到登录",capture_failed:"捕获失败",expired:"已过期",starting:"启动中",opening:"打开中",waiting_scan:"等待扫码",processing:"处理中",queued:"排队中"};return map[value] || value || "未知";}
    function typeText(value){const map={qianwen_qr:"扫码登录",login_cookie:"扫码捕获",guest:"游客池"};return map[value] || value || "未知";}
    function capText(value){const map={chat:"对话",image:"生图",video:"生视频"};return map[value] || value;}
    function breakdown(obj){const keys=Object.keys(obj || {});if(!keys.length) return "暂无";return keys.map(function(key){return key+":"+obj[key];}).join(" / ");}
    function initial(name){return String(name || "Q").slice(0,1).toUpperCase();}
    function isProtectedAccount(id){return id==="default" || id==="guest";}
    function fmtClock(ts){return ts ? new Date(ts*1000).toLocaleTimeString("zh-CN",{hour12:false}) : "-";}
    onMounted(function(){refreshAll();pollTimer=setInterval(function(){if(tab.value==="logs"){loadLogs().catch(function(){});} if(tab.value==="accounts" || scanModal.value){loadSessions().catch(function(){});}},5000);});
    onBeforeUnmount(function(){if(pollTimer) clearInterval(pollTimer);if(toast.timer) clearTimeout(toast.timer);});
    return{tabs,tab,title,busy,accounts,sessions,tasks,models,logs,summary,serviceKey,apiBase,selectedId,selectedSessionId,selectedAccount,selectedSession,validCount,activeSessionCount,taskBreakdown,defaultChatModel,addModal,scanModal,newAccount,accountProbe,test,toast,adminKey,locationOrigin,refreshAll,loadAccounts,loadSessions,selectAccount,selectSession,openSessionModal,openNoVNC,startMaintenance,closeScanModal,openLatestSessionModal,openAdd,closeAdd,createAccount,testAccount,syncQuota,deleteAccount,clickLoginEntry,refreshSession,captureSession,deleteSession,runTest,copy,screenshotUrl,capabilityList,formatCaps,modelKind,statusClass,statusText,typeText,capText,initial,isProtectedAccount,fmtClock,saveServiceKey};
  }
}).mount("#app");
</script>
</body>
</html>`
