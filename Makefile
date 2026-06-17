.PHONY: build run test test-unit test-arch lint tidy

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

# Clean build artifacts
clean:
	rm -rf bin/
