# Architecture — Central Auth Demo

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#1-system-architecture)
3. [Domain & Nginx Routing](#2-domain--nginx-routing)
4. [Authentication & Token Flow](#3-authentication--token-flow)
5. [Cookie & Token Scope](#4-cookie--token-scope)
6. [Service Internal Architecture](#5-service-internal-architecture)
7. [Docker Infrastructure](#6-docker-infrastructure)
8. [Data Flow — Protected Page Load](#7-data-flow--protected-page-load)
9. [Logout Flow](#8-logout-flow)

---

## Overview

Central Auth Demo implements a **two-tier token architecture** across four independent microservices behind a single nginx reverse proxy.

**Tier 1 — Central identity token (`central_auth`)**
Issued by the auth service after login. Long-lived (24 h), shared across all `*.centralauth.local` subdomains. Proves *who you are*.

**Tier 2 — Service tokens (`analytics_token`, `report_token`, `transaction_token`)**
Issued by each service on first access, in exchange for a valid central token. Short-lived, host-only, carry service-specific scope and permissions. Prove *what you can do* within that service.

This mirrors how real-world SSO works: one identity provider, multiple resource servers each enforcing their own authorization policies.

**Tech stack**

| Layer | Technology |
|---|---|
| Backend services | Go 1.21 + Gin |
| Frontend | Static HTML + Vanilla JS (nginx-served) |
| Reverse proxy | nginx (alpine) |
| Token format | JWT HS256 |
| Database | PostgreSQL 16 (auth-service only) |
| Container runtime | Docker Compose |

---

## 1. System Architecture

```mermaid
graph TB
    Browser(["Browser"])

    subgraph Host ["Host Machine — 127.0.0.1"]
        subgraph Docker ["Docker Network"]
            Nginx["nginx:alpine\nPort 80 — host 8080"]

            subgraph Frontends ["Static Frontends — nginx volumes"]
                AuthFE["auth.centralauth.local:8080"]
                AnalyticsFE["analytics.centralauth.local:8080"]
                ReportFE["report.centralauth.local:8080"]
                TxFE["transaction.centralauth.local:8080"]
            end

            subgraph BackendAPIs ["Go Backend APIs"]
                AuthSvc["auth-service :4000\napi.auth.centralauth.local:8080\nIssues central_auth JWT 24h"]
                AnalyticsSvc["analytics-service :4002\napi.analytics.centralauth.local:8080\nIssues analytics_token 1h"]
                ReportSvc["report-service :4001\napi.report.centralauth.local:8080\nIssues report_token 30min"]
                TxSvc["transaction-service :4003\napi.transaction.centralauth.local:8080\nIssues transaction_token 15min"]
            end

            PG[("PostgreSQL :5432\nhost port 5433")]
        end
    end

    Browser -->|"*.centralauth.local:8080\n/etc/hosts to 127.0.0.1"| Nginx
    Nginx -->|"auth.centralauth.local"| AuthFE
    Nginx -->|"api.auth.centralauth.local"| AuthSvc
    Nginx -->|"analytics.centralauth.local"| AnalyticsFE
    Nginx -->|"api.analytics.centralauth.local"| AnalyticsSvc
    Nginx -->|"report.centralauth.local"| ReportFE
    Nginx -->|"api.report.centralauth.local"| ReportSvc
    Nginx -->|"transaction.centralauth.local"| TxFE
    Nginx -->|"api.transaction.centralauth.local"| TxSvc
    AuthSvc -->|"DB_DSN"| PG

    classDef frontend fill:#e0f2fe,stroke:#0284c7
    classDef backend fill:#dcfce7,stroke:#16a34a
    classDef infra fill:#fef9c3,stroke:#ca8a04
    class AuthFE,AnalyticsFE,ReportFE,TxFE frontend
    class AuthSvc,AnalyticsSvc,ReportSvc,TxSvc backend
    class Nginx,PG infra
```

---

## 2. Domain & Nginx Routing

Every subdomain of `centralauth.local` resolves to `127.0.0.1` via `/etc/hosts`. Nginx inspects the `Host` header and routes to the correct static file directory or backend service.

```mermaid
graph LR
    subgraph EtcHosts ["/etc/hosts"]
        H["127.0.0.1  *.centralauth.local"]
    end

    subgraph NginxRouting ["nginx virtual hosts — port 8080 to 80"]
        direction TB
        R1["centralauth.local → /var/www/app"]
        R2["auth.centralauth.local → /var/www/auth"]
        R3["api.auth.centralauth.local → proxy auth-service:4000"]
        R4["analytics.centralauth.local → /var/www/analytics"]
        R5["api.analytics.centralauth.local → proxy analytics-service:4002"]
        R6["report.centralauth.local → /var/www/report"]
        R7["api.report.centralauth.local → proxy report-service:4001"]
        R8["transaction.centralauth.local → /var/www/transaction"]
        R9["api.transaction.centralauth.local → proxy transaction-service:4003"]
    end

    subgraph Backends ["Go Services — Docker internal"]
        B1["auth-service:4000"]
        B2["analytics-service:4002"]
        B3["report-service:4001"]
        B4["transaction-service:4003"]
    end

    subgraph StaticFiles ["Static Files — Docker volumes"]
        S1["/var/www/auth"]
        S2["/var/www/analytics"]
        S3["/var/www/report"]
        S4["/var/www/transaction"]
        S5["/var/www/app"]
    end

    EtcHosts -->|"browser to 127.0.0.1:8080"| NginxRouting
    R1 --> S5
    R2 --> S1
    R3 --> B1
    R4 --> S2
    R5 --> B2
    R6 --> S3
    R7 --> B3
    R8 --> S4
    R9 --> B4
```

---

## 3. Authentication & Token Flow

### 3a. Login — Central Token Issued Once

```mermaid
sequenceDiagram
    actor User
    participant FE   as Frontend
    participant SVC  as Service API
    participant AUTH as Auth Frontend
    participant AAPI as Auth API
    participant DB   as PostgreSQL

    User  ->> FE:   Visit protected page
    FE    ->> SVC:  GET /data with credentials
    SVC   -->> FE:  401 unauthorized

    FE    ->> AUTH: Redirect to login with return_to param
    Note over AUTH: Shows login form<br/>return_to in hidden field

    User  ->> AAPI: POST /login with email and password
    AAPI  ->> DB:   Verify credentials via bcrypt
    DB    -->> AAPI: user record
    AAPI  ->> AAPI: Sign central_auth JWT — 24h TTL
    AAPI  -->> User: 302 Redirect to return_to
    Note over AAPI,User: Set-Cookie central_auth=JWT<br/>Domain=.centralauth.local<br/>HttpOnly SameSite=Lax Max-Age=86400
```

### 3b. First Request to a Service — Token Exchange

```mermaid
sequenceDiagram
    actor User
    participant FE  as Frontend
    participant SVC as Service API

    User ->> FE:  GET page after login redirect
    FE   ->> SVC: GET /data — sends central_auth cookie
    Note over SVC: Tier 1 — no service token yet
    Note over SVC: Tier 2 — central_auth valid
    SVC  ->> SVC: Issue scoped service token
    SVC  -->> FE: 200 with data
    Note over SVC,FE: Set-Cookie service_token=JWT<br/>host-only HttpOnly short TTL
    FE   -->> User: Page rendered with data
```

### 3c. Subsequent Requests — Fast Path

```mermaid
sequenceDiagram
    actor User
    participant FE  as Frontend
    participant SVC as Service API

    User ->> FE:  Any page interaction
    FE   ->> SVC: GET /data — sends service_token cookie
    Note over SVC: Tier 1 — service token valid — done
    SVC  -->> FE: 200 with data
    FE   -->> User: Page updated
```

### 3d. Service Token Expiry — Auto-Renewal

```mermaid
sequenceDiagram
    actor User
    participant FE  as Frontend
    participant SVC as Service API

    Note over User,SVC: service_token expired — central_auth still valid

    User ->> FE:  Page interaction
    FE   ->> SVC: GET /data — expired service_token plus central_auth
    Note over SVC: Tier 1 — service token expired — skip
    Note over SVC: Tier 2 — central_auth valid — reissue
    SVC  ->> SVC: Issue fresh service token
    SVC  -->> FE: 200 with data plus new Set-Cookie
    FE   -->> User: Page updated seamlessly
```

---

## 4. Cookie & Token Scope

### Central token — shared identity

```mermaid
graph TB
    subgraph CentralCookie ["central_auth — Domain = .centralauth.local — 24h"]
        CC["Issued by auth-service on login\nClaims: sub email\nSent to ALL *.centralauth.local requests"]
    end

    subgraph Uses ["Used for token exchange at first visit to each service"]
        AA["api.analytics.centralauth.local\nexchanges for analytics_token"]
        RA["api.report.centralauth.local\nexchanges for report_token"]
        TA["api.transaction.centralauth.local\nexchanges for transaction_token"]
    end

    CentralCookie --> AA
    CentralCookie --> RA
    CentralCookie --> TA

    classDef central fill:#dbeafe,stroke:#3b82f6,color:#000
    classDef svc fill:#d1fae5,stroke:#059669,color:#000
    class CentralCookie,CC central
    class AA,RA,TA svc
```

### Service tokens — scoped per service host

```mermaid
graph TB
    subgraph AT ["analytics_token — host-only — 1h"]
        A["scope: analytics\nread:stats read:channels\nOnly sent to api.analytics.centralauth.local"]
    end

    subgraph RT ["report_token — host-only — 30min"]
        R["scope: reports\nread:reports create:reports\nOnly sent to api.report.centralauth.local"]
    end

    subgraph TT ["transaction_token — host-only — 15min"]
        T["scope: transactions\nread:transactions\nOnly sent to api.transaction.centralauth.local"]
    end

    AT -. "rejected — wrong scope" .-> RT
    AT -. "rejected — wrong scope" .-> TT

    classDef svc fill:#d1fae5,stroke:#059669,color:#000
    classDef reject color:#dc2626
    class AT,A,RT,R,TT,T svc
```

**Why the TTLs differ**

| Token | TTL | Rationale |
|---|---|---|
| `central_auth` | 24h | Identity token — low sensitivity, long session |
| `analytics_token` | 1h | Aggregate metrics — moderate sensitivity |
| `report_token` | 30min | Business reports — higher sensitivity |
| `transaction_token` | 15min | Financial data — most sensitive, shortest window |

---

## 5. Service Internal Architecture

### Shared 3-layer structure (all services)

```mermaid
graph TB
    subgraph MW ["Middleware — runs before every request"]
        CORS["CORS\nmiddleware/cors.go\nAllow-Origin header\nOPTIONS preflight"]
        Auth["RequireAuth\nmiddleware/auth.go\nTier 1 — validate service token\nTier 2 — validate central token\nIssue service token if needed\nSet email user_id permissions in context"]
    end

    subgraph HL ["Handler Layer — handlers/"]
        H["Read request params\nCall service layer\nWrite JSON response\nIncludes scope and permissions"]
    end

    subgraph SL ["Service Layer — service/"]
        SD["Business logic file\nOrchestrates data"]
        ST["Token file\nIssue and validate service JWT\nDefines scope TTL permissions"]
    end

    subgraph RL ["Repository Layer — repository/"]
        R["Data access\nDB queries for auth\nIn-memory structs for others"]
    end

    Req["Request"] --> CORS --> Auth --> H
    H --> SD --> R
    Auth --> ST
    H --> Resp["Response"]

    classDef mw fill:#fde68a,stroke:#d97706
    classDef layer fill:#dbeafe,stroke:#3b82f6
    classDef token fill:#fce7f3,stroke:#db2777
    class CORS,Auth mw
    class H,SD,R layer
    class ST token
```

### auth-service — has database, no service token

```mermaid
graph LR
    subgraph auth ["auth-service"]
        H["handlers/auth.go\nPostLogin\nGetLogout"]
        S["service/auth.go\nLogin — bcrypt verify\nMintJWT — central_auth 24h"]
        R["repository/user.go\nFindByEmail"]
        DB[("PostgreSQL\nusers table")]
        DBPkg["db/db.go\nConnect\nSeedIfEmpty"]
        MW["middleware/cors.go\nCORS only — no auth middleware"]
    end

    MW --> H --> S --> R --> DB
    DBPkg --> DB

    classDef h fill:#dbeafe,stroke:#3b82f6
    classDef s fill:#dcfce7,stroke:#16a34a
    classDef r fill:#fef9c3,stroke:#ca8a04
    classDef mw fill:#fde68a,stroke:#d97706
    class H h
    class S s
    class R,DB,DBPkg r
    class MW mw
```

### analytics / report / transaction — two-tier auth, stateless data

```mermaid
graph LR
    subgraph analytics ["analytics-service — same pattern for report and transaction"]
        MW1["middleware/cors.go\nCORS"]
        MW2["middleware/auth.go\nTier 1 — validate analytics_token\nTier 2 — exchange central_auth\nfor new analytics_token"]
        H["handlers/analytics.go\nGetAnalytics\nIncludes scope and permissions"]
        SD["service/analytics.go\nGetData"]
        ST["service/token.go\nIssueAnalyticsToken 1h\nValidateAnalyticsToken\nscope=analytics"]
        R["repository/analytics.go\nGetAnalyticsData\nIn-memory structs"]
    end

    MW1 --> MW2 --> H --> SD --> R
    MW2 --> ST

    classDef mw fill:#fde68a,stroke:#d97706
    classDef h fill:#dbeafe,stroke:#3b82f6
    classDef s fill:#dcfce7,stroke:#16a34a
    classDef token fill:#fce7f3,stroke:#db2777
    classDef r fill:#fef9c3,stroke:#ca8a04
    class MW1,MW2 mw
    class H h
    class SD s
    class ST token
    class R r
```

---

## 6. Docker Infrastructure

```mermaid
graph TB
    subgraph Host ["Host Machine"]
        P8080["Port 8080 — main entry"]
        P4000["Port 4000 — debug"]
        P4001["Port 4001 — debug"]
        P4002["Port 4002 — debug"]
        P4003["Port 4003 — debug"]
        P5433["Port 5433 — debug"]
    end

    subgraph DockerNet ["Docker bridge network"]
        Nginx["nginx:alpine\nPort 80\nVolumes: app auth analytics report transaction"]
        Auth["auth-service :4000\nIssues central_auth JWT\nDepends on postgres healthy"]
        Analytics["analytics-service :4002\nIssues analytics_token\nDepends on auth-service"]
        Report["report-service :4001\nIssues report_token\nDepends on auth-service"]
        Tx["transaction-service :4003\nIssues transaction_token\nDepends on auth-service"]
        PG[("postgres:16-alpine :5432\nusers table\nseed.sql on init")]
    end

    P8080 --> Nginx
    P4000 --> Auth
    P4001 --> Report
    P4002 --> Analytics
    P4003 --> Tx
    P5433 --> PG

    Nginx -->|"proxy_pass"| Auth & Analytics & Report & Tx
    Auth -->|"DB_DSN"| PG
    PG -->|"healthcheck pg_isready"| Auth

    classDef infra fill:#fef9c3,stroke:#ca8a04
    classDef svc fill:#dcfce7,stroke:#16a34a
    classDef port fill:#f3f4f6,stroke:#6b7280
    class Nginx,PG infra
    class Auth,Analytics,Report,Tx svc
    class P8080,P4000,P4001,P4002,P4003,P5433 port
```

---

## 7. Data Flow — Protected Page Load

### First visit (token exchange happens transparently)

```mermaid
sequenceDiagram
    participant B  as Browser
    participant N  as nginx :8080
    participant FE as Static Frontend
    participant AP as analytics-service :4002

    B  ->> N:  GET analytics.centralauth.local:8080
    N  ->> FE: Serve index.html
    FE -->> B: HTML and JS

    Note over B: JS runs — sends central_auth cookie only

    B  ->> N:  GET api.analytics.centralauth.local:8080/analytics
    Note over N: Forwards Cookie central_auth to service
    N  ->> AP: proxy GET /analytics with Cookie central_auth

    Note over AP: CORS middleware sets Allow-Origin header
    Note over AP: RequireAuth Tier 1 — no analytics_token yet
    Note over AP: RequireAuth Tier 2 — central_auth valid
    Note over AP: Issue analytics_token 1h sign with JWT_SECRET

    AP -->> N: 200 JSON with scope and permissions
    Note over AP,N: Set-Cookie analytics_token=JWT host-only Max-Age=3600
    N  -->> B: Forward response with Set-Cookie

    Note over B: Browser stores analytics_token for api.analytics host<br/>JS renders analytics page
```

### Subsequent visits (fast path — no token operations)

```mermaid
sequenceDiagram
    participant B  as Browser
    participant N  as nginx :8080
    participant AP as analytics-service :4002

    B  ->> N:  GET api.analytics.centralauth.local:8080/analytics
    Note over N: Forwards both central_auth and analytics_token cookies
    N  ->> AP: proxy GET /analytics with both cookies

    Note over AP: CORS middleware
    Note over AP: RequireAuth Tier 1 — analytics_token valid — done

    AP -->> N: 200 JSON — no Set-Cookie this time
    N  -->> B: Forward response

    Note over B: JS updates page
```

---

## 8. Logout Flow

Logout clears only `central_auth`. Service tokens are short-lived and expire on their own schedule. After logout, `central_auth` is gone so no new service tokens can be issued — all services reject further requests once their service tokens expire.

```mermaid
sequenceDiagram
    actor User
    participant N    as nginx
    participant AU   as auth-service :4000
    participant AUTH as Auth Frontend

    Note over User: Clicks Logout on any service page

    User ->> N:   GET api.auth.centralauth.local:8080/logout
    N    ->> AU:  proxy GET /logout
    AU   -->> N:  302 Redirect to auth frontend
    Note over AU,N: Set-Cookie central_auth empty Max-Age=-1<br/>Domain=.centralauth.local — clears across all subdomains
    N    -->> User: Redirect followed

    Note over User: central_auth deleted<br/>Service tokens still in browser but expire on schedule<br/>No new service tokens can be issued

    User ->> AUTH: GET auth.centralauth.local:8080
    AUTH -->> User: Login form
```
