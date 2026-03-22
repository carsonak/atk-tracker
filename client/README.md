# ATK Client (Linux Daemon)

The client monitors local HID activity and emits aggregated heartbeat payloads every 5 minutes.

## Responsibilities

- Discover input devices supporting EV_KEY or EV_REL.
- Parse raw 24-byte Linux input_event records.
- Listen to login1 D-Bus session lifecycle events.
- Aggregate activity into 1-second buckets and 5-minute heartbeats.
- Buffer failed heartbeats in local SQLite and replay when connectivity returns.

## Run Locally

1. Build:

   go build ./cmd/atk-client

2. Run:

   ATK_SERVER_URL=http://127.0.0.1:8080 \
   ATK_MACHINE_ID=zone01-dev-01 \
   ATK_BUFFER_PATH=/tmp/atk-client-buffer.db \
   ./atk-client

## Environment Variables

- ATK_SERVER_URL: API base URL.
- ATK_MACHINE_ID: Machine identifier sent with sessions/heartbeats.
- ATK_BUFFER_PATH: SQLite queue path.
- ATK_HEARTBEAT_WINDOW_SECONDS: Defaults to 300.
- ATK_REQUEST_TIMEOUT_SECONDS: Defaults to 10.
- ATK_FLUSH_INTERVAL_SECONDS: Defaults to 30.

## systemd Service

Service unit file: client/deploy/atk-tracker.service

Install example:

1. sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin atk-tracker || true
2. sudo usermod -aG input atk-tracker
3. sudo install -d -o atk-tracker -g atk-tracker /var/lib/atk-tracker
4. sudo install -m 0755 ./atk-client /usr/local/bin/atk-client
5. sudo install -m 0644 ./deploy/atk-tracker.service /etc/systemd/system/atk-tracker.service
6. sudo systemctl daemon-reload
7. sudo systemctl enable --now atk-tracker.service

## Tests

go test ./...
