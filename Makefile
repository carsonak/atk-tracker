.PHONY: test-client test-server build-client build-server db-migrate db-seed-demo run-server run-client run-frontend

ATK_DATABASE_URL ?= postgres://localhost:5432/atk_tracker?sslmode=disable

test-client:
	cd client && go test ./...

test-server:
	cd server && go test ./...

build-client:
	cd client && go build ./cmd/atk-client

build-server:
	cd server && go build ./cmd/atk-server

db-migrate:
	psql "$(ATK_DATABASE_URL)" -v ON_ERROR_STOP=1 -f server/migrations/001_init.sql
	psql "$(ATK_DATABASE_URL)" -v ON_ERROR_STOP=1 -f server/migrations/002_ttl_cleanup.sql

db-seed-demo:
	chmod +x server/scripts/load_demo_data.sh
	ATK_DATABASE_URL="$(ATK_DATABASE_URL)" ./server/scripts/load_demo_data.sh

run-server:
	cd server && ATK_DATABASE_URL="$(ATK_DATABASE_URL)" go run ./cmd/atk-server

run-client:
	cd client && ATK_SERVER_URL="http://127.0.0.1:8080" ATK_MACHINE_ID="zone01-dev-01" ATK_BUFFER_PATH="/tmp/atk-client-buffer.db" go run ./cmd/atk-client

run-frontend:
	cd admin-frontend && VITE_API_URL="http://127.0.0.1:8080" npm run dev
