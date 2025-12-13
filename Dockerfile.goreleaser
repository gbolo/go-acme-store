# Runtime stage - GoReleaser will provide the pre-built binaries
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# Create non-root user
RUN addgroup -g 1000 acme && \
    adduser -D -u 1000 -G acme acme

# Create necessary directories
RUN mkdir -p /app /app/web /etc/acme-store && \
    chown -R acme:acme /app /etc/acme-store

WORKDIR /app

# Build arg for target platform (provided by buildx)
ARG TARGETPLATFORM

# Copy pre-built binary from goreleaser context
# GoReleaser organizes binaries by platform: linux/amd64/acme-store, linux/arm64/acme-store, etc.
COPY ${TARGETPLATFORM}/acme-store .

# Copy web UI files from goreleaser context
COPY web ./web

# Copy default config (optional)
COPY config.yml /etc/acme-store/config.yml.example

# Switch to non-root user
USER acme

# Expose the default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/healthz || exit 1

# Set default environment variables
ENV CONFIG_FILE=/etc/acme-store/config.yml \
    LOG_LEVEL=INFO \
    SERVER_BIND_ADDRESS=0.0.0.0 \
    SERVER_BIND_PORT=8080

# Run the binary
ENTRYPOINT ["/app/acme-store"]
CMD ["-config", "/etc/acme-store/config.yml"]

