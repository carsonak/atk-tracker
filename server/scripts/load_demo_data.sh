#!/usr/bin/env bash
set -euo pipefail

DB_URL="${ATK_DATABASE_URL:-postgres://localhost:5432/atk_tracker?sslmode=disable}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

run_sql() {
  local file="$1"
  echo "Applying ${file}"
  psql "${DB_URL}" -v ON_ERROR_STOP=1 -f "${file}"
}

run_sql "${ROOT_DIR}/server/migrations/001_init.sql"
run_sql "${ROOT_DIR}/server/migrations/002_ttl_cleanup.sql"
run_sql "${ROOT_DIR}/server/migrations/003_demo_seed.sql"

echo "Demo data loaded successfully."
