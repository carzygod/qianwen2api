# QIANWEN-WEB-01 / qianwen2api

QIANWEN-WEB-01 is a qianwen.com web reverse-proxy fork based on `kao0312/qianwen2api`.

This fork keeps the original guest chat adapter and adds the operational foundation needed for a Doubao2API-style provider:

- SQLite storage, no Redis required.
- Admin WebUI at `/admin?key=...`.
- Login account material CRUD for qianwen.com cookies/sessions.
- Account test API with conservative validity rules.
- Model registry backed by SQLite.
- Image/video compatible route skeletons.
- Persistent task records for image/video work.
- Health and Admin APIs for deployment management.

Important current boundary:

The original upstream repository only implemented guest chat. qianwen.com logged-in image/video protocols still require browser login and Network capture. Until that protocol is captured and implemented, image/video routes intentionally return `qianwen_image_protocol_required` or `qianwen_video_protocol_required` instead of pretending to be usable.

## Quick Start

```bash
cp .env.example .env
go run main.go
```

Docker:

```bash
docker build -t qianwen-web-01:latest .
docker run -d --name qianwen-web-01 \
  -p 19091:8000 \
  -e AUTH_KEY=change-me-api-key \
  -e ADMIN_KEY=change-me-admin-key \
  -e POOL_SIZE=1 \
  -v "$PWD/data:/app/data" \
  qianwen-web-01:latest
```

Open:

```text
http://127.0.0.1:19091/admin?key=change-me-admin-key
```

## Environment

| Variable | Default | Description |
|---|---:|---|
| `HOST` | `0.0.0.0` | Listen host |
| `PORT` | `8080` locally, `8000` in Docker | Listen port |
| `AUTH_KEY` | empty | Bearer token for `/v1/*` APIs |
| `ADMIN_KEY` | `AUTH_KEY` | Admin WebUI/API key |
| `POOL_SIZE` | `20` | Guest chat pool size; set `0` to disable |
| `REFRESH_HOURS` | `10` | Guest UMID refresh period |
| `DATA_DIR` | `./data` | Data directory |
| `DATABASE_PATH` | `./data/qianwen-web-01.sqlite` | SQLite database path |
| `PUBLIC_BASE_URL` | empty | Public base URL for future asset URLs |
| `DEFAULT_CHAT_MODEL` | `tongyi-qwen3-max-model` | Default chat model |
| `DEFAULT_IMAGE_MODEL` | `Qwen-Image-2.0` | Placeholder image model until protocol capture |
| `DEFAULT_VIDEO_MODEL` | `Wan2.2` | Placeholder video model until protocol capture |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Public APIs

| Method | Path | Status |
|---|---|---|
| `GET` | `/health` | Implemented |
| `GET` | `/v1/models` | Implemented |
| `POST` | `/v1/chat/completions` | Implemented through the original guest adapter |
| `POST` | `/v1/images/generations` | Route and task persistence implemented; upstream protocol pending |
| `POST` | `/v1/video/generations` | Route and task persistence implemented; upstream protocol pending |
| `GET` | `/v1/video/generations/{task_id}` | Implemented |
| `POST` | `/v1/videos/generations` | Compatibility route implemented |
| `GET` | `/v1/videos/generations/{task_id}` | Compatibility route implemented |
| `GET` | `/v1/tasks/{task_id}` | Implemented |

## Admin APIs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/summary` | Runtime summary |
| `GET` | `/api/accounts` | Account list |
| `POST` | `/api/accounts` | Add account material |
| `GET` | `/api/accounts/{id}` | Account detail |
| `DELETE` | `/api/accounts/{id}` | Delete account |
| `POST` | `/api/accounts/{id}/test` | Test account |
| `POST` | `/api/accounts/{id}/quota/sync` | Placeholder for quota sync |
| `GET` | `/api/tasks` | Task list |
| `GET` | `/api/tasks/{id}` | Task detail |
| `GET` | `/api/models` | Model registry |

Admin requests accept:

```text
X-Admin-Key: <ADMIN_KEY>
```

or:

```text
/admin?key=<ADMIN_KEY>
```

## Chat Example

```bash
curl http://127.0.0.1:19091/v1/chat/completions \
  -H "Authorization: Bearer change-me-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tongyi-qwen3-max-model",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": false
  }'
```

## Add Account Material

```bash
curl http://127.0.0.1:19091/api/accounts \
  -H "X-Admin-Key: change-me-admin-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "qianwen-account-1",
    "type": "login_cookie",
    "cookie_string": "a=b; c=d",
    "capabilities_json": "{\"chat\":true,\"image\":true,\"video\":true}",
    "enabled": true
  }'
```

The account will remain `unknown` until a real qianwen.com logged-in model call succeeds.

## Image/Video Boundary

Image and video routes already perform:

1. Bearer authentication.
2. Request validation.
3. Account capability selection.
4. SQLite task creation.
5. Clear protocol-required error response.

They do not yet call qianwen.com upstream image/video endpoints because those endpoints and payloads must be captured from a logged-in browser session.

Next implementation step:

1. Login to qianwen.com on the deployment server IP.
2. Capture chat/image/video request URL, headers, payload, polling response and quota response.
3. Implement `QianwenLoginClient`, `QianwenImageClient` and `QianwenVideoClient`.
4. Enable real account tests and scheduler failover.
