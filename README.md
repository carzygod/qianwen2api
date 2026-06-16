# QIANWEN-WEB-01 / qianwen2api

QIANWEN-WEB-01 is a maintained qianwen.com Web reverse-proxy fork for the gen2api stack.

Fork:

- Current repository: `https://github.com/carzygod/qianwen2api`
- Original upstream base: `https://github.com/kao0312/qianwen2api`
- Storage: SQLite only
- Redis: not used
- Upstream account type: qianwen.com logged-in Web session captured by QR login

This project does not use official Qwen API keys. It replays qianwen.com Web-session traffic from logged-in accounts that you add through the built-in Admin WebUI.

## Current Capabilities

| Area | Status |
|---|---|
| Admin WebUI | Implemented at `/admin?key=<ADMIN_KEY>` |
| QR login account pool | Implemented with server-side Chromium sessions |
| QR refresh/delete | Implemented; delete closes Chromium and removes the temporary profile |
| SQLite account storage | Implemented |
| Real account test | Implemented; sends a real chat request and requires non-empty output |
| OpenAI-compatible chat | Implemented via `/v1/chat/completions` |
| OpenAI-compatible image generation | Implemented via `/v1/images/generations` |
| OpenAI-compatible video generation | Implemented via `/v1/video/generations` and `/v1/videos/generations` |
| Video task polling | Implemented via `/v1/video/generations/{task_id}`, `/v1/videos/generations/{task_id}`, and `/v1/tasks/{task_id}` |
| Guest pool | Optional fallback only; recommended `POOL_SIZE=0` |

## Models

The default model registry contains concrete model names only:

| Type | Model |
|---|---|
| Chat | `tongyi-qwen3-max-model` |
| Chat | `tongyi-qwen3-max-thinking` |
| Image | `Qwen-Image-2.0` |
| Video | `Wan2.2` |

The video Web payload currently maps the qianwen.com video route to the observed `HappyHorse 1.0` upstream route while exposing `Wan2.2` as the OpenAI-compatible model id.

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

Open Admin:

```text
http://127.0.0.1:18002/admin?key=change-me-admin-key
```

## Environment

| Variable | Default | Description |
|---|---:|---|
| `HOST` | `0.0.0.0` | Listen host |
| `PORT` | `8080` locally, `8000` in Docker | Listen port |
| `AUTH_KEY` | empty | Bearer token for `/v1/*` APIs |
| `ADMIN_KEY` | `AUTH_KEY` | Admin WebUI/API key |
| `POOL_SIZE` | `0` | Guest chat fallback pool size; keep `0` for QR login account-pool deployments |
| `REFRESH_HOURS` | `10` | Guest UMID refresh period when guest pool is enabled |
| `DATA_DIR` | `./data` | Data directory |
| `DATABASE_PATH` | `./data/qianwen-web-01.sqlite` | SQLite database path |
| `PUBLIC_BASE_URL` | empty | Public base URL exposed in Admin summary |
| `DEFAULT_CHAT_MODEL` | `tongyi-qwen3-max-model` | Default chat model |
| `DEFAULT_IMAGE_MODEL` | `Qwen-Image-2.0` | Default image model |
| `DEFAULT_VIDEO_MODEL` | `Wan2.2` | Default video model |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Add Accounts

Accounts must be added in QIANWEN-WEB-01 Admin WebUI.

1. Open `/admin?key=<ADMIN_KEY>`.
2. Click `Add account`.
3. Enter a readable account name.
4. Click `Generate QR`.
5. Scan the QR code shown in the screenshot.
6. Wait until the screenshot shows a logged-in qianwen.com page.
7. Click `Confirm scan`.
8. Click `Test` on the saved account.

The account is safe to route only after the Admin test succeeds. The test calls a real qianwen.com default chat model and requires a non-empty assistant response.

