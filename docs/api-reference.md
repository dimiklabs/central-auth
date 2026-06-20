# API Reference

All backend APIs are exposed through nginx at `api.<service>.centralauth.local:8080`.

### Two-tier authentication

Every protected endpoint runs a two-tier check:

1. **Service token** (`analytics_token` / `report_token` / `transaction_token`) — short-lived, host-only, service-scoped. Used on all requests after the first visit.
2. **Central identity token** (`central_auth`) — long-lived, shared across `.centralauth.local`. Used only when no valid service token is found; the service transparently issues a new service token and sets it in the response `Set-Cookie`.

Pass `credentials: 'include'` (fetch) or `-b/-c cookies.txt` (curl) so both cookies are sent/stored automatically.

---

## Auth Service

**Base URL:** `http://api.auth.centralauth.local:8080`

---

### GET /health

Returns the service status. No authentication required.

**Response `200`**
```json
{
  "service": "auth",
  "status": "ok"
}
```

---

### POST /login

Validates credentials, sets the `central_auth` JWT cookie, and redirects to `return_to`.

**Request** — `application/x-www-form-urlencoded`

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | User email address |
| `password` | string | yes | User password |
| `return_to` | string | no | URL to redirect to after successful login |

**Responses**

| Status | Condition | Body |
|---|---|---|
| `302` | Success | Redirect to `return_to` (or `DEFAULT_REDIRECT_URL`). Sets `central_auth` cookie. |
| `302` | Bad credentials | Redirect to `AUTH_FRONTEND_URL?error=invalid+credentials&return_to=...` |

**Cookie set on success**

```
Set-Cookie: central_auth=<JWT>; Domain=.centralauth.local; Path=/; HttpOnly; SameSite=Lax; Max-Age=86400
```

**Example (curl)**
```bash
curl -c /tmp/cookies.txt -v \
  -X POST http://api.auth.centralauth.local:8080/login \
  -d "email=alice@example.com&password=demo123&return_to=http://analytics.centralauth.local:8080"
```

---

### GET /logout

Clears the `central_auth` cookie and redirects to the auth frontend.

**Response `302`**

Redirect to `AUTH_FRONTEND_URL`. Sets `central_auth` with `Max-Age=-1` to expire it immediately on all `.centralauth.local` subdomains.

**Example (curl)**
```bash
curl -b /tmp/cookies.txt -v \
  http://api.auth.centralauth.local:8080/logout
```

---

## Analytics Service

**Base URL:** `http://api.analytics.centralauth.local:8080`

Protected endpoints use two-tier auth: validate `analytics_token` first (fast path), fall back to `central_auth` to exchange for a new service token. Returns `401` if neither is present or valid.

---

### GET /health

**Response `200`**
```json
{
  "service": "analytics",
  "status": "ok"
}
```

---

### GET /analytics

Returns analytics stats and channel breakdown for the authenticated user.

**Request headers**

| Header | Value |
|---|---|
| `Cookie` | `central_auth=<JWT>` (sent automatically with `credentials:'include'`) |
| `Origin` | `http://analytics.centralauth.local:8080` (browser sets automatically) |

On the **first request** (no `analytics_token` yet), the response also includes:
```
Set-Cookie: analytics_token=<JWT>; Path=/; HttpOnly; SameSite=Lax; Max-Age=3600
```

**Response `200`**
```json
{
  "email": "alice@example.com",
  "user_id": "1",
  "scope": "analytics",
  "permissions": ["read:stats", "read:channels"],
  "stats": [
    { "label": "Monthly Active Users", "value": "24,812", "delta": "↑ 12% vs last month", "up": true },
    { "label": "Avg. Session Length",  "value": "4m 32s",  "delta": "↑ 8% vs last month",  "up": true },
    { "label": "Conversion Rate",      "value": "3.7%",    "delta": "↑ 0.4 pp vs last month", "up": true },
    { "label": "Bounce Rate",          "value": "41%",     "delta": "↑ 2 pp vs last month", "up": false }
  ],
  "channels": [
    { "name": "Organic Search", "sessions": 11240, "share": 45 },
    { "name": "Direct",         "sessions": 7445,  "share": 30 },
    { "name": "Referral",       "sessions": 3722,  "share": 15 },
    { "name": "Paid Search",    "sessions": 2405,  "share": 10 }
  ]
}
```

**Response `401`** — missing or invalid cookie
```json
{ "error": "unauthorized" }
```

