.PHONY: test-client test-server build-client build-server

test-client:
	cd client && go test ./...

test-server:
	cd server && go test ./...

build-client:
	cd client && go build ./cmd/atk-client

build-server:
	cd server && go build ./cmd/atk-server
