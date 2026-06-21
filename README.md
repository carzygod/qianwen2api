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
| Image-to-video material handling | qianwen.com fallback uses official `attachments[].materialId`; generated images return `metadata.qianwen_material_id` for reuse |

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

When the qianwen.com fallback route is used, image-to-video requests must reference a Qianwen material id. The service records this automatically when an image is generated through `/v1/images/generations`.

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

Use the returned `url` in `image_url`, or pass `metadata.qianwen_material_id` / `metadata.qwen_resource` explicitly. A bare external image URL that cannot be mapped to a Qianwen material id is rejected instead of being treated as an uploaded attachment.

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
