# QIANWEN-WEB-01 / qianwen2api

[English](#english) | [中文](#中文) | [Русский](#русский)

## English

QIANWEN-WEB-01 is the maintained qianwen.com Web reverse-proxy provider used by gen2api.

Maintained repository: https://github.com/carzygod/qianwen2api

Original upstream base: https://github.com/kao0312/qianwen2api

This project does not use official Qwen API keys. It captures logged-in qianwen.com browser sessions through server-side Chromium QR login, stores accounts in SQLite, and exposes OpenAI-compatible APIs for chat, image generation, and async video generation.

### Capabilities

| Area | Status |
|---|---|
| Storage | SQLite |
| Redis | Not used |
| Admin WebUI | `/admin?key=<ADMIN_KEY>` |
| Admin workflow | Doubao-style Chinese account pool with add-account modal, QR scan modal, account test, test bench, and request logs |
| Account import | QR login with server-side Chromium |
| QR lifecycle | Create, refresh, confirm, delete; delete closes Chromium |
| Account pool | Multiple qianwen.com Web accounts |
| Account delete | Removes the SQLite account row, account events, detaches historical task ownership, and closes captured QR sessions |
| Account test | Sends a real default chat request and requires model output |
| Chat API | `/v1/chat/completions` |
| Image API | `/v1/images/generations` |
| Video API | `/v1/videos`, `/v1/video/generations`, `/v1/videos/generations` |
| Video polling | `/v1/videos/{task_id}` plus legacy aliases |
| Video cancel | `/v1/videos/{task_id}/cancel` plus legacy aliases |
| NewAPI use | Can be added as an OpenAI-compatible channel for chat/image/video |
| Image-to-video material handling | qianwen.com fallback uses official `attachments[].materialId`; existing Qianwen material ids are reused, and public URLs / data URI / base64 images are uploaded through the qianwen.com Web runtime before submission |

### Models

| Capability | Model |
|---|---|
| Chat | `tongyi-qwen3-max-model` |
| Chat | `tongyi-qwen3-max-thinking` |
| Image | `Qwen-Image-2.0` |
| Video | `HappyHorse 1.0` |

The public video model is `HappyHorse 1.0`. The observed upstream qianwen.com provider/root model is `happyhorse`.

### Quick Start

```bash
docker build -t qianwen-web-01:latest .
docker run -d --name qianwen-web-01 \
  -p 18002:8000 \
  -e HOST=0.0.0.0 \
  -e PORT=8000 \
  -e AUTH_KEY=change-me-api-key \
  -e ADMIN_KEY=change-me-admin-key \
  -e POOL_SIZE=0 \
  -e DATA_DIR=/app/data \
  -e DATABASE_PATH=/app/data/qianwen-web-01.sqlite \
  -v "$PWD/data:/app/data" \
  qianwen-web-01:latest
```

## 账户检修（私有 noVNC）

`Dockerfile.novnc` 在原有服务结构外增加 Xvfb、x11vnc 和 noVNC。每个账户使用
`DATA_DIR/account-chrome-profiles/<account-id>` 独立 profile；检修租约活动时该账户
不会参与正常调度。容器环境需设置 `VNC_PASSWORD`，并将容器 `6080` 端口只映射到
宿主机 `127.0.0.1`。`NOVNC_URL` 用于后台和聚合层返回操作入口，noVNC 模式会强制
`BROWSER_HEADLESS=false`。

维护接口位于 `/api/accounts/{id}/maintenance`，支持 `start`、`heartbeat`、`stop`、
`validate`；登录状态通过 `/api/login-sessions/{lease-owner}/capture` 捕获。捕获后应先
结束检修，再执行账号测活。不要自动删除 Chromium `Singleton*` 锁文件。

Open `http://127.0.0.1:18002/admin?key=change-me-admin-key`.

### Add Accounts

1. Click `新增账号`.
2. Enter a readable account name.
3. Click `生成二维码`.
4. Scan the QR code shown in the live screenshot.
5. Wait until qianwen.com is logged in.
6. Click `确认扫码`.
7. Run account `测活`; only accounts with real model output should receive traffic.

### API Example

```bash
curl http://127.0.0.1:18002/v1/videos \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "HappyHorse 1.0",
    "prompt": "a white cube slowly rotating on a desk, realistic photo style",
    "duration": 5,
    "resolution": "720P",
    "ratio": "16:9"
  }'
```

### Image-To-Video Notes

When the qianwen.com fallback route is used, image-to-video requests are converted to the official qianwen.com attachment shape:

```json
{
  "attachments": [
    {"type": "image", "materialId": "..."}
  ]
}
```

The resolver uses this order:

1. Reuse explicit `metadata.qianwen_material_id` / `metadata.qwen_resource`.
2. Reuse a stored SQLite asset mapping from a previous `/v1/images/generations` result.
3. Upload public URL, data URI, or raw base64 image input with the qianwen.com Web runtime and use the returned material id.

Image generation response example:

```json
{
  "data": [
    {
      "url": "https://workspace-zb-cdn.qianwen.com/...png",
      "metadata": {
        "qianwen_material_id": "6ebee98cea4b4ca5aa5440a233e244ac",
        "qwen_resource": {
          "id": "6ebee98cea4b4ca5aa5440a233e244ac",
          "url": "https://workspace-zb-cdn.qianwen.com/...png",
          "width": 1024,
          "height": 576
        }
      }
    }
  ]
}
```

Use the returned `url` in `image_url`, or pass `metadata.qianwen_material_id` / `metadata.qwen_resource` explicitly. Public URL, data URI, and raw base64 image inputs are also accepted; they are uploaded to qianwen.com and then submitted as `attachments[].materialId`.

Two-image video request example:

```json
{
  "model": "HappyHorse 1.0",
  "prompt": "Use the first image as the opening frame and the second image as the visual target.",
  "first_frame_image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "reference_images": [
    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
  ],
  "duration": 5,
  "resolution": "720P",
  "aspect_ratio": "16:9"
}
```

### NewAPI Validation

QIANWEN-WEB-01 can be configured in NewAPI as an OpenAI-compatible channel with `HappyHorse 1.0` in the channel model list. The tested NewAPI route is:

```text
POST /v1/video/generations
GET  /v1/video/generations/{task_id}
```

Validation performed on 2026-06-21:

| Item | Result |
|---|---|
| NewAPI task | `task_GW8rRGXBlR1xNW8tNhcBpUBy22BaD2X9` |
| QIANWEN-WEB-01 task | `4fa38a69-326d-4abc-aa35-c45ca85f6067` |
| Model | `HappyHorse 1.0` |
| Input | `first_frame_image` plus one `reference_images` item, both as data URI PNG |
| Parameters | `duration=5`, `resolution=720P`, `aspect_ratio=16:9` |
| Upstream payload | 2 official `attachments[].materialId` values |
| Status | NewAPI `SUCCESS`, QIANWEN-WEB-01 `succeeded` |
| Output | 2 reachable `video/mp4` URLs |

Known NewAPI caveat: task polling returns the video URLs correctly. The optional content proxy endpoint `/v1/videos/{task_id}/content` depends on NewAPI fetch/SSRF settings and may need its allowed ports/domains adjusted before it can proxy the video bytes itself.

## 中文

QIANWEN-WEB-01 是 gen2api 使用的 qianwen.com / 通义千问 Web 反代维护版。

维护仓库： https://github.com/carzygod/qianwen2api

原始上游基底： https://github.com/kao0312/qianwen2api

本项目不使用官方 Qwen API Key，而是通过服务端 Chromium 生成二维码，扫码后捕获 qianwen.com 登录会话，将账号保存到 SQLite 账号池，并对外提供对话、生图、异步生视频接口。

### 能力

| 模块 | 状态 |
|---|---|
| 存储 | SQLite |
| Redis | 不使用 |
| Admin WebUI | `/admin?key=<ADMIN_KEY>` |
| 后台交互 | 对齐豆包风格的中文账号池：新增账号、扫码弹窗、测活、接口测试、请求日志 |
| 账号导入 | 服务端 Chromium 二维码扫码登录 |
| 二维码生命周期 | 创建、刷新、确认、删除；删除会关闭 Chromium |
| 账号池 | 多个 qianwen.com Web 登录账号 |
| 账号测试 | 发送真实默认对话模型请求，拿到模型返回才算成功 |
| 对话接口 | `/v1/chat/completions` |
| 生图接口 | `/v1/images/generations` |
| 生视频接口 | `/v1/videos`、`/v1/video/generations`、`/v1/videos/generations` |
| 视频轮询 | `/v1/videos/{task_id}` 及旧别名 |
| 视频取消 | `/v1/videos/{task_id}/cancel` 及旧别名 |
| NewAPI | 可作为 OpenAI 兼容渠道接入对话 / 图片 / 视频 |
| 图文生视频素材处理 | qianwen.com fallback 使用官方 `attachments[].materialId`；已存在的素材 ID 会复用，公网 URL / data URI / base64 图片会通过 qianwen.com Web 运行时自动上传后再提交 |

### 模型

| 能力 | 模型 |
|---|---|
| 对话 | `tongyi-qwen3-max-model` |
| 对话 | `tongyi-qwen3-max-thinking` |
| 生图 | `Qwen-Image-2.0` |
| 生视频 | `HappyHorse 1.0` |

对外视频模型名为 `HappyHorse 1.0`。网页侧实际观测到的 provider/root model 是 `happyhorse`。

### 快速启动

```bash
docker build -t qianwen-web-01:latest .
docker run -d --name qianwen-web-01 \
  -p 18002:8000 \
  -e HOST=0.0.0.0 \
  -e PORT=8000 \
  -e AUTH_KEY=change-me-api-key \
  -e ADMIN_KEY=change-me-admin-key \
  -e POOL_SIZE=0 \
  -e DATA_DIR=/app/data \
  -e DATABASE_PATH=/app/data/qianwen-web-01.sqlite \
  -v "$PWD/data:/app/data" \
  qianwen-web-01:latest
```

打开 `http://127.0.0.1:18002/admin?key=change-me-admin-key`。

### 新增账号

1. 点击 `新增账号`。
2. 填写账号名称。
3. 点击 `生成二维码`。
4. 扫描实时截图中的二维码。
5. 等待页面进入 qianwen.com 登录后状态。
6. 点击 `确认扫码` 保存账号。
7. 点击账号 `测活`，只有真实模型请求返回成功的账号才进入可用路由。

### 调用示例

```bash
curl http://127.0.0.1:18002/v1/videos \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "HappyHorse 1.0",
    "prompt": "一个白色立方体在桌面上缓慢旋转，真实摄影风格",
    "duration": 5,
    "resolution": "720P",
    "ratio": "16:9"
  }'
```

### 图文生视频说明

qianwen.com fallback 路线会把图文生视频输入转换成官方附件结构：

```json
{
  "attachments": [
    {"type": "image", "materialId": "..."}
  ]
}
```

素材解析顺序：

1. 优先复用显式传入的 `metadata.qianwen_material_id` / `metadata.qwen_resource`。
2. 如果是本服务生图返回的 URL，则从 SQLite 资产表复用历史素材映射。
3. 如果是公网 URL、data URI 或裸 base64 图片，则通过 qianwen.com Web 前端运行时上传成官方素材，再写入 `attachments[].materialId`。

两图生视频示例：

```json
{
  "model": "HappyHorse 1.0",
  "prompt": "以第一张图作为开场画面，第二张图作为视觉目标，生成 5 秒平滑转场视频",
  "first_frame_image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "reference_images": [
    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
  ],
  "duration": 5,
  "resolution": "720P",
  "aspect_ratio": "16:9"
}
```

### NewAPI 验证

QIANWEN-WEB-01 可以在 NewAPI 中配置为 OpenAI 兼容渠道，渠道模型列表中包含 `HappyHorse 1.0`。已验证的 NewAPI 路由：

```text
POST /v1/video/generations
GET  /v1/video/generations/{task_id}
```

2026-06-21 验证结果：

| 项 | 结果 |
|---|---|
| NewAPI 任务 | `task_GW8rRGXBlR1xNW8tNhcBpUBy22BaD2X9` |
| QIANWEN-WEB-01 上游任务 | `4fa38a69-326d-4abc-aa35-c45ca85f6067` |
| 模型 | `HappyHorse 1.0` |
| 输入 | `first_frame_image` + 1 张 `reference_images`，均为 data URI PNG |
| 参数 | `duration=5`、`resolution=720P`、`aspect_ratio=16:9` |
| 上游 payload | 2 个官方 `attachments[].materialId` |
| 状态 | NewAPI `SUCCESS`，QIANWEN-WEB-01 `succeeded` |
| 输出 | 2 个可访问的 `video/mp4` URL |

已知边界：NewAPI 任务轮询可以正确返回视频 URL。可选的 `/v1/videos/{task_id}/content` 内容代理受 NewAPI fetch/SSRF 白名单影响，若要由 NewAPI 代理视频字节，需要额外放开对应的代理目标端口或域名。

## Русский

QIANWEN-WEB-01 — поддерживаемый Web reverse-proxy для qianwen.com, используемый в gen2api.

Поддерживаемый репозиторий: https://github.com/carzygod/qianwen2api

Исходная база upstream: https://github.com/kao0312/qianwen2api

Проект не использует официальные Qwen API keys. Он создает QR login через server-side Chromium, сохраняет qianwen.com Web-сессии в SQLite и предоставляет OpenAI-compatible API для chat, image и async video generation.

### Возможности

| Area | Status |
|---|---|
| Storage | SQLite |
| Redis | Не используется |
| Admin WebUI | `/admin?key=<ADMIN_KEY>` |
| Admin workflow | Doubao-style Chinese UI: account pool, QR modal, account test, API test bench, request logs |
| Account import | QR login через server-side Chromium |
| QR lifecycle | create, refresh, confirm, delete; delete закрывает Chromium |
| Account pool | Несколько qianwen.com Web аккаунтов |
| Account delete | Removes the SQLite account row, account events, detaches historical task ownership, and closes captured QR sessions |
| Account test | Реальный default chat request с проверкой ответа модели |
| Chat API | `/v1/chat/completions` |
| Image API | `/v1/images/generations` |
| Video API | `/v1/videos`, `/v1/video/generations`, `/v1/videos/generations` |
| Video polling | `/v1/videos/{task_id}` и legacy aliases |
| Video cancel | `/v1/videos/{task_id}/cancel` и legacy aliases |
| NewAPI use | Можно добавить как OpenAI-compatible канал |
| Image-to-video material handling | qianwen.com fallback submits official `attachments[].materialId`; public URLs, data URI, and base64 images are uploaded through the qianwen.com Web runtime |

### Модели

| Capability | Model |
|---|---|
| Chat | `tongyi-qwen3-max-model` |
| Chat | `tongyi-qwen3-max-thinking` |
| Image | `Qwen-Image-2.0` |
| Video | `HappyHorse 1.0` |

Публичное имя video-модели: `HappyHorse 1.0`. Наблюдаемое upstream provider/root model: `happyhorse`.

### Быстрый старт

```bash
docker build -t qianwen-web-01:latest .
docker run -d --name qianwen-web-01 \
  -p 18002:8000 \
  -e HOST=0.0.0.0 \
  -e PORT=8000 \
  -e AUTH_KEY=change-me-api-key \
  -e ADMIN_KEY=change-me-admin-key \
  -e POOL_SIZE=0 \
  -e DATA_DIR=/app/data \
  -e DATABASE_PATH=/app/data/qianwen-web-01.sqlite \
  -v "$PWD/data:/app/data" \
  qianwen-web-01:latest
```

Откройте `http://127.0.0.1:18002/admin?key=change-me-admin-key`.

### Добавление аккаунтов

1. Нажмите `新增账号`.
2. Введите понятное имя аккаунта.
3. Нажмите `生成二维码`.
4. Отсканируйте QR на live screenshot.
5. Дождитесь входа в qianwen.com.
6. Нажмите `确认扫码`.
7. Запустите `测活`; только аккаунты с реальным ответом модели должны получать трафик.

### Пример API

```bash
curl http://127.0.0.1:18002/v1/videos \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "HappyHorse 1.0",
    "prompt": "a white cube slowly rotating on a desk, realistic photo style",
    "duration": 5,
    "resolution": "720P",
    "ratio": "16:9"
  }'
```
