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
- [ ] `TRUSTED_PROXIES` set if behind a load balancer
- [ ] `CORS_ORIGINS` set to the exact frontend origin
- [ ] `docker-compose.yml` not used as-is in production; use a
      managed Postgres instance or a properly secured cluster

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
