# QIANWEN-WEB-01 / qianwen2api

QIANWEN-WEB-01 is the maintained qianwen.com Web reverse-proxy provider used by
the gen2api stack.

Repository:

```text
https://github.com/carzygod/qianwen2api
```

Original upstream base:

```text
https://github.com/kao0312/qianwen2api
```

This service does not use official Qwen API keys. It drives logged-in qianwen.com
Web sessions captured by QR login and exposes OpenAI-compatible APIs for chat,
image generation, and video generation.

## Capabilities

| Area | Status |
|---|---|
| Storage | SQLite |
| Redis | Not used |
| Admin WebUI | `/admin?key=<ADMIN_KEY>` |
| Account import | QR login with server-side Chromium |
| QR lifecycle | Create, refresh, confirm, delete; delete closes Chromium profile |
| Account pool | Multiple qianwen.com Web accounts |
| Account test | Sends a real default chat request and requires model output |
| Chat API | OpenAI-compatible `/v1/chat/completions` |
| Image API | OpenAI-compatible `/v1/images/generations` |
| Video API | OpenAI-compatible `/v1/videos` plus legacy aliases |
| Video polling | `/v1/videos/{task_id}` plus legacy aliases |
| Video cancel | Local cancel on `/v1/videos/{task_id}/cancel` plus legacy aliases |
| NewAPI use | Can be added as an OpenAI-compatible channel for chat/image/video |

## Models

| Capability | Model |
|---|---|
| Chat | `tongyi-qwen3-max-model` |
| Chat | `tongyi-qwen3-max-thinking` |
| Image | `Qwen-Image-2.0` |
| Video | `HappyHorse 1.0` |

The public video model is `HappyHorse 1.0`. The captured qianwen.com upstream
provider model/root model is `happyhorse`.

## Quick Start

Local:

```bash
cp .env.example .env
go run main.go
```

Docker:

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

Open the Admin WebUI:

```text
http://127.0.0.1:18002/admin?key=change-me-admin-key
```

## Environment

| Variable | Default | Description |
|---|---|---|
| `HOST` | `0.0.0.0` | Listen host |
| `PORT` | `8080` locally, `8000` in Docker | Listen port |
| `AUTH_KEY` | empty | Bearer token for `/v1/*` APIs |
| `ADMIN_KEY` | `AUTH_KEY` | Admin WebUI/API key |
| `POOL_SIZE` | `0` | Guest fallback pool size; keep `0` for QR-account deployments |
| `REFRESH_HOURS` | `10` | Guest UMID refresh period when guest pool is enabled |
| `DATA_DIR` | `./data` | Data directory |
| `DATABASE_PATH` | `./data/qianwen-web-01.sqlite` | SQLite database path |
| `PUBLIC_BASE_URL` | empty | Public base URL shown in Admin summary |
| `DEFAULT_CHAT_MODEL` | `tongyi-qwen3-max-model` | Default chat model |
| `DEFAULT_IMAGE_MODEL` | `Qwen-Image-2.0` | Default image model |
| `DEFAULT_VIDEO_MODEL` | `HappyHorse 1.0` | Default video model |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Add Accounts

Accounts are added through the Admin WebUI.

1. Open `/admin?key=<ADMIN_KEY>`.
2. Click `Add account`.
3. Enter a readable account name.
4. Click `Generate QR`.
5. Scan the QR code shown in the screenshot.
6. Wait until the screenshot shows a logged-in qianwen.com page.
7. Click `Confirm scan`.
8. Run `Test` on the saved account.

The account should be routed only after the test succeeds. The test sends a real
chat request to qianwen.com and requires a non-empty assistant response.

## Public APIs

All `/v1/*` APIs use:

```text
Authorization: Bearer <AUTH_KEY>
```

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Runtime health |
| `GET` | `/v1/models` | Model list |
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat |
| `POST` | `/v1/images/generations` | OpenAI-compatible image generation |
| `POST` | `/v1/videos` | OpenAI-compatible async video creation |
| `GET` | `/v1/videos/{task_id}` | OpenAI-compatible video polling |
| `POST` | `/v1/videos/{task_id}/cancel` | OpenAI-compatible local cancel |
| `POST` | `/v1/video/generations` | Legacy async video creation |
| `GET` | `/v1/video/generations/{task_id}` | Legacy video polling |
| `POST` | `/v1/video/generations/{task_id}/cancel` | Legacy local cancel |
| `POST` | `/v1/videos/generations` | Legacy plural alias |
| `GET` | `/v1/videos/generations/{task_id}` | Legacy plural polling |
| `POST` | `/v1/videos/generations/{task_id}/cancel` | Legacy plural cancel |
| `POST` | `/v1/video/generations/sync` | Blocking video generation helper |
| `GET` | `/v1/tasks/{task_id}` | Raw task record |

## Admin APIs

Admin requests accept `X-Admin-Key: <ADMIN_KEY>` or `/admin?key=<ADMIN_KEY>`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/summary` | Runtime summary |
| `GET` | `/api/accounts` | Account list |
| `POST` | `/api/accounts` | Start a QR login session for a new account |
| `DELETE` | `/api/accounts/{id}` | Delete account |
| `POST` | `/api/accounts/{id}/test` | Real upstream account test |
| `POST` | `/api/accounts/{id}/quota/sync` | Reserved quota sync endpoint |
| `GET` | `/api/login-sessions` | QR login session list |
| `POST` | `/api/login-sessions` | Start QR login session |
| `DELETE` | `/api/login-sessions/{id}` | Delete QR session and close Chromium |
| `GET` | `/api/login-sessions/{id}/screenshot` | Login screenshot / QR image |
| `POST` | `/api/login-sessions/{id}/refresh` | Restart QR login session |
| `POST` | `/api/login-sessions/{id}/capture` | Capture logged-in cookies into account pool |
| `GET` | `/api/tasks` | Recent tasks |
| `GET` | `/api/models` | SQLite model registry |

## Examples

Chat:

```bash
curl http://127.0.0.1:18002/v1/chat/completions \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tongyi-qwen3-max-model",
    "messages": [{"role": "user", "content": "Reply with OK only."}],
    "stream": false
  }'
```

Image:

```bash
curl http://127.0.0.1:18002/v1/images/generations \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen-Image-2.0",
    "prompt": "a white cube on a desk, realistic photo style",
    "n": 1,
    "size": "1:1"
  }'
```

OpenAI-compatible video:

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

Poll:

```bash
curl http://127.0.0.1:18002/v1/videos/<task_id> \
  -H "Authorization: Bearer change-me-api-key"
```
