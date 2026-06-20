# Local Development Guide

## Prerequisites

| Tool | Minimum version | Check |
|---|---|---|
| Docker | 24+ | `docker --version` |
| Docker Compose | v2 plugin | `docker compose version` |
| Go | 1.21+ (optional, for local builds) | `go version` |

---

## One-time Setup

### 1. Add /etc/hosts entries

All services run on subdomains of `centralauth.local`. Add them to your hosts file once:

```bash
sudo ./setup-hosts.sh
```

This appends the following block to `/etc/hosts`:

```
# centralauth.local (central-auth dev)
127.0.0.1 centralauth.local
127.0.0.1 auth.centralauth.local
127.0.0.1 api.auth.centralauth.local
127.0.0.1 analytics.centralauth.local
127.0.0.1 api.analytics.centralauth.local
127.0.0.1 report.centralauth.local
127.0.0.1 api.report.centralauth.local
127.0.0.1 transaction.centralauth.local
127.0.0.1 api.transaction.centralauth.local
```

---

## Starting the Project

```bash
# First run (or after source changes) — builds Docker images
docker compose up --build --remove-orphans

# Subsequent runs (no source changes)
docker compose up

# Background
docker compose up -d --build
```

### Teardown

```bash
# Stop containers (keeps data)
docker compose down

# Stop and wipe the postgres volume
docker compose down -v
```

---

## Service URLs

All traffic goes through nginx on **port 8080**.

### Frontends (browser entry points)

| Service | URL |
|---|---|
| Landing page | http://centralauth.local:8080 |
| Login | http://auth.centralauth.local:8080 |
| Analytics | http://analytics.centralauth.local:8080 |
| Reports | http://report.centralauth.local:8080 |
| Transactions | http://transaction.centralauth.local:8080 |

### Backend APIs (JSON)

| Service | Base URL | Endpoints |
|---|---|---|
| Auth | http://api.auth.centralauth.local:8080 | `POST /login`, `GET /logout`, `GET /health` |
| Analytics | http://api.analytics.centralauth.local:8080 | `GET /analytics`, `GET /health` |
| Reports | http://api.report.centralauth.local:8080 | `GET /reports`, `GET /health` |
| Transactions | http://api.transaction.centralauth.local:8080 | `GET /transactions`, `GET /health` |

### Debug ports (direct access, bypasses nginx)

| Service | Port |
|---|---|
| auth-service | http://localhost:4000 |
| report-service | http://localhost:4001 |
| analytics-service | http://localhost:4002 |
| transaction-service | http://localhost:4003 |
| PostgreSQL | localhost:5433 |

---

## Demo Accounts

| Email | Password |
|---|---|
| alice@example.com | demo123 |
| bob@example.com | demo123 |
| carol@example.com | demo123 |

---

## Verifying Everything Works

### 1. Check API health endpoints

```bash
curl http://api.auth.centralauth.local:8080/health
# {"service":"auth","status":"ok"}

curl http://api.analytics.centralauth.local:8080/health
# {"service":"analytics","status":"ok"}

curl http://api.report.centralauth.local:8080/health
# {"service":"report","status":"ok"}

curl http://api.transaction.centralauth.local:8080/health
# {"service":"transaction","status":"ok"}
```

### 2. Login and observe the central_auth cookie being set

```bash
# Login — follow the redirect and capture all Set-Cookie headers
curl -c /tmp/cookies.txt -D - -s \
  -X POST http://api.auth.centralauth.local:8080/login \
  -d "email=alice@example.com&password=demo123&return_to="

# Inspect what is in the jar
cat /tmp/cookies.txt
# You should see: central_auth  .centralauth.local
```

### 3. First request to a service — observe the service token being issued

```bash
# First request to analytics — sends central_auth, gets analytics_token back
curl -c /tmp/cookies.txt -b /tmp/cookies.txt -D - -s \
  http://api.analytics.centralauth.local:8080/analytics

# The response headers should contain:
# Set-Cookie: analytics_token=...; Path=/; HttpOnly; SameSite=Lax; Max-Age=3600

# The response body includes scope and permissions:
# {
#   "email": "alice@example.com",
#   "user_id": "1",
#   "scope": "analytics",
#   "permissions": ["read:stats","read:channels"],
#   "stats": [...],
#   "channels": [...]
# }
```

