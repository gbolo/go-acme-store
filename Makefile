.PHONY: all build clean install test

# Build variables
BINARY_DAEMON=acme-store
BINARY_FETCHER=acme-store-fetcher
CMD_DAEMON=./cmd/acme-store
CMD_FETCHER=./cmd/acme-store-fetcher
INSTALL_PATH=/usr/local/bin

# Build all binaries
all: build

# Build both binaries
build: build-daemon build-fetcher

# Build daemon
build-daemon:
	@echo "Building $(BINARY_DAEMON)..."
	go build -o $(BINARY_DAEMON) $(CMD_DAEMON)

# Build fetcher
build-fetcher:
	@echo "Building $(BINARY_FETCHER)..."
	go build -o $(BINARY_FETCHER) $(CMD_FETCHER)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_DAEMON) $(BINARY_FETCHER)

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
	go test -v ./...

# Run daemon
run-daemon: build-daemon
	./$(BINARY_DAEMON)

# Run fetcher
run-fetcher: build-fetcher
	./$(BINARY_FETCHER)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build all binaries (default)"
	@echo "  build        - Build all binaries"
	@echo "  build-daemon - Build acme-store daemon only"
	@echo "  build-fetcher- Build acme-store-fetcher only"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install binaries to $(INSTALL_PATH)"
	@echo "  test         - Run tests"
	@echo "  run-daemon   - Build and run daemon"
	@echo "  run-fetcher  - Build and run fetcher"
	@echo "  help         - Show this help message"

