#!/usr/bin/env bash
# =============================================================================
# dev-run.sh — arranca el backend Go + Postgres en local con un solo comando.
# =============================================================================
#
# Por qué existe este script:
#   Go no lee `.env` por sí solo (a diferencia de Next.js o Vite). Sin
#   esto, `go run ./cmd/api` falla con "ENV is required" porque
#   `os.Getenv("ENV")` retorna vacío.
#
#   El script hace tres cosas:
#     1. Verifica que existen los dos `.env` necesarios y aborta con un
#        mensaje claro si falta uno.
#     2. Levanta Postgres con `docker compose up -d` si no está corriendo
#        y espera al healthcheck.
#     3. Exporta las variables de `cmd/api/.env` al entorno del proceso
#        antes de `go run` (el binario las lee con `os.Getenv`).
#
# Uso:
#   ./scripts/dev-run.sh
#
# Parar:
#   Ctrl-C (la API) y `docker compose down` (Postgres).
#
# Reset completo (borra el volumen de Postgres):
#   docker compose down -v
# =============================================================================

set -euo pipefail

# ── Paths ─────────────────────────────────────────────────────────
# El script debe ejecutarse desde la raíz del repo (o con la raíz como
# cwd). Detectamos la raíz buscando el go.mod hacia arriba del cwd para
# que el script sea robusto a "lo ejecuto desde cualquier subdirectorio".
find_repo_root() {
  local dir="${PWD}"
  while [ "${dir}" != "/" ]; do
    if [ -f "${dir}/go.mod" ]; then
      echo "${dir}"
      return 0
    fi
    dir="$(dirname "${dir}")"
  done
  return 1
}

REPO_ROOT="$(find_repo_root)"
cd "${REPO_ROOT}"

# ── Guard clauses ─────────────────────────────────────────────────
# Fail-fast con mensaje humano en vez del críptico "ENV is required"
# que devolvería Go si los secretos no están exportados.
if [ ! -f ".env" ]; then
  echo "❌ Falta .env en la raíz del repo." >&2
  echo "   Cópialo y rellena los secretos:" >&2
  echo "     cp .env.example .env" >&2
  echo "     openssl rand -base64 24   # pega en POSTGRES_PASSWORD" >&2
  exit 1
fi

if [ ! -f "cmd/api/.env" ]; then
  echo "❌ Falta cmd/api/.env." >&2
  echo "   Cópialo y rellena los secretos:" >&2
  echo "     cp cmd/api/.env.example cmd/api/.env" >&2
  echo "     openssl rand -base64 48   # pega en JWT_SECRET" >&2
  echo "     openssl rand -base64 24   # pega en DB_PASSWORD" >&2
  exit 1
fi

# ── Postgres ──────────────────────────────────────────────────────
# `docker compose ps --status running` filtra por servicio + estado.
# Sin grep, devolvería el header. Si el contenedor ya está corriendo,
# no hacemos nada (idempotente).
if docker compose ps --status running postgres 2>/dev/null | grep -qE 'postgres\s+'; then
  echo "▶ Postgres ya está corriendo"
else
  echo "▶ Iniciando Postgres..."
  docker compose up -d postgres

  # Esperar al healthcheck. `pg_isready` retorna 0 cuando la BD acepta
  # conexiones; un timeout de 60s evita colgarse si algo va mal.
  echo "  Esperando a que Postgres acepte conexiones..."
  for i in $(seq 1 60); do
    if docker compose exec -T postgres pg_isready -U dogpaw_user -d dogpaw_db >/dev/null 2>&1; then
      echo "  ✓ Postgres listo (intento ${i})"
      break
    fi
    sleep 1
    if [ "${i}" -eq 60 ]; then
      echo "❌ Postgres no respondió en 60s. Revisa 'docker compose logs postgres'." >&2
      exit 1
    fi
  done
fi

# ── Backend ────────────────────────────────────────────────────────
# `set -a` (alias de `allexport`) marca como export cualquier variable
# que se asigne en el shell a partir de aquí. `source cmd/api/.env` las
# carga en el entorno del proceso. `set +a` revierte el modo. Sin esto
# Go vería `os.Getenv("ENV") == ""` y LoadConfig fallaría.
echo "▶ Iniciando API (Go)..."
echo "  Variables cargadas desde cmd/api/.env"
echo "  Ctrl-C para parar"

set -a
# shellcheck disable=SC1091
source "cmd/api/.env"
set +a

# `exec` reemplaza el shell actual por el binario de Go. Así Ctrl-C va
# directamente al proceso de Go y el shell no se queda colgado.
exec go run ./cmd/api
