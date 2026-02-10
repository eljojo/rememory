.PHONY: build test test-e2e test-e2e-headed lint clean install wasm ts build-all bump-patch bump-minor bump-major man html serve demo

BINARY := rememory
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Build WASM module first, then the main binary
build: wasm
	go build $(LDFLAGS) -o $(BINARY) ./cmd/rememory

# Compile TypeScript to JavaScript (bundled as IIFE for inline use)
ts:
	@echo "Compiling TypeScript..."
	esbuild internal/html/assets/src/shared.ts --bundle --format=iife --global-name=_shared --outfile=internal/html/assets/shared.js --target=es2020
	esbuild internal/html/assets/src/app.ts --bundle --format=iife --outfile=internal/html/assets/app.js --target=es2020
	esbuild internal/html/assets/src/create-app.ts --bundle --format=iife --outfile=internal/html/assets/create-app.js --target=es2020

# Build WASM modules
# - recover.wasm: Small, recovery-only (for bundles)
# - create.wasm: Full, includes bundle creation logic (for maker.html)
wasm: ts
	@mkdir -p internal/html/assets
	@echo "Building recover.wasm (recovery only)..."
	GOOS=js GOARCH=wasm go build -o internal/html/assets/recover.wasm ./internal/wasm
	@echo "Building create.wasm (full bundle creation)..."
	GOOS=js GOARCH=wasm go build -tags create -o internal/html/assets/create.wasm ./internal/wasm
	@if [ ! -f internal/html/assets/wasm_exec.js ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		echo "Warning: wasm_exec.js not found"; \
	fi

install: wasm
	go install $(LDFLAGS) ./cmd/rememory

test:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run Playwright e2e tests (requires npm install first)
test-e2e: build
	@if [ ! -d node_modules ]; then echo "Run 'npm install' first"; exit 1; fi
	REMEMORY_BIN=./$(BINARY) npx playwright test

# Run e2e tests with visible browser
test-e2e-headed: build
	@if [ ! -d node_modules ]; then echo "Run 'npm install' first"; exit 1; fi
	REMEMORY_BIN=./$(BINARY) npx playwright test --headed

lint:
	go vet ./...
	test -z "$$(gofmt -w .)"

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -f internal/html/assets/recover.wasm internal/html/assets/create.wasm
	rm -f internal/html/assets/app.js internal/html/assets/create-app.js internal/html/assets/shared.js internal/html/assets/types.js
	rm -rf dist/ man/

# Generate man pages
man: build
	@mkdir -p man
	./$(BINARY) doc man
	@echo "View with: man ./man/rememory.1"

# Generate standalone HTML files for static hosting
html: build
	@mkdir -p dist/screenshots
	./$(BINARY) html index > dist/index.html
	./$(BINARY) html create > dist/maker.html
	./$(BINARY) html docs > dist/docs.html
	./$(BINARY) html recover > dist/recover.html
	@cp docs/screenshots/*.png dist/screenshots/ 2>/dev/null || true
	@echo "Generated dist/ site"

# Preview the website locally
serve: html
	@echo "Serving at http://localhost:8000"
	@cd dist && python3 -m http.server 8000

# Run demo: clean, build, and create a demo project
demo: build
	rm -rf demo-recovery
	./$(BINARY) demo
	open demo-recovery/output/bundles/bundle-alice.zip

# Cross-compile for all platforms (used by CI)
build-all: wasm
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-linux-amd64 ./cmd/rememory
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/rememory-linux-arm64 ./cmd/rememory
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-darwin-amd64 ./cmd/rememory
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/rememory-darwin-arm64 ./cmd/rememory
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-windows-amd64.exe ./cmd/rememory

# Bump version tags (usage: make bump-patch, bump-minor, bump-major)
bump-patch:
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | cut -d. -f1 | tr -d v); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	new="v$$major.$$minor.$$((patch + 1))"; \
	echo "Bumping $$current -> $$new"; \
	git tag -a $$new -m "Release $$new"; \
	echo ""; \
	read -p "Push tag $$new to origin? [y/N] " answer; \
	if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
		git push origin $$new; \
	else \
		echo "Tag created locally. Push with: git push origin $$new"; \
	fi

bump-minor:
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | cut -d. -f1 | tr -d v); \
	minor=$$(echo $$current | cut -d. -f2); \
	new="v$$major.$$((minor + 1)).0"; \
	echo "Bumping $$current -> $$new"; \
	git tag -a $$new -m "Release $$new"; \
	echo ""; \
	read -p "Push tag $$new to origin? [y/N] " answer; \
	if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
		git push origin $$new; \
	else \
		echo "Tag created locally. Push with: git push origin $$new"; \
	fi

bump-major:
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | cut -d. -f1 | tr -d v); \
	new="v$$((major + 1)).0.0"; \
	echo "Bumping $$current -> $$new"; \
	git tag -a $$new -m "Release $$new"; \
	echo ""; \
	read -p "Push tag $$new to origin? [y/N] " answer; \
	if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
		git push origin $$new; \
	else \
		echo "Tag created locally. Push with: git push origin $$new"; \
	fi
