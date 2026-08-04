.PHONY: dev-daemon dev-relay dev-mobile build-daemon build-relay build test

dev-daemon:
	cd apps/daemon && go run .

dev-relay:
	cd apps/relay && go run .

dev-mobile:
	pnpm --filter mobile dev

build-daemon:
	cd apps/daemon && mkdir -p bin && go build -o bin/riffpad ./cmd/riffpad && go build -o bin/riffpadd ./cmd/riffpadd

build-relay:
	cd apps/relay && go build -o bin/relay .

build:
	$(MAKE) build-daemon
	$(MAKE) build-relay
	pnpm -r build

test:
	cd apps/daemon && go test ./...
	cd packages/protocol && go test ./...
	cd apps/relay && go test ./...
	pnpm -r test
