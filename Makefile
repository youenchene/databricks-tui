.PHONY: build run test test-unit test-arch lint tidy clean snapshot release install-goreleaser

# Build binary
build:
	go build -o bin/databricks-tui ./cmd/tui

# Run with default profile
run: build
	./bin/databricks-tui

# Run with specific profile
run-profile: build
	./bin/databricks-tui $(PROFILE)

# Run all tests
test: test-unit test-arch test-adapter

# Unit tests (domain layer)
test-unit:
	go test ./test/unit/... -v -count=1

# Adapter mapping tests
test-adapter:
	go test ./internal/adapters/... -v -count=1

# Architecture tests
test-arch:
	go test ./test/architecture/... -v -count=1

# Tidy modules
tidy:
	go mod tidy

# Lint
lint:
	golangci-lint run ./...

# Install goreleaser (macOS/Linux)
install-goreleaser:
	@which goreleaser >/dev/null 2>&1 && echo "goreleaser already installed" || \
	(echo "Installing goreleaser..." && go install github.com/goreleaser/goreleaser/v2@latest)

# Build snapshot for local platforms (no publishing, no tag needed)
snapshot: install-goreleaser
	@goreleaser build --snapshot --clean --single-target

# Build snapshot for all configured platforms (no publishing)
snapshot-all: install-goreleaser
	goreleaser release --snapshot --clean --skip=publish,validate

# Full release (requires GITHUB_TOKEN and git tag)
release: install-goreleaser test
	@goreleaser release --clean

# Clean build artifacts
clean:
	rm -rf bin/ dist/
