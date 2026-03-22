# ATK Tracker Monorepo

This repository contains the ATK Tracker platform.

## Layout

- client: Linux daemon that monitors HID activity and syncs heartbeat telemetry.
- server: REST API and nightly rollup worker for attendance analytics.
- admin-frontend: React + TypeScript dashboard for live and historical views.
- shared: Shared contracts/types used by Go services.

## Quick Start

1. Start PostgreSQL and create a database.
2. Run SQL migrations from server/migrations.
3. Build and run the server.
4. Build and run the client daemon (typically managed by systemd on Linux hosts).
5. Start the frontend development server.
