.PHONY: dev-daemon dev-relay dev-mobile build-daemon build-relay build install-daemon install-relay test

PREFIX ?= $(HOME)/.local/bin

dev-daemon:
	cd apps/daemon && go run .

dev-relay:
	cd apps/relay && go run .

dev-mobile:
	pnpm --filter mobile dev

build-daemon:
	cd apps/daemon && mkdir -p bin && go build -o bin/riffpad ./cmd/riffpad && go build -o bin/riffpadd ./cmd/riffpadd

# Build and install riffpad/riffpadd to $(PREFIX) (default ~/.local/bin) so they
# are available on PATH from any directory. Existing files are overwritten.
install-daemon: build-daemon
	mkdir -p $(PREFIX)
	cp -f apps/daemon/bin/riffpad $(PREFIX)/riffpad
	cp -f apps/daemon/bin/riffpadd $(PREFIX)/riffpadd
	@echo "installed: $(PREFIX)/riffpad $(PREFIX)/riffpadd"

install-relay:
	cd apps/relay && mkdir -p bin && go build -o bin/relay .
	cp -f apps/relay/bin/relay $(PREFIX)/relay
	@echo "installed: $(PREFIX)/relay"

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
