.PHONY: build test test-tlock test-e2e test-e2e-headed lint clean install wasm wasm-cjk build-cjk ts build-all bump man html serve demo demo-tlock generate-fixtures full update-pdf-png screenshots release check-translations

BINARY := rememory
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"

# Prefer the esbuild (and other npm tools) installed by package.json in
# node_modules/.bin; fall back to a system-provided one (e.g. Nix devShell).
export PATH := $(CURDIR)/node_modules/.bin:$(PATH)

# File opener for the demo targets: xdg-open on Linux, open on macOS.
# If neither exists, just print the bundle path instead of failing.
OPEN := $(shell command -v xdg-open || command -v open || echo echo)

# Build WASM module first, then the main binary
build: wasm
	go build $(LDFLAGS) -o $(BINARY) ./cmd/rememory

# Compile TypeScript to JavaScript (bundled as IIFE for inline use)
# Uses --loader:.txt=text to bundle BIP39 wordlists as strings
#
# Network-posture model:
#   shared.js        — no network, used by all pages
#   app.js           — __TLOCK__=false: offline recovery, no tlock/drand code
#   app-tlock.js     — __TLOCK__=true:  recovery with tlock (HTTP to drand for decryption)
#   create-app.js    — __SELFHOSTED__=false: bundle creation, tlock encryption is offline
#   create-app-selfhosted.js — __SELFHOSTED__=true: + server integration
#
# tlock-create.ts is gone — encryption functions are inline in create-app.ts
# using the offline drand client (zero HTTP calls).
# tlock-recover.ts is imported by app.ts behind __TLOCK__ guards.
ts:
	@if [ ! -d node_modules ]; then echo "Run 'npm install' first"; exit 1; fi
	@echo "Compiling TypeScript..."
	esbuild internal/html/assets/src/shared.ts --bundle --format=iife --global-name=_shared --outfile=internal/html/assets/shared.js --target=es2020
	esbuild internal/html/assets/src/app.ts --bundle --format=iife --define:__TLOCK__=false --minify-syntax --outfile=internal/html/assets/app.js --target=es2020 --loader:.txt=text --conditions=zbar-inlined
	esbuild internal/html/assets/src/app.ts --bundle --format=iife --define:__TLOCK__=true --minify-syntax --outfile=internal/html/assets/app-tlock.js --target=es2020 --loader:.txt=text --conditions=zbar-inlined
	esbuild internal/html/assets/src/create-app.ts --bundle --format=iife --define:__SELFHOSTED__=false --minify-syntax --outfile=internal/html/assets/create-app.js --target=es2020
	@echo "Compiling selfhosted TypeScript variant..."
	esbuild internal/html/assets/src/create-app.ts --bundle --format=iife --define:__SELFHOSTED__=true --minify-syntax --outfile=internal/html/assets/create-app-selfhosted.js --target=es2020

