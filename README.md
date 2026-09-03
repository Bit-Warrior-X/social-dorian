# Social Dorian

Admin app for managing an organization's connected social accounts and proxies.

## Stack

| Layer    | Technology                |
|----------|---------------------------|
| Frontend | Vue 3 + Vite + Vue Router |
| Backend  | Go 1.19+ (stdlib HTTP)    |
| Database | MySQL (`doriansocial`)    |

## Database setup

Schema + seed data live in `back-end/sql/schema.sql`.

```bash
mysql -u root -p < back-end/sql/schema.sql
```

This creates the `doriansocial` database with:

- `proxies` — proxy management records
- `accounts` — connected social accounts (`proxy_id` FK → `proxies.id`)

## Getting Started

### Backend

```bash
cd back-end
cp .env.example .env   # edit DB_USER / DB_PASSWORD / etc.
go run .
```

The API reads `back-end/.env` automatically (no `export` needed).

API runs at `http://localhost:8080`

| Endpoint                    | Description        |
|-----------------------------|--------------------|
| `GET /api/health`           | Health check       |
| `GET /api/dashboard`        | Aggregated stats   |
| `GET /api/accounts`         | List accounts      |
| `POST /api/accounts`        | Connect account    |
| `GET /api/accounts/{id}`    | Get one account    |
| `PUT /api/accounts/{id}`    | Update account     |
| `POST /api/accounts/{id}/open` | Start remote Chromium via account proxy (persistent profile) |
| `GET /api/sessions/{token}/vnc/*` | noVNC viewer for the remote browser |
| `GET /api/sessions/{token}/websockify` | VNC WebSocket stream |
| `DELETE /api/sessions/{token}` | Stop remote Chromium session |
| `GET /api/proxies`          | List proxies       |
| `POST /api/proxies`         | Add proxy          |
| `GET /api/proxies/{id}`     | Get one proxy      |
| `PUT /api/proxies/{id}`     | Update proxy       |
| `DELETE /api/proxies/{id}`  | Delete proxy       |

### Frontend

```bash
cd front-end
npm install
npm run dev
```

App runs at `http://localhost:5173`. Requests to `/api/*` are proxied to the Go backend.
