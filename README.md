# Central Auth — SSO Demo

A local microservices demo that shows how a single shared identity cookie enables SSO across multiple subdomains, with each service additionally issuing its own short-lived scoped token.

## How it works

**Two-tier token flow**

1. `auth` is the only service with a login form. On success it sets `central_auth` (signed JWT, 24h) on `Domain=.centralauth.local` — shared across all subdomains.
2. On the first request to any downstream service, that service exchanges the `central_auth` for its own short-lived service token (`analytics_token`, `report_token`, or `transaction_token`) scoped to only that service.
3. Subsequent requests use the service token directly (fast path). When the service token expires, `central_auth` auto-renews it — no re-login.

```
First visit:  GET /analytics → 401 → redirect to login → POST /login
              → Set-Cookie central_auth → redirect → GET /analytics
              → exchange central_auth → Set-Cookie analytics_token → 200

Warm request: GET /analytics → analytics_token valid → 200 (no token ops)

Expiry:       analytics_token expired → central_auth still valid
              → reissue analytics_token → 200

Logout:       GET /logout → Max-Age=-1 on central_auth → service tokens
              expire on schedule → 401 on next request everywhere
```

## Services

| Directory | Port | Role |
|---|---|---|
| `auth/` | 4000 | Login · logout · issues `central_auth` JWT 24h |
| `analytics/` | 4002 | Issues `analytics_token` 1h — read:stats, read:channels |
| `report/` | 4001 | Issues `report_token` 30min — read:reports, create:reports |
| `transaction/` | 4003 | Issues `transaction_token` 15min — read:transactions |
| `nginx/` | 8080 | Reverse proxy — routes all `*.centralauth.local` subdomains |

All frontends live under `frontends/`:

| Directory | URL |
|---|---|
| `frontends/app/` | http://centralauth.local:8080 |
| `frontends/auth/` | http://auth.centralauth.local:8080 |
| `frontends/analytics/` | http://analytics.centralauth.local:8080 |
| `frontends/report/` | http://report.centralauth.local:8080 |
| `frontends/transaction/` | http://transaction.centralauth.local:8080 |

## Demo accounts

All use password `demo123`:
- `alice@example.com`
- `bob@example.com`
- `carol@example.com`

## Running

### 1. Add /etc/hosts entries (one time)

```bash
sudo ./setup-hosts.sh
```

### 2. Start

```bash
docker compose up --build
```

Then open http://centralauth.local:8080

### 3. Teardown

```bash
docker compose down        # keep data
docker compose down -v     # wipe postgres volume
```

## Project layout

```
central-auth/
├── auth/                   # Go: login, logout, issues central_auth JWT
│   ├── db/
│   │   ├── db.go           # Connect, SeedIfEmpty
│   │   └── seed.sql        # CREATE TABLE users
│   ├── handlers/auth.go
│   ├── middleware/
│   │   ├── cors.go
│   │   └── security.go     # security headers, rate limiter, request ID
│   ├── repository/user.go
│   ├── service/auth.go
│   ├── Dockerfile
│   └── main.go
│
├── analytics/              # Go: issues analytics_token 1h
├── report/                 # Go: issues report_token 30min
├── transaction/            # Go: issues transaction_token 15min
│   # each service has:
│   #   middleware/auth.go    — two-tier token validation + audit log
│   #   middleware/security.go — headers, request ID
│   #   service/token.go      — Issue/Validate service JWT
│
├── frontends/
│   ├── app/index.html       # Landing page
│   ├── auth/index.html      # Login form
│   ├── analytics/index.html
│   ├── report/index.html
│   └── transaction/index.html
│
├── nginx/nginx.conf         # Virtual hosts + rate limiting + CSP headers
├── docs/                    # Architecture diagrams and API reference
├── docker-compose.yml
└── setup-hosts.sh
```

## Security highlights

- Per-IP login rate limiting (5 req/min, 15-min lockout after 5 failures) at both nginx and application layers
- JWT `iss`/`aud`/`nbf` claims on every token; scope isolation prevents cross-service token reuse
- `SameSite=Strict` on service tokens; `SameSite=Lax` on central identity token
- Open redirect protection: `return_to` validated to `.centralauth.local` domain only
- Content Security Policy on every frontend, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff` on every response
- bcrypt cost 12 for passwords; JWT secret minimum 32 chars enforced at startup
- Full structured audit log (`slog`) on login, token exchange, and data access events

See `docs/` for full architecture diagrams and API reference.