### 4. Subsequent request — service token is used directly (fast path)

```bash
# Second request to analytics — analytics_token now in jar, no central_auth needed
curl -c /tmp/cookies.txt -b /tmp/cookies.txt -D - -s \
  http://api.analytics.centralauth.local:8080/analytics

# No Set-Cookie in the response — service token already issued, no exchange needed
```

### 5. Test SSO — each service issues its own scoped token

```bash
# All three services accept the same central_auth and each issues its own token
curl -c /tmp/cookies.txt -b /tmp/cookies.txt -D - -s \
  http://api.report.centralauth.local:8080/reports
# Set-Cookie: report_token=...; Max-Age=1800

curl -c /tmp/cookies.txt -b /tmp/cookies.txt -D - -s \
  http://api.transaction.centralauth.local:8080/transactions
# Set-Cookie: transaction_token=...; Max-Age=900

cat /tmp/cookies.txt
# Four cookies: central_auth, analytics_token, report_token, transaction_token
```

### 6. Verify scope isolation — a service token is rejected by a different service

```bash
# Extract just the analytics_token from the jar
ANALYTICS_TOKEN=$(awk '/analytics_token/ {print $NF}' /tmp/cookies.txt)

# Send it to the report service — should return 401 (wrong scope)
curl -H "Cookie: analytics_token=$ANALYTICS_TOKEN" \
  http://api.report.centralauth.local:8080/reports
# {"error":"unauthorized"}
```

---

## Project Structure

```
central-auth/
├── auth-service/
│   ├── db/
│   │   └── db.go               # Connect() and SeedIfEmpty()
│   ├── handlers/
│   │   └── auth.go             # PostLogin, GetLogout
│   ├── middleware/
│   │   └── cors.go             # CORS only (auth-service has no RequireAuth)
│   ├── repository/
│   │   └── user.go             # FindByEmail — PostgreSQL
│   ├── service/
│   │   └── auth.go             # Login(), mintJWT() — issues central_auth 24h
│   └── main.go
│
├── analytics-service/
│   ├── handlers/
│   │   └── analytics.go        # GetAnalytics — returns scope, permissions, data
│   ├── middleware/
│   │   ├── auth.go             # RequireAuth: Tier 1 analytics_token, Tier 2 central_auth
│   │   └── cors.go
│   ├── repository/
│   │   └── analytics.go        # GetAnalyticsData — in-memory structs
│   ├── service/
│   │   ├── analytics.go        # GetData()
│   │   └── token.go            # IssueAnalyticsToken 1h, ValidateAnalyticsToken
│   └── main.go
│
├── report-service/
│   ├── handlers/
│   │   └── report.go
│   ├── middleware/
│   │   ├── auth.go             # RequireAuth: Tier 1 report_token 30min, Tier 2 central_auth
│   │   └── cors.go
│   ├── repository/
│   │   └── report.go
│   ├── service/
│   │   ├── report.go
│   │   └── token.go            # IssueReportToken 30min, ValidateReportToken
│   └── main.go
│
├── transaction-service/
│   ├── handlers/
│   │   └── transaction.go
│   ├── middleware/
│   │   ├── auth.go             # RequireAuth: Tier 1 transaction_token 15min, Tier 2 central_auth
│   │   └── cors.go
│   ├── repository/
│   │   └── transaction.go
│   ├── service/
│   │   ├── transaction.go
│   │   └── token.go            # IssueTransactionToken 15min, ValidateTransactionToken
│   └── main.go
│
├── frontends/
│   ├── auth/index.html         # Login form — posts to api.auth.centralauth.local:8080
│   ├── analytics/index.html    # Fetches api.analytics.centralauth.local:8080/analytics
│   ├── report/index.html       # Fetches api.report.centralauth.local:8080/reports
│   └── transaction/index.html  # Fetches api.transaction.centralauth.local:8080/transactions
│
├── nginx/
│   └── nginx.conf              # 9 virtual host blocks — frontends and API proxies
│
├── app/
│   └── index.html              # Landing page with links to all services
│
├── docs/
│   ├── architecture.md         # System diagrams — routing, auth flow, service layers
│   ├── central-auth-architecture.md  # Deep dive: dual-token flow and scope isolation
│   ├── api-reference.md        # All endpoints with request/response examples
│   └── local-development.md    # This file
│
├── docker-compose.yml
└── setup-hosts.sh
```

