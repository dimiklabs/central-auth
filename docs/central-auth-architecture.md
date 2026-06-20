# Central Auth — How It Works

---

## The Core Idea

Two-tier token architecture: **Central Auth** issues a long-lived *identity* token that proves who you are. Each service then exchanges it for a short-lived *service token* scoped to that service's own permissions. Subsequent requests skip the identity check entirely and use the faster service token.

---

## Token Comparison

| | Central Auth Token | Service Token |
|---|---|---|
| **Cookie name** | `central_auth` | `analytics_token` / `report_token` / `transaction_token` |
| **Issued by** | auth | The service itself |
| **TTL** | 24 hours | 1h / 30min / 15min (per service) |
| **Cookie domain** | `.centralauth.local` (shared) | Host-only (service API only) |
| **Claims** | `sub`, `email` | `sub`, `email`, `scope`, `permissions` |
| **Purpose** | Prove identity across all services | Authorize within one service |

---

## 1. System Overview

```mermaid
graph TB
    User(["User / Browser"])

    subgraph AuthSvc ["Central Auth Service"]
        AuthDB[("PostgreSQL\nusers table")]
        AuthIssue["Issues central_auth JWT\n24h TTL\nDomain .centralauth.local"]
    end

    subgraph AnalyticsSvc ["Analytics Service"]
        AT["Issues analytics_token\n1h TTL — host-only\nread:stats read:channels"]
    end

    subgraph ReportSvc ["Report Service"]
        RT["Issues report_token\n30min TTL — host-only\nread:reports create:reports"]
    end

    subgraph TxSvc ["Transaction Service"]
        TT["Issues transaction_token\n15min TTL — host-only\nread:transactions"]
    end

    User -- "POST /login" --> AuthSvc
    AuthSvc -- "Set-Cookie central_auth" --> User

    User -- "central_auth cookie" --> AnalyticsSvc
    AnalyticsSvc -- "Set-Cookie analytics_token" --> User

    User -- "central_auth cookie" --> ReportSvc
    ReportSvc -- "Set-Cookie report_token" --> User

    User -- "central_auth cookie" --> TxSvc
    TxSvc -- "Set-Cookie transaction_token" --> User

    classDef auth fill:#fde68a,stroke:#d97706,color:#000
    classDef svc fill:#d1fae5,stroke:#059669,color:#000
    class AuthSvc,AuthDB,AuthIssue auth
    class AnalyticsSvc,AT,ReportSvc,RT,TxSvc,TT svc
```

---

## 2. Two-Tier Auth Flow — First Visit to a Service

```mermaid
sequenceDiagram
    actor User
    participant FE   as Frontend
    participant SVC  as Service API
    participant AUTH as Auth Frontend
    participant AAPI as Auth API
    participant DB   as PostgreSQL

    Note over User,DB: Step 1 — Login (one time for all services)

    User  ->> FE:   Visit protected page
    FE    ->> SVC:  GET /data with credentials
    SVC   -->> FE:  401 unauthorized
    FE    ->> AUTH: Redirect to login with return_to param
    User  ->> AAPI: POST /login with email and password
    AAPI  ->> DB:   Verify credentials
    DB    -->> AAPI: user record
    AAPI  -->> User: 302 Redirect to return_to
    Note over AAPI,User: Set-Cookie central_auth=JWT 24h<br/>Domain=.centralauth.local

    Note over User,DB: Step 2 — First request to a service (token exchange)

    User  ->> FE:   GET protected page
    FE    ->> SVC:  GET /data — sends central_auth cookie
    Note over SVC:  Tier 1 check: no service token yet
    Note over SVC:  Tier 2 check: central_auth valid
    SVC   ->> SVC:  Issue service-scoped token
    SVC   -->> FE:  200 with data
    Note over SVC,FE: Set-Cookie service_token=JWT<br/>host-only — short TTL

    Note over User,DB: Step 3 — Subsequent requests (fast path)

    FE    ->> SVC:  GET /data — sends service_token cookie
    Note over SVC:  Tier 1 check: service token valid — done
    SVC   -->> FE:  200 with data — no central token needed
```

---

## 3. Token Validation Logic (per service)

```mermaid
graph TD
    Req["Incoming Request"]

    Req --> T1{"service_token\ncookie present?"}

    T1 -- "yes" --> V1{"Valid JWT?\nCorrect scope?\nNot expired?"}
    V1 -- "pass" --> OK["Set email, user_id, permissions\nin request context\n\nProceed to handler"]
    V1 -- "fail" --> T2

    T1 -- "no" --> T2{"central_auth\ncookie present?"}
    T2 -- "no" --> R401["Return 401 JSON"]
    T2 -- "yes" --> V2{"Valid JWT?\nNot expired?"}
    V2 -- "fail" --> R401
    V2 -- "pass" --> Issue["Issue service token\nSet-Cookie — host-only\nShort TTL + scoped permissions"]
    Issue --> OK

    classDef ok fill:#d1fae5,stroke:#059669
    classDef fail fill:#fee2e2,stroke:#dc2626
    classDef check fill:#fef9c3,stroke:#ca8a04
    class OK,Issue ok
    class R401 fail
    class V1,V2,T1,T2 check
```

