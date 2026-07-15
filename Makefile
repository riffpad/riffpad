.PHONY: install install-go dev dev-local build test lint clean

# Install all JavaScript dependencies and tidy Go modules
install:
	pnpm install
	$(MAKE) install-go

install-go:
	cd apps/api && go mod tidy

# Start all services locally using Docker Compose + dev servers
dev-local:
	docker compose -f infra/docker-compose.yml up -d
	cd apps/app && pnpm dev &
	cd apps/landing && pnpm dev &
	cd apps/api && go run ./cmd/api

# Start only the app, landing, and api without Docker Compose
# Assumes Postgres/Redis are running elsewhere (e.g. Supabase)
dev:
	cd apps/app && pnpm dev &
	cd apps/landing && pnpm dev &
	cd apps/api && go run ./cmd/api

# Build all applications
build:
	cd apps/app && pnpm build
	cd apps/landing && pnpm build
	cd apps/api && go build -o bin/api ./cmd/api

# Run all tests
test:
	cd apps/app && pnpm test
	cd apps/landing && pnpm test
	cd apps/api && go test ./...

# Run all linters
lint:
	cd apps/app && pnpm lint
	cd apps/landing && pnpm lint

# Stop local Docker Compose services
down:
	docker compose -f infra/docker-compose.yml down

# Clean build artifacts and dependencies
clean:
	rm -rf apps/app/.next apps/app/out
	rm -rf apps/landing/.next apps/landing/out
	rm -rf apps/api/bin
	find . -type d -name node_modules -prune -exec rm -rf {} \;