---

## Environment Variables

### auth-service

| Variable | Description | Default |
|---|---|---|
| `DB_DSN` | PostgreSQL connection string | — |
| `JWT_SECRET` | HMAC secret — signs `central_auth` JWTs | — |
| `COOKIE_MAX_AGE` | Cookie TTL in seconds | `86400` |
| `COOKIE_DOMAIN` | Cookie domain (must have leading dot) | `.centralauth.local` |
| `AUTH_FRONTEND_URL` | Redirect target on logout / error | `http://auth.centralauth.local:8080` |
| `DEFAULT_REDIRECT_URL` | Redirect target after login with no `return_to` | `http://analytics.centralauth.local:8080` |
| `ALLOWED_ORIGIN` | CORS allowed origin | `http://auth.centralauth.local:8080` |
| `PORT` | Listening port | `4000` |

### analytics / report / transaction services

| Variable | Description |
|---|---|
| `JWT_SECRET` | Must match auth-service — used to both **verify** `central_auth` and **sign** service tokens |
| `ALLOWED_ORIGIN` | CORS allowed origin (the corresponding frontend URL) |
| `PORT` | Listening port |

> **Note on JWT_SECRET:** A single shared secret is used for simplicity. In production you would use asymmetric keys: auth-service signs with a private key, each service verifies with the corresponding public key, and each service has its own private key for its service tokens.

---

## Service Token TTLs (hardcoded in `service/token.go`)

| Service | Cookie | TTL | Rationale |
|---|---|---|---|
| auth-service | `central_auth` | 24h | Long-lived identity token |
| analytics-service | `analytics_token` | 1h | Aggregate metrics — moderate sensitivity |
| report-service | `report_token` | 30min | Business reports — higher sensitivity |
| transaction-service | `transaction_token` | 15min | Financial data — shortest window |

---

## Troubleshooting

**Port 8080 already in use**
```bash
ss -tlnp | grep 8080
# Change the nginx port mapping in docker-compose.yml and update all frontend URLs
```

**"Could not reach API" on a frontend page**
- Confirm `/etc/hosts` entries: `grep centralauth /etc/hosts`
- Confirm all containers are running: `docker compose ps`
- Check health endpoints with curl (see above)

**Login redirects to `/` instead of the original page**
- `return_to` must be URL-encoded. The login form handles this automatically.
- Check browser DevTools → Network → the POST body to confirm the value.

**"invalid credentials" error**
- Use exactly `demo123` as the password.
- Database may need reseeding: `docker compose down -v && docker compose up --build`

**central_auth cookie not shared between subdomains**
- Confirm `COOKIE_DOMAIN=.centralauth.local` in docker-compose.yml (leading dot required).
- Confirm browser is accessing via `*.centralauth.local:8080`, not `localhost`.
- Check the `Set-Cookie` header in DevTools → Network on the auth POST response.

**Service token not being set on first request**
- Confirm `JWT_SECRET` is identical across all four services in docker-compose.yml.
- If it differs, `parseCentralToken` will reject the central_auth signature and fall through to a 401.
- Check service logs: `docker compose logs analytics-service`

**Service token accepted but data still returns 401**
- The service token carries a `scope` claim. A token issued for analytics is rejected by report or transaction even with the same secret.
- Use `jwt.io` to decode the cookie value and check the `scope` field.

**Stale images after code changes**
```bash
docker compose up --build --remove-orphans
```
Without `--build`, Docker reuses cached layers and will run old code.