---

## 4. Service Token Scope Isolation

Each service token carries a `scope` claim. A token issued by one service is rejected by another even if the signature is valid.

```mermaid
graph LR
    CT["central_auth cookie\nsub + email\n24h TTL\nDomain .centralauth.local"]

    subgraph AnalyticsAPI ["Analytics API"]
        AT["analytics_token\nscope: analytics\nread:stats read:channels\n1h TTL — host-only"]
    end

    subgraph ReportAPI ["Report API"]
        RT["report_token\nscope: reports\nread:reports create:reports\n30min TTL — host-only"]
    end

    subgraph TxAPI ["Transaction API"]
        TT["transaction_token\nscope: transactions\nread:transactions\n15min TTL — host-only"]
    end

    CT -- "exchanged at first visit" --> AT
    CT -- "exchanged at first visit" --> RT
    CT -- "exchanged at first visit" --> TT

    AT -. "rejected by report API\nwrong scope" .-> RT
    AT -. "rejected by tx API\nwrong scope" .-> TT

    classDef central fill:#dbeafe,stroke:#3b82f6
    classDef svc fill:#d1fae5,stroke:#059669
    classDef reject stroke-dasharray: 5 5,color:#dc2626
    class CT central
    class AT,RT,TT svc
```

---

## 5. Full Token Lifecycle

```mermaid
sequenceDiagram
    actor User
    participant SVC as Service API

    Note over User,SVC: T+0 — Login. central_auth set.

    User  ->> SVC:  Request 1 — central_auth only
    SVC   -->> User: Data + Set-Cookie service_token

    Note over User,SVC: T+0 to expiry — service token used directly

    User  ->> SVC:  Requests 2..N — service_token only
    SVC   -->> User: Data (no token operations)

    Note over User,SVC: Service token expires. central_auth still valid.

    User  ->> SVC:  Request N+1 — expired service_token
    Note over SVC:  Tier 1 fails — service token expired
    Note over SVC:  Tier 2 — central_auth still valid
    SVC   -->> User: Data + Set-Cookie new service_token

    Note over User,SVC: central_auth expires. Full re-login required.

    User  ->> SVC:  Request — both tokens expired
    SVC   -->> User: 401 unauthorized
```

---

## 6. Cookie Properties Side-by-Side

```mermaid
graph TB
    subgraph CentralCookie ["central_auth — Identity Cookie"]
        CC["Domain: .centralauth.local\nPath: /\nHttpOnly: true\nSameSite: Lax\nMax-Age: 86400 — 24h\nScope: all subdomains"]
    end

    subgraph ServiceCookies ["Service Cookies — Scoped per API host"]
        AC["analytics_token\nDomain: api.analytics.centralauth.local\nPath: /\nHttpOnly: true\nSameSite: Lax\nMax-Age: 3600 — 1h"]
        RC["report_token\nDomain: api.report.centralauth.local\nPath: /\nHttpOnly: true\nSameSite: Lax\nMax-Age: 1800 — 30min"]
        TC["transaction_token\nDomain: api.transaction.centralauth.local\nPath: /\nHttpOnly: true\nSameSite: Lax\nMax-Age: 900 — 15min"]
    end

    CentralCookie -- "exchanged for" --> ServiceCookies

    classDef central fill:#dbeafe,stroke:#3b82f6,color:#000
    classDef svc fill:#d1fae5,stroke:#059669,color:#000
    class CC central
    class AC,RC,TC svc
```

---

## Key Properties

| Property | Central Token | Analytics Token | Report Token | Transaction Token |
|---|---|---|---|---|
| Issued by | auth | analytics | report | transaction |
| TTL | 24h | 1h | 30min | 15min |
| Cookie | `central_auth` | `analytics_token` | `report_token` | `transaction_token` |
| Domain | `.centralauth.local` | host-only | host-only | host-only |
| Scope claim | _(none)_ | `analytics` | `reports` | `transactions` |
| Permissions | _(none)_ | `read:stats` `read:channels` | `read:reports` `create:reports` | `read:transactions` |
| Cross-service usable | Yes (identity) | No (scope rejected) | No (scope rejected) | No (scope rejected) |
| DB required | Yes | No | No | No |
