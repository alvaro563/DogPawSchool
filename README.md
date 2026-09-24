# DogPaw — Backend

API for managing dog care activities, reservations, passes and users.
Built with Go + Gin + PostgreSQL.

---

## Quick start (local development)

### 1. Prerequisites

- Go ≥ 1.22
- Docker (for Postgres)
- A copy of `openssl` (to generate secrets)

### 2. Configure secrets

The server **refuses to start without real secrets**. There are no
fallback defaults — defaults for `JWT_SECRET` or `DB_PASSWORD` are how
an operator ends up running production with a publicly-known signing
key.

```sh
# Backend env (copy template, fill in real values)
cp cmd/api/.env.example cmd/api/.env
openssl rand -base64 48      # paste into JWT_SECRET in cmd/api/.env
openssl rand -base64 24      # paste into DB_PASSWORD in cmd/api/.env

# docker-compose env (same idea, separate file)
cp .env.example .env
openssl rand -base64 24      # paste into POSTGRES_PASSWORD in .env
```

> **Never commit `cmd/api/.env` or root `.env`.** Both are git-ignored.
> The only secrets-related files in the repo are the `*.example`
> templates.

### 3. Run the dev script

The fastest way to get the whole stack up is `scripts/dev-run.sh`. It
verifies the `.env` files exist, starts Postgres if it's not already
running, waits for the healthcheck, then launches the API with the
secrets exported into the process.

```sh
./scripts/dev-run.sh
```

The script aborts with a clear message if either `.env` is missing —
no more "ENV is required" from Go.

If you want to run things manually instead:

```sh
# 1. Start Postgres
docker compose up -d

# 2. Wait for it to be healthy (the script does this automatically)
docker compose exec -T postgres pg_isready -U dogpaw_user -d dogpaw_db

# 3. Export secrets and run the API
set -a; source cmd/api/.env; set +a
go run ./cmd/api
```

Postgres is bound to `127.0.0.1:5432` only — no public surface.

### 4. Run the frontend

```sh
cd frontend
cp .env.example .env       # VITE_API_PROXY_TARGET=http://localhost:8080 by default
npm install
npm run dev
```

Open `http://localhost:5173/`. The Vite dev server proxies every
`/api/*` request to the Go backend on `localhost:8080` (configured via
`VITE_API_PROXY_TARGET`). Cookies travel same-origin so they survive
the round-trip without CORS preflight.

If you see "No se pudo conectar con el servidor" on the login form,
the API is not running or `VITE_API_PROXY_TARGET` points to the wrong
host. Check the Vite dev-server terminal — proxy errors are logged
there.

The API logs a single config banner on startup:

```
{"time":"...","level":"INFO","msg":"config loaded",
 "env":"development","port":8080,"db_sslmode":"disable","tls":false}
```

Confirm the `env` matches the value you expect. If you see `"env":"development"`
on a production box, something is wrong with your deployment.

---

## Configuration reference

### Required (every environment)

| Variable       | Constraint                                | Notes |
| -------------- | ----------------------------------------- | ----- |
| `ENV`          | `development` \| `staging` \| `production` | Typos fail. |
| `JWT_SECRET`   | ≥ 32 bytes                                | HS256 entropy floor (OWASP). |
| `DB_PASSWORD`  | ≥ 12 bytes                                | OWASP password baseline. |
| `DB_SSLMODE`   | `disable` \| `allow` \| `prefer` \| `require` \| `verify-ca` \| `verify-full` | Production = `require` or higher. |
| `DB_HOST`      | non-empty                                 | No `localhost` default on purpose. |
| `DB_USER`      | non-empty                                 | No default. |
| `DB_NAME`      | non-empty                                 | No default. |

### Optional (with sensible defaults)

`PORT` (8080), `SHUTDOWN_TIMEOUT` (15s), `DB_PORT` (5432),
`DB_MAX_OPEN_CONNS` (25), `DB_MAX_IDLE_CONNS` (5),
`DB_CONN_MAX_LIFETIME` (5m), `DB_PING_TIMEOUT` (30s),
`CORS_ORIGINS`, `TRUSTED_PROXIES`, `TLS_KEY_FILE`, `TLS_CERT_FILE`.

---

## Security

### Secrets

- **No fallback defaults for any secret.** If you forget to set
  `JWT_SECRET` or `DB_PASSWORD`, the binary exits with a clear error
  and a stack trace pointing to the env var it expected.
- **Length validation.** `JWT_SECRET` < 32 bytes or
  `DB_PASSWORD` < 12 bytes both fail at startup.
- **Closed value sets.** `ENV` and `DB_SSLMODE` must match one of
  the accepted strings. A typo like `ENV=producton` fails
  immediately rather than silently downgrading to development
  semantics.

### Network

- **docker-compose binds Postgres to `127.0.0.1:5432` only.** The
  port is not reachable from other hosts.
- The API binds to `0.0.0.0` by default because that's how Gin
  works; in production, put it behind a reverse proxy with TLS
  (`TLS_KEY_FILE` / `TLS_CERT_FILE`) and a firewall.

### Production checklist

