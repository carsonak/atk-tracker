# ATK Server

Go API + rollup worker backed by PostgreSQL.

## Environment Variables

- ATK_DATABASE_URL: PostgreSQL DSN.
- ATK_SERVER_ADDR: Bind address (default :8080).

Example:

```bash
export ATK_DATABASE_URL="postgres://atk_user:atk_pass@127.0.0.1:5432/atk_tracker?sslmode=disable"
export ATK_SERVER_ADDR=":8080"
```

## Run

```bash
go run ./cmd/atk-server
```

## Migrations

```bash
psql "$ATK_DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/001_init.sql
psql "$ATK_DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/002_ttl_cleanup.sql
```

## Demo Data

```bash
chmod +x scripts/load_demo_data.sh
ATK_DATABASE_URL="$ATK_DATABASE_URL" ./scripts/load_demo_data.sh
```

Seed script path: server/migrations/003_demo_seed.sql

## API

- POST /sessions
- PUT /sessions/{id}/end
- POST /heartbeats
- GET /live
- GET /stats?apprentice_id=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD

Quick API smoke test:

```bash
curl -sS -X POST http://127.0.0.1:8080/sessions \
	-H "Content-Type: application/json" \
	-d '{"apprentice_id":"demo-anna","machine_id":"zone01-node-01"}'
```

## Rollups

Nightly rollups merge overlapping active windows into a single apprentice timeline,
then persist aggregated minutes to daily_summaries.

Manual rollup verification query:

```bash
psql "$ATK_DATABASE_URL" -c "SELECT summary_date, apprentice_id, total_active_minutes FROM daily_summaries ORDER BY summary_date DESC, apprentice_id LIMIT 20;"
```
