# Social Dorian

Admin app for managing an organization's connected social accounts, proxies, email inboxes, and automation tasks.

## Stack

| Layer    | Technology                |
|----------|---------------------------|
| Frontend | Vue 3 + Vite + Vue Router |
| Backend  | Go 1.19+ (stdlib HTTP)    |
| Database | MySQL (`doriansocial`)    |
| Workers  | Python Selenium scripts in `script/` (Facebook) |

## Database setup

Base schema + seed data live in `back-end/sql/schema.sql`. Task/credit tables are also created automatically on API start (`ensureSchema`) and documented in `back-end/sql/migrate_tasks.sql`.

```bash
mysql -u root -p < back-end/sql/schema.sql
```

This creates the `doriansocial` database with:

- `proxies` — proxy management records
- `accounts` — connected social accounts (`proxy_id` FK → `proxies.id`)
- `gmails` — email inboxes used for verification codes
- `users` — admin login (seeded from `.env`)
- `tasks` / `task_items` / `task_logs` / `workspace_credits` — campaigns engine (auto-migrated)

## Getting Started

### Backend

```bash
cd back-end
cp .env.example .env   # edit DB_USER / DB_PASSWORD / etc.
go run .
```

The API reads `back-end/.env` automatically (no `export` needed).

Optional: `SCRIPT_DIR=/home/social/script` if scripts are not at the default path.

API runs at `http://localhost:8080`

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check |
| `POST /api/auth/login` | Sign in (cookie session) |
| `POST /api/auth/logout` | Sign out |
| `GET /api/auth/me` | Current admin user |
| `GET /api/dashboard` | Aggregated stats |
| `GET/POST /api/accounts` | List / connect accounts |
| `GET/PUT /api/accounts/{id}` | Get / update account |
| `POST /api/accounts/{id}/open` | Start remote Chromium via account proxy |
| `GET /api/sessions/{token}/vnc/*` | noVNC viewer |
| `GET /api/sessions/{token}/websockify` | VNC WebSocket stream |
| `DELETE /api/sessions/{token}` | Stop remote Chromium session |
| `GET/POST /api/proxies` | List / add proxies |
| `GET/PUT/DELETE /api/proxies/{id}` | Proxy CRUD |
| `GET/POST /api/gmails` | Email inbox CRUD |
| `POST /api/gmails/{id}/pin-code` | Fetch Facebook verification code from mailbox |
| `GET /api/tasks` | List tasks / campaigns |
| `POST /api/tasks` | Create and queue a task (debits credits) |
| `GET /api/tasks/{id}` | Task detail + logs |
| `POST /api/tasks/{id}/cancel` | Cancel a running/queued task |
| `GET /api/credits` | Workspace credit balance |
| `GET /api/busy-accounts` | Account IDs currently in jobs |
| `GET /api/monitor/feed` | Platform activity feed (`?after=&limit=&level=&source=&taskId=&accountId=&q=&errors=1`) |
| `POST /api/uploads` | Upload image/video for New post tasks |
| `GET /api/uploads/{file}` | Serve an uploaded media file |
| `GET/POST /api/users` | List / create team users (**admin only**) |
| `PUT/DELETE /api/users/{id}` | Update / delete team user (**admin only**) |

The first admin is created from `ADMIN_EMAIL` / `ADMIN_PASSWORD` in `back-end/.env`. Additional **member** or **admin** users can be managed under Settings. All other API routes require a session cookie.

### Task workers

Queued tasks run Facebook automation via `script/fb_report.py`, `fb_reply.py`, `fb_post.py`, `fb_browse.py`, and `fb_login.py`.

- **Platform:** Facebook only for now (other platforms can be selected in UI but workers will fail with a clear log).
- **Proxy:** Honors `useAccountProxy` on the task and each account’s assigned proxy.
- **Profiles:** Reuses `/var/lib/dorian-browser/profiles/account-{id}`.
- **Media:** New post can upload files to `/var/lib/dorian-browser/uploads` (or paste a URL); local uploads are attached via `fb_post.py --media`.
- **Credits:** 1 credit per selected account is debited when the task is launched. Launch is blocked if balance is too low.
- **Failures:** Item status is set to `failed` with error details in `task_logs` (visible in task detail and Live feed).
- **Activity log:** Platform-wide `activity_logs` captures task worker output, account open/login steps, and proxy checks. Use **Live feed** with source/level filters (or Problems only) to diagnose what happened.

### Frontend

```bash
cd front-end
npm install
npm run dev
```

App runs at `http://localhost:5173`. Requests to `/api/*` are proxied to the Go backend.