- [ ] `ENV=production`
- [ ] `JWT_SECRET` ≥ 32 bytes, from a secret manager (not in `.env`)
- [ ] `DB_PASSWORD` ≥ 12 bytes, from a secret manager
- [ ] `DB_SSLMODE=require` or `verify-full`
- [ ] TLS configured (reverse proxy or `TLS_KEY_FILE` / `TLS_CERT_FILE`)
- [ ] `TRUSTED_PROXIES` set if behind a proxy (enables X-Forwarded-For so the IP rate-limit middleware and the login lockout key on the REAL client; unset = XFF ignored and all proxied requests share the proxy IP's buckets)
- [ ] IP rate limits tuned for expected traffic (`LOGIN_IP_*`, `REGISTER_IP_*`); account lockout defaults (`LOCKOUT_*`) are sane but adjustable
- [ ] `CORS_ORIGINS` set to the exact frontend origin (REQUIRED in production; LoadConfig fails fast if missing)
- [ ] `JWT_ACCESS_TTL` (default 1h) and `JWT_REFRESH_TTL` (default 24h) match the desired UX
- [ ] `COOKIE_SECURE=true` (default in production — set false only for HTTP-only deploys)
- [ ] `/swagger` is unreachable in production (the route is gated to `ENV != "production"`)
- [ ] Reverse proxy / CDN configured to forward the `Set-Cookie` and `Cookie` headers (Caddy/Cloudflare do this by default; nginx needs `proxy_pass_header Set-Cookie` only in special cases)
- [ ] `docker-compose.yml` not used as-is in production; use a
      managed Postgres instance or a properly secured cluster

### Authentication model

The API uses two HttpOnly cookies:

- `access_token` — short-lived (1h default), `SameSite=Lax`, used by every authenticated request.
- `refresh_token` — longer-lived (24h default), `SameSite=Lax`, used only by `POST /api/v1/auth/refresh`.

Both cookies are `SameSite=Lax` because the SPA and the API share one origin in production: the Render static site proxies `/api/*` to the backend (see `render.yaml`), so every fetch is same-site. `SameSite=None` (cross-site) is deliberately not used — it requires `Secure`, invites third-party-cookie blocking, and is unnecessary behind the proxy.

The SPA never sees the JWT. The http-client uses `credentials: 'include'` so cookies travel automatically, and a single-flight `tryRefresh()` interceptor handles token renewals transparently.

`POST /api/v1/auth/logout` clears both cookies. There is no server-side revocation list — the JWTs remain technically valid until they expire, but the cookie clear forces the browser to stop sending them. (For strict server-side revocation, bump `token_version` server-side, which is what `PATCH /api/v1/auth/password` already does on password change.)

`GET /api/v1/users/me` returns the user identified by the access cookie, used on first SPA load to decide whether to render the auth shell or kick to `/auth/login`.

### CSRF posture

`SameSite=Lax` on both cookies blocks the classic cross-site POST: an attacker's form or fetch cannot attach the session cookies, and there is no explicit CSRF token because the cookie policy already covers the vector. The same-origin `/api` proxy means legitimate traffic never needs `SameSite=None`. If a future deploy moves the SPA to a different registrable domain (true cross-site), revisit this: cookies would need `SameSite=None; Secure` — plus `Partitioned` (CHIPS) for browsers that block third-party cookies — or a double-submit CSRF token.

### Deployment on Render (same-origin proxy)

Production topology: **Render static site** (SPA, serves `frontend/dist`) + **Render web service** (Go API at `https://dogpawschool.onrender.com`). The static site rewrites `/api/*` to the API (declared in `render.yaml`, or manually in Dashboard → Static Site → Redirects/Rewrites):

```yaml
routes:
  - type: rewrite
    source: /api/*
    destination: https://dogpawschool.onrender.com/api/*
  - type: rewrite
    source: /*
    destination: /index.html
```

The browser then only ever talks to the SPA origin: cookies are first-party, CORS preflights disappear, and third-party-cookie blocking can no longer break the session.

**API environment (dashboard):**

- `ENV=production` — also forces `COOKIE_SECURE=true`.
- `CORS_ORIGINS=https://<spa-host>` — still required: browsers attach `Origin` to same-origin POSTs, and gin-contrib/cors rejects an unexpected origin with an empty `403` before the handler runs.
- `TRUSTED_PROXIES=<proxy CIDRs>` — so rate limits and lockout key on the client IP from `X-Forwarded-For` instead of the shared proxy IP.
- `JWT_SECRET`, `DB_*`, `COOKIE_SECURE` per the production checklist.

**Frontend build:** `frontend/.env.production` pins `VITE_API_BASE_URL=/api/v1`; never bake an absolute backend URL into the production bundle.

**Verify after deploy:**

```sh
# 1. API direct (cookie-jar round-trip proves server-side auth)
curl -c jar.txt -X POST https://dogpawschool.onrender.com/api/v1/auth/login \
  -H 'Content-Type: application/json' -d '{"email":"<email>","password":"<pw>"}'
curl -b jar.txt https://dogpawschool.onrender.com/api/v1/users/me   # expect 200

# 2. Through the SPA host (proves the proxy rewrite)
curl -i https://<spa-host>/api/v1/users/me   # expect the API's 401/200 JSON, not index.html

# 3. Browser: login → Application → Cookies → the SPA host holds
#    access_token (HttpOnly, Secure, SameSite=Lax); /users/me returns 200.
```

### Generating secrets

```sh
# JWT signing key
openssl rand -base64 48

# Database password
openssl rand -base64 24
```

---

## Development

### Build

```sh
make build      # -> bin/api
```

### Test

```sh
make test-unit          # fast, no Docker
make test               # everything (needs Docker for integration)
make test-race          # everything + race detector
```

### Lint

```sh
make vet
make lint               # needs golangci-lint
```

### Regenerate OpenAPI docs

```sh
make swagger
```

---

## Project layout

```
cmd/api/             # entrypoint + config + HTTP wiring
internal/domain/     # pure business entities (no I/O)
internal/usecase/    # application services (one per aggregate)
internal/handler/    # HTTP handlers, JSON marshalling
internal/repository/ # Postgres adapters (one per aggregate)
internal/crypto/     # JWT helpers
migrations/          # SQL migrations (golang-migrate format)
scripts/             # dev-run.sh and other local helpers
frontend/            # React + Vite + TanStack Router
docs/                # generated swagger.json / swagger.yaml
```