# Build WASM module for maker.html (bundle creation tool)
# Note: recover.html uses native JavaScript crypto, no WASM needed
wasm: ts
	@mkdir -p internal/html/assets
	@echo "Building create.wasm (bundle creation for maker.html)..."
	GOOS=js GOARCH=wasm go build -tags create -o internal/html/assets/create.wasm ./internal/wasm
	@if [ ! -f internal/html/assets/wasm_exec.js ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		echo "Warning: wasm_exec.js not found"; \
	fi

# Build WASM with CJK font support (produces a larger ~21 MB maker.html).
# After running this, use 'make build-cjk' to build the CLI that embeds it.
wasm-cjk: ts
	@mkdir -p internal/html/assets
	@echo "Building create-cjk.wasm (bundle creation with CJK font support)..."
	GOOS=js GOARCH=wasm go build -tags "create cjk" -o internal/html/assets/create-cjk.wasm ./internal/wasm
	@if [ ! -f internal/html/assets/wasm_exec.js ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" internal/html/assets/ 2>/dev/null || \
		echo "Warning: wasm_exec.js not found"; \
	fi

# Build CLI that generates CJK-capable maker.html (requires 'make wasm-cjk' first).
# The resulting binary produces a ~21 MB maker.html with Chinese/Japanese/Korean support.
# Usage: make wasm-cjk && make build-cjk && ./rememory-cjk html create > maker-cjk.html
build-cjk: wasm-cjk
	go build $(LDFLAGS) -tags cjk -o $(BINARY)-cjk ./cmd/rememory

install: wasm
	go install $(LDFLAGS) ./cmd/rememory

test:
	@test -f internal/html/assets/app.js && test -f $(BINARY) || $(MAKE) build
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

# Run tlock integration tests (requires internet; drand network access)
test-tlock: build
	REMEMORY_TEST_TLOCK=1 go test -v -run TestTlock ./...
	@if [ ! -d node_modules ]; then echo "Run 'npm install' first"; exit 1; fi
	REMEMORY_TEST_TLOCK=1 REMEMORY_BIN=./$(BINARY) npx playwright test

# Clean rebuild + all tests (unit + e2e + tlock)
full: clean build test test-e2e test-tlock lint

lint:
	go vet ./...
	test -z "$$(gofmt -w .)"
	npx tsc --noEmit --project internal/html/assets/tsconfig.json

clean:
	rm -f $(BINARY) $(BINARY)-cjk coverage.out coverage.html
	rm -f internal/html/assets/recover.wasm internal/html/assets/create.wasm internal/html/assets/create-cjk.wasm
	rm -f internal/html/assets/app.js internal/html/assets/app-tlock.js internal/html/assets/create-app.js internal/html/assets/shared.js internal/html/assets/types.js internal/html/assets/create-app-selfhosted.js
	rm -rf dist/ man/
	go clean -testcache

# Generate man pages
man: build
	@mkdir -p man
	./$(BINARY) doc man
	@echo "View with: man ./man/rememory.1"

# Generate standalone HTML files for static hosting
html: build
	./$(BINARY) html site -o dist

# Preview the website locally
serve: html
	@echo "Serving at http://localhost:8000"
	@cd dist && python3 -m http.server 8000

# Run demo: clean, build, and create a demo project
demo: build
	rm -rf demo-recovery
	./$(BINARY) demo --pages
	$(OPEN) demo-recovery/output/bundles/bundle-alice.zip

# Run demo with a 5-minute time lock
demo-tlock: build
	rm -rf demo-recovery
	./$(BINARY) demo --timelock 5min
	$(OPEN) demo-recovery/output/bundles/bundle-alice.zip

# Check that all languages have the same translation keys as English
check-translations:
	REMEMORY_CHECK_TRANSLATIONS=1 go test -v -run TestAllLanguagesHaveSameKeys ./internal/translations/

# Regenerate golden test fixtures (one-time, output is committed)
generate-fixtures:
	go test -v -run TestGenerateGoldenFixtures ./internal/core/ -args -generate

# Cross-compile for all platforms (used by CI)
build-all: wasm
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-linux-amd64 ./cmd/rememory
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/rememory-linux-arm64 ./cmd/rememory
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-darwin-amd64 ./cmd/rememory
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/rememory-darwin-arm64 ./cmd/rememory
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/rememory-windows-amd64.exe ./cmd/rememory

# Stamp CHANGELOG, update VERSION, commit. Asks which bump type.
release:
	@current=$$(cat VERSION 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | cut -d. -f1 | tr -d v); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	echo "Current version: $$current"; \
	echo "  1) patch — v$$major.$$minor.$$((patch + 1))"; \
	echo "  2) minor — v$$major.$$((minor + 1)).0"; \
	echo "  3) major — v$$((major + 1)).0.0"; \
	printf "Which bump? [1] "; \
	read choice; \
	case "$${choice:-1}" in \
		1) new="v$$major.$$minor.$$((patch + 1))" ;; \
		2) new="v$$major.$$((minor + 1)).0" ;; \
		3) new="v$$((major + 1)).0.0" ;; \
		*) echo "Invalid choice"; exit 1 ;; \
	esac; \
	today=$$(date +%Y-%m-%d); \
	if ! grep -q '^## Unreleased' CHANGELOG.md; then \
		echo "No Unreleased section found in CHANGELOG.md"; exit 1; \
	fi; \
	perl -i -pe "s/^## Unreleased$$/## Unreleased\n\n## $$new — $$today/" CHANGELOG.md; \
	printf '%s\n' "$$new" > VERSION; \
	git add CHANGELOG.md VERSION; \
	git commit -m "Release $$new"; \
	echo "Done. Run 'make bump' to tag and push."

# Tag the current VERSION and optionally push.
bump:
	@version=$$(cat VERSION 2>/dev/null); \
	if [ -z "$$version" ]; then echo "VERSION file missing"; exit 1; fi; \
	echo "Tagging $$version"; \
	git tag -a $$version -m "Release $$version"; \
	printf "Push to origin? [Y/n] "; \
	read push; \
	case "$${push:-y}" in \
		[Yy]*) git push origin $$version ;; \
		*) echo "Tag created locally. Push with: git push origin $$version" ;; \
	esac

# Generate PNG screenshots from demo PDF pages (requires pdftoppm from poppler)
update-pdf-png: build
	@rm -rf demo-recovery
	./$(BINARY) demo
	@mkdir -p docs/screenshots/demo-pdf docs/screenshots/demo-pdf-es
	@rm -f docs/screenshots/demo-pdf/*.png docs/screenshots/demo-pdf-es/*.png
	@unzip -o demo-recovery/output/bundles/bundle-alice.zip README.pdf -d demo-recovery/output/bundles/bundle-alice/
	@unzip -o demo-recovery/output/bundles/bundle-camila.zip LEEME.pdf -d demo-recovery/output/bundles/bundle-camila/
	pdftoppm -png -r 200 demo-recovery/output/bundles/bundle-alice/README.pdf docs/screenshots/demo-pdf/page
	pdftoppm -png -r 200 demo-recovery/output/bundles/bundle-camila/LEEME.pdf docs/screenshots/demo-pdf-es/page
	@echo "Generated PDF page screenshots in docs/screenshots/demo-pdf/ (English) and docs/screenshots/demo-pdf-es/ (Spanish)"

# Generate localized guide screenshots via Playwright (en, es, de, fr)
screenshots: build
	@if [ ! -d node_modules ]; then echo "Run 'npm install' first"; exit 1; fi
	REMEMORY_BIN=./$(BINARY) REMEMORY_TEST_SCREENSHOTS=1 npx playwright test --project=chromium
