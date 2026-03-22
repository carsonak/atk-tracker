# ATK Tracker Monorepo

ATK Tracker is a client-server attendance telemetry platform for Zone01 clusters.

## Repository Layout

- client: Linux daemon (Go) for HID activity aggregation and heartbeat sync.
- server: REST API + nightly rollup worker (Go, PostgreSQL).
- admin-frontend: React + TypeScript dashboard for live and historical analytics.
- shared: shared Go DTOs, constants, and validation helpers.

## Prerequisites

- Go 1.22+
- Node.js 20+
- npm 10+
- PostgreSQL 14+

You already installed PostgreSQL, so you can use the commands below directly.

## 1) Initialize PostgreSQL

Create the database and ensure your user can connect:

```bash
sudo -u postgres psql -c "CREATE DATABASE atk_tracker;"
sudo -u postgres psql -c "CREATE USER atk_user WITH PASSWORD 'atk_pass';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE atk_tracker TO atk_user;"
```

Export a DSN used by server tooling:

```bash
export ATK_DATABASE_URL="postgres://atk_user:atk_pass@127.0.0.1:5432/atk_tracker?sslmode=disable"
```

## 2) Apply Migrations

```bash
psql "$ATK_DATABASE_URL" -v ON_ERROR_STOP=1 -f server/migrations/001_init.sql
psql "$ATK_DATABASE_URL" -v ON_ERROR_STOP=1 -f server/migrations/002_ttl_cleanup.sql
```

## 3) Load Demo Data

This seeds sessions, raw heartbeats, and daily summaries for apprentice IDs:

- demo-anna
- demo-bao
- demo-caro

```bash
chmod +x server/scripts/load_demo_data.sh
ATK_DATABASE_URL="$ATK_DATABASE_URL" ./server/scripts/load_demo_data.sh
```

You can inspect seeded data quickly:

```bash
psql "$ATK_DATABASE_URL" -c "SELECT apprentice_id, count(*) FROM sessions GROUP BY 1 ORDER BY 1;"
psql "$ATK_DATABASE_URL" -c "SELECT apprentice_id, summary_date, total_active_minutes FROM daily_summaries WHERE apprentice_id LIKE 'demo-%' ORDER BY summary_date DESC LIMIT 12;"
```

## 4) Run Server

```bash
cd server
ATK_DATABASE_URL="$ATK_DATABASE_URL" ATK_SERVER_ADDR=":8080" go run ./cmd/atk-server
```

## 5) Run Client (dev mode)

```bash
cd client
ATK_SERVER_URL="http://127.0.0.1:8080" \
ATK_MACHINE_ID="zone01-dev-01" \
ATK_BUFFER_PATH="/tmp/atk-client-buffer.db" \
go run ./cmd/atk-client
```

## 6) Run Admin Frontend

```bash
cd admin-frontend
npm install
VITE_API_URL="http://127.0.0.1:8080" npm run dev
```

Open http://127.0.0.1:5173 and query a demo apprentice such as demo-anna.

## Build and Test

From repository root:

```bash
make test-client
make test-server
make build-client
make build-server
```

## Component Docs

- client README: client/README.md
- server README: server/README.md
- admin frontend README: admin-frontend/README.md
- shared README: shared/README.md
