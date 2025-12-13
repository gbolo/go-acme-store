.PHONY: all build clean install test

# Build variables
BINARY_DAEMON=acme-store
BINARY_FETCHER=acme-store-fetcher
CMD_DAEMON=./cmd/acme-store
CMD_FETCHER=./cmd/acme-store-fetcher

# Version information
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "devel")
COMMIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date +%FT%T%z)

# ldflags for version injection
LDFLAGS = -s -w \
	-X go-acme-store/pkg/meta.Name=$(BINARY_DAEMON) \
	-X go-acme-store/pkg/meta.Version=$(VERSION) \
	-X go-acme-store/pkg/meta.CommitSHA=$(COMMIT_SHA) \
	-X go-acme-store/pkg/meta.BuildDate=$(BUILD_DATE)

LDFLAGS_FETCHER = -s -w \
	-X go-acme-store/pkg/meta.Name=$(BINARY_FETCHER) \
	-X go-acme-store/pkg/meta.Version=$(VERSION) \
	-X go-acme-store/pkg/meta.CommitSHA=$(COMMIT_SHA) \
	-X go-acme-store/pkg/meta.BuildDate=$(BUILD_DATE)

# Build all binaries
all: build

# Build both binaries
build: build-daemon build-fetcher

# Build daemon
build-daemon:
	@echo "Building $(BINARY_DAEMON)..."
	@echo "Version: $(VERSION), Commit: $(COMMIT_SHA)"
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_DAEMON) $(CMD_DAEMON)

# Build fetcher
build-fetcher:
	@echo "Building $(BINARY_FETCHER)..."
	@echo "Version: $(VERSION), Commit: $(COMMIT_SHA)"
	go build -ldflags="$(LDFLAGS_FETCHER)" -o $(BINARY_FETCHER) $(CMD_FETCHER)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_DAEMON) $(BINARY_FETCHER)
	rm -rf dist/

# Install binaries to system
install: build
	@echo "Installing to $(INSTALL_PATH)..."
	sudo cp $(BINARY_DAEMON) $(INSTALL_PATH)/
	sudo cp $(BINARY_FETCHER) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_DAEMON)
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_FETCHER)
	@echo "Installation complete!"

# Run tests
test:
	go test -v -count=1 ./...

# Run daemon
run-daemon: build-daemon
	./$(BINARY_DAEMON)

# Run fetcher
run-fetcher: build-fetcher
	./$(BINARY_FETCHER)

# Integration tests
test-setup:
	@echo "Starting test environment..."
	docker compose -f docker-compose.test.yml up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 20
	@echo "Verifying services..."
	@curl -s http://localhost:8200/v1/sys/health > /dev/null && echo "✓ Vault is ready" || echo "✗ Vault not ready"
	@curl -sk https://localhost:14000/dir > /dev/null 2>&1 && echo "✓ Pebble is ready" || echo "✗ Pebble not ready"
	@curl -s http://localhost:8053/health > /dev/null 2>&1 && echo "✓ acme-dns is ready" || echo "✗ acme-dns not ready"
	@curl -s http://localhost:15872/api/healthz > /dev/null 2>&1 && echo "✓ acme-store is ready" || echo "✗ acme-store not ready"; docker logs acme-store-test

test-cleanup:
	@echo "Stopping test environment..."
	docker compose -f docker-compose.test.yml down -v
	@echo "Cleaning up test data..."
	rm -rf testdata/certs/
	@echo "Test environment cleaned up!"

test-integration:
	@echo "Running integration tests..."
	@go test -v -count=1 -tags=integration ./tests/integration/...
	@echo "✓ Integration tests complete!"

test-all: test-cleanup test-setup test-integration test-cleanup

# Docker targets
.PHONY: docker-build docker-run docker-push docker-clean

docker-build:
	@echo "Building Docker image..."
	docker build -t acme-store:latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_REF=$(COMMIT_REF) \
		.
	@echo "✓ Docker image built: acme-store:latest"

docker-run:
	@echo "Running Docker Compose stack..."
	docker compose up -d
	@echo "✓ Stack is running"
	@echo "  • API: http://localhost:8080/api"
	@echo "  • UI:  http://localhost:8080/ui"
	@echo "  • Vault: http://localhost:8200"

docker-stop:
	@echo "Stopping Docker Compose stack..."
	docker compose down
	@echo "✓ Stack stopped"

docker-logs:
	docker compose logs -f acme-store

docker-clean:
	@echo "Cleaning Docker resources..."
	docker compose down -v
	docker rmi acme-store:latest 2>/dev/null || true
	@echo "✓ Docker resources cleaned"

docker-push:
	@echo "Pushing Docker image..."
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "Error: DOCKER_REGISTRY not set"; \
		echo "Usage: make docker-push DOCKER_REGISTRY=your-registry.com"; \
		exit 1; \
	fi
	docker tag acme-store:latest $(DOCKER_REGISTRY)/acme-store:$(VERSION)
	docker tag acme-store:latest $(DOCKER_REGISTRY)/acme-store:latest
	docker push $(DOCKER_REGISTRY)/acme-store:$(VERSION)
	docker push $(DOCKER_REGISTRY)/acme-store:latest
	@echo "✓ Images pushed to $(DOCKER_REGISTRY)"

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Build all binaries (default)"
	@echo "  build            - Build all binaries"
	@echo "  build-daemon     - Build acme-store daemon only"
	@echo "  build-fetcher    - Build acme-store-fetcher only"
	@echo "  clean            - Remove build artifacts"
	@echo "  install          - Install binaries to $(INSTALL_PATH)"
	@echo "  test             - Run unit tests"
	@echo "  test-setup       - Start Docker test environment"
	@echo "  test-cleanup     - Stop and clean Docker test environment"
	@echo "  test-integration - Run integration tests"
	@echo "  test-all         - Run full test suite (setup + test + cleanup)"
	@echo "  run-daemon       - Build and run daemon"
	@echo "  run-fetcher      - Build and run fetcher"
	@echo "  help             - Show this help message"


# Show version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT_SHA)"
	@echo "Build Date: $(BUILD_DATE)"

# GoReleaser targets
.PHONY: release release-snapshot release-check release-build

# Create a release (requires git tag)
release:
	@echo "Creating release with goreleaser..."
	goreleaser release --clean

# Create a snapshot release (no git tag required)
release-snapshot:
	@echo "Creating snapshot release with goreleaser..."
	goreleaser release --snapshot --clean

# Check goreleaser configuration
release-check:
	@echo "Checking goreleaser configuration..."
	goreleaser check

# Build with goreleaser locally
release-build:
	@echo "Building with goreleaser (local only)..."
	goreleaser build --snapshot --clean --single-target