QR sessions expire quickly. Use `Refresh QR` to create a new login browser session, or `Delete` to close Chromium and remove the temporary profile.

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
| `POST` | `/v1/images/generations` | Image generation; returns image URLs |
| `POST` | `/v1/video/generations` | Async video task creation |
| `GET` | `/v1/video/generations/{task_id}` | Video task polling |
| `POST` | `/v1/videos/generations` | Compatibility alias |
| `GET` | `/v1/videos/generations/{task_id}` | Compatibility alias polling |
| `GET` | `/v1/tasks/{task_id}` | Raw task record |

## Admin APIs

Admin requests accept either `X-Admin-Key: <ADMIN_KEY>` or `/admin?key=<ADMIN_KEY>`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/summary` | Runtime summary |
| `GET` | `/api/accounts` | Account list |
| `POST` | `/api/accounts` | Start a QR login session for a new account |
| `GET` | `/api/accounts/{id}` | Account detail |
| `DELETE` | `/api/accounts/{id}` | Delete account |
| `POST` | `/api/accounts/{id}/test` | Real upstream account test |
| `POST` | `/api/accounts/{id}/quota/sync` | Reserved; quota sync endpoint still needs qianwen.com quota protocol capture |
| `GET` | `/api/login-sessions` | QR login session list |
| `POST` | `/api/login-sessions` | Start QR login session |
| `GET` | `/api/login-sessions/{id}` | QR login session detail |
| `DELETE` | `/api/login-sessions/{id}` | Delete QR session and close Chromium |
| `GET` | `/api/login-sessions/{id}/screenshot` | Login screenshot / QR image |
| `POST` | `/api/login-sessions/{id}/refresh` | Restart QR login session |
| `POST` | `/api/login-sessions/{id}/click-login` | Try clicking qianwen.com login entry |
| `POST` | `/api/login-sessions/{id}/capture` | Capture logged-in browser cookies into account pool |
| `GET` | `/api/tasks` | Recent tasks |
| `GET` | `/api/tasks/{id}` | Task detail |
| `GET` | `/api/models` | SQLite model registry |

## Chat Example

```bash
curl http://127.0.0.1:18002/v1/chat/completions \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tongyi-qwen3-max-model",
    "messages": [
      {"role": "user", "content": "你好，只回复 ok"}
    ],
    "stream": false
  }'
```

## Image Example

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

Successful responses follow the OpenAI image shape:

```json
{
  "created": 1780000000,
  "data": [
    { "url": "https://workspace-zb-cdn.qianwen.com/..." }
  ]
}
```

## Video Example

Create a task:

```bash
curl http://127.0.0.1:18002/v1/video/generations \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Wan2.2",
    "prompt": "a white cube slowly rotating on a desk, realistic photo style, five seconds",
    "duration": 5,
    "resolution": "720P",
    "aspect_ratio": "16:9"
  }'
```

Poll:

```bash
curl http://127.0.0.1:18002/v1/video/generations/<task_id> \
  -H "Authorization: Bearer change-me-api-key"
```

Completed tasks return generated media URLs in `data.urls`.

## NewAPI Integration

Use this service as a custom OpenAI-compatible provider:

| NewAPI Field | Value |
|---|---|
| Base URL | `http://<host>:18002/v1` |
| Key | `<AUTH_KEY>` |
| Chat models | `tongyi-qwen3-max-model`, `tongyi-qwen3-max-thinking` |
| Image models | `Qwen-Image-2.0` |
| Video models | `Wan2.2` |

Standard NewAPI can proxy chat-shaped requests directly. Image/video compatibility depends on NewAPI support for the corresponding media endpoints and async video task polling routes.

## Operational Notes

- Keep qianwen.com login and service traffic on the same server/IP whenever possible.
- QR login material can expire or be risk-controlled by qianwen.com; re-login through Admin when tests fail.
- Delete stale QR sessions so Chromium processes and temporary profiles do not accumulate.
- Failed image/video submissions mark the selected account `unknown`; run Admin test or re-login before putting it back into rotation.
- Quota sync is not implemented because qianwen.com quota endpoint and response shape still need stable protocol capture.

## Data

Default data files:

```text
./data/qianwen-web-01.sqlite
./data/login-sessions/<session-id>/
```

Docker deployment should mount `/app/data` to a persistent host directory.
