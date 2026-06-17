# Central Auth — SSO Demo

Five services, one shared cookie. Demonstrates the same-domain session-sharing pattern used by Google for Gmail, Drive, and Docs.

## How the auth works

`auth-service` is the only service with a login form. On successful login it sets a single `central_auth` cookie (signed JWT) on `localhost`. Because all services are on `localhost` (RFC 6265 ignores ports in domain matching), every downstream service receives this cookie automatically on every request.

Each downstream service's middleware validates the JWT locally using the shared `JWT_SECRET`. No redirect chain, no callback endpoints — just one shared cookie validated at the edge of each service.

```
Cold start:   GET /reports → 302 /login?return_to=... → user logs in → SET central_auth cookie → 302 /reports → renders page
Warm start:   GET /analytics → middleware sees valid cookie → renders page  (zero redirects)
Logout:       GET /logout → Max-Age=0 on central_auth → cookie gone everywhere
```

## Services

| Service             | Port | Description                      |
|---------------------|------|----------------------------------|
| auth-service        | 4000 | Login, logout, cookie owner      |
| report-service      | 4001 | Protected page (fake reports)    |
| analytics-service   | 4002 | Protected page (fake metrics)    |
| transaction-service | 4003 | Protected page (fake txns)       |
| app (nginx)         | 5173 | Static launcher page             |

Demo accounts (all use password `demo123`):
- `alice@example.com`
- `bob@example.com`
- `carol@example.com`

## Running with Docker Compose

```bash
docker compose up --build
```

Then open http://localhost:5173

> **Note:** Port 5432 may conflict with a local Postgres. The compose file maps postgres to 5433:5432 to avoid it.

## Running locally (without Docker)

**Postgres** must be running. Create the database and schema:

```bash
psql -U postgres -c "CREATE DATABASE centralauth;"
psql -U postgres -d centralauth < auth-service/db/seed.sql
```

Copy `.env.example` → `.env` in each service directory, then start each in its own terminal:

```bash
cd auth-service    && cp .env.example .env && go run .   # :4000
cd report-service  && cp .env.example .env && go run .   # :4001
cd analytics-service && cp .env.example .env && go run . # :4002
cd transaction-service && cp .env.example .env && go run . # :4003

# landing page — any static server will do:
cd app && python3 -m http.server 5173
```

Demo users are seeded automatically by auth-service on first startup when the `users` table is empty.

## Demo script

1. Open http://localhost:5173
2. Click **Reports** — land on the login form (one time only)
3. Sign in as `alice@example.com` / `demo123`
4. Land on the Reports page: "Logged in as alice@example.com"
5. Click **Transactions** in the nav — page loads directly, no login form
6. Click **Analytics** — same, instant, no login form
7. Click **Logout** — clears `central_auth` everywhere
8. Visit any service — login form appears again

## Project layout

```
central-auth/
├── auth-service/              # :4000 — login form, JWT cookie issuer
│   ├── db/
│   │   ├── db.go              # Postgres + auto-seed on first startup
│   │   └── seed.sql           # CREATE TABLE users
│   ├── handlers/auth.go       # GET /login  POST /login  GET /logout
│   ├── templates/login.html
│   └── main.go
├── report-service/            # :4001
├── analytics-service/         # :4002
├── transaction-service/       # :4003
│   # each downstream service has:
│   #   middleware/auth.go     — JWT cookie validation, redirect if missing
│   #   handlers/<name>.go     — renders the protected page
│   #   templates/<name>.html
├── app/index.html             # :5173 — static launcher
└── docker-compose.yml
```
