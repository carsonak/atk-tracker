# ATK Server

## Environment

- ATK_DATABASE_URL: PostgreSQL DSN.
- ATK_SERVER_ADDR: Bind address (default :8080).

## API

- POST /sessions
- PUT /sessions/{id}/end
- POST /heartbeats
- GET /live
- GET /stats?apprentice_id=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD

## Rollups

Nightly rollups merge overlapping active windows into a single apprentice timeline,
then persist aggregated minutes to daily_summaries.