**Example (curl)**
```bash
curl -b /tmp/cookies.txt \
  http://api.analytics.centralauth.local:8080/analytics
```

---

## Report Service

**Base URL:** `http://api.report.centralauth.local:8080`

Protected endpoints use two-tier auth: validate `report_token` first, fall back to `central_auth` to exchange for a new service token. Returns `401` if neither is present or valid.

---

### GET /health

**Response `200`**
```json
{ "service": "report", "status": "ok" }
```

---

### GET /reports

Returns the list of reports for the authenticated user.

On the **first request** (no `report_token` yet), the response also includes:
```
Set-Cookie: report_token=<JWT>; Path=/; HttpOnly; SameSite=Lax; Max-Age=1800
```

**Response `200`**
```json
{
  "email": "alice@example.com",
  "user_id": "1",
  "scope": "reports",
  "permissions": ["read:reports", "create:reports"],
  "reports": [
    { "id": "#001", "title": "Q1 Revenue Summary",          "date": "2024-03-31", "author": "alice@example.com", "status": "published" },
    { "id": "#002", "title": "User Acquisition — Feb 2024", "date": "2024-02-29", "author": "bob@example.com",   "status": "published" },
    { "id": "#003", "title": "Infrastructure Cost Analysis","date": "2024-04-15", "author": "carol@example.com", "status": "draft"     },
    { "id": "#004", "title": "Churn Risk Cohort Study",     "date": "2024-05-01", "author": "alice@example.com", "status": "draft"     }
  ]
}
```

**Response `401`**
```json
{ "error": "unauthorized" }
```

**Example (curl)**
```bash
curl -b /tmp/cookies.txt \
  http://api.report.centralauth.local:8080/reports
```

---

## Transaction Service

**Base URL:** `http://api.transaction.centralauth.local:8080`

Protected endpoints use two-tier auth: validate `transaction_token` first, fall back to `central_auth` to exchange for a new service token. Returns `401` if neither is present or valid.

---

### GET /health

**Response `200`**
```json
{ "service": "transaction", "status": "ok" }
```

---

### GET /transactions

Returns the transaction list for the authenticated user.

On the **first request** (no `transaction_token` yet), the response also includes:
```
Set-Cookie: transaction_token=<JWT>; Path=/; HttpOnly; SameSite=Lax; Max-Age=900
```

**Response `200`**
```json
{
  "email": "alice@example.com",
  "user_id": "1",
  "scope": "transactions",
  "permissions": ["read:transactions"],
  "transactions": [
    { "id": "TXN-8821", "date": "2024-05-10", "description": "Stripe payout — April",       "amount": "+$14,220.00", "type": "credit", "status": "settled" },
    { "id": "TXN-8820", "date": "2024-05-09", "description": "AWS — monthly bill",           "amount": "-$3,412.55",  "type": "debit",  "status": "settled" },
    { "id": "TXN-8819", "date": "2024-05-08", "description": "Contractor payment — design",  "amount": "-$2,500.00",  "type": "debit",  "status": "pending" },
    { "id": "TXN-8818", "date": "2024-05-07", "description": "Stripe payout — mid-month",   "amount": "+$8,100.00",  "type": "credit", "status": "settled" },
    { "id": "TXN-8817", "date": "2024-05-06", "description": "Refund — order #44291",        "amount": "-$149.00",    "type": "debit",  "status": "failed"  }
  ]
}
```

**Response `401`**
```json
{ "error": "unauthorized" }
```

**Example (curl)**
```bash
curl -b /tmp/cookies.txt \
  http://api.transaction.centralauth.local:8080/transactions
```

---

## CORS Policy

Protected API endpoints return these headers on every response (required for browser `fetch` with `credentials: 'include'`):

| Header | Value |
|---|---|
| `Access-Control-Allow-Origin` | The service's corresponding frontend origin (e.g. `http://analytics.centralauth.local:8080`) |
| `Access-Control-Allow-Credentials` | `true` |
| `Access-Control-Allow-Methods` | `GET, OPTIONS` (analytics/report/tx) · `GET, POST, OPTIONS` (auth) |
| `Access-Control-Allow-Headers` | `Content-Type` |

Preflight `OPTIONS` requests receive `204 No Content` with the same headers.

> **Note:** `Access-Control-Allow-Credentials: true` cannot be combined with `Access-Control-Allow-Origin: *`. Each service is configured with its exact frontend origin via the `ALLOWED_ORIGIN` environment variable.
