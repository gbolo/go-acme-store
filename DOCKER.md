# Docker Deployment Guide

This guide explains how to build and deploy `acme-store` using Docker.

## Quick Start

### 1. Build the Docker Image

```bash
make docker-build
```

Or manually:

```bash
docker build -t acme-store:latest .
```

### 2. Run with Docker Compose

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your configuration
nano .env

# Start the stack (includes Vault)
make docker-run
```

The application will be available at:
- **API**: http://localhost:8080/api
- **Web UI**: http://localhost:8080/ui
- **Vault**: http://localhost:8200

### 3. View Logs

```bash
make docker-logs
```

### 4. Stop the Stack

```bash
make docker-stop
```

## Docker Image Details

### Multi-Stage Build

The Dockerfile uses a multi-stage build process:

1. **Builder Stage**: 
   - Base: `golang:1.25.3-alpine`
   - Compiles the Go binary with optimizations
   - Static binary with no CGO dependencies

2. **Runtime Stage**:
   - Base: `alpine:3.21`
   - Minimal runtime environment
   - Non-root user (`acme:1000`)
   - Includes CA certificates for HTTPS

### Image Size

The final image is approximately **25-30 MB** (compressed).

### Security Features

- ✅ Non-root user execution
- ✅ Minimal attack surface (Alpine Linux)
- ✅ No unnecessary tools or shells
- ✅ Static binary (no dependencies)
- ✅ Health check endpoint
- ✅ Read-only root filesystem compatible

## Configuration

### Environment Variables

All configuration can be set via environment variables:

#### Logging
- `LOG_LEVEL` - Log level (DEBUG, INFO, WARN, ERROR)

#### Server
- `SERVER_BIND_ADDRESS` - Bind address (default: 0.0.0.0)
- `SERVER_BIND_PORT` - Bind port (default: 8080)

#### ACME
- `ACME_DIRECTORY` - ACME server URL
- `ACME_ACCOUNT_EMAIL` - Account email
- `ACME_DNS_SERVER` - DNS server for challenges
- `ACME_DNS_PROVIDER` - DNS provider (digitalocean, acmedns)
- `DIGITALOCEAN_TOKEN` - DigitalOcean API token

#### Vault
- `VAULT_ADDR` - Vault server address
- `VAULT_TOKEN` - Vault authentication token
- `VAULT_KV2_MOUNT` - KV2 mount path
- `VAULT_KV2_SECRET_PATH` - Secret path prefix
- `VAULT_TLS_SKIP_VERIFY` - Skip TLS verification (true/false)

### Config File

Alternatively, mount a config file:

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yml:/etc/acme-store/config.yml:ro \
  -e VAULT_TOKEN=your-token \
  -e DIGITALOCEAN_TOKEN=your-token \
  acme-store:latest
```

## Production Deployment

### Using Docker Compose

```yaml
services:
  acme-store:
    image: acme-store:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      LOG_LEVEL: INFO
      ACME_DIRECTORY: https://acme-v02.api.letsencrypt.org/directory
      ACME_ACCOUNT_EMAIL: admin@example.com
      VAULT_ADDR: https://vault.example.com:8200
      VAULT_TOKEN: ${VAULT_TOKEN}
      DIGITALOCEAN_TOKEN: ${DIGITALOCEAN_TOKEN}
    volumes:
      - ./config.yml:/etc/acme-store/config.yml:ro
    networks:
      - internal
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M
```

### Using Kubernetes

#### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: acme-store
  namespace: acme
spec:
  replicas: 1
  selector:
    matchLabels:
      app: acme-store
  template:
    metadata:
      labels:
        app: acme-store
    spec:
      serviceAccountName: acme-store
      containers:
      - name: acme-store
        image: acme-store:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: LOG_LEVEL
          value: "INFO"
        - name: VAULT_ADDR
          value: "http://vault:8200"
        - name: VAULT_TOKEN
          valueFrom:
            secretKeyRef:
              name: acme-store-secrets
              key: vault-token
        - name: DIGITALOCEAN_TOKEN
          valueFrom:
            secretKeyRef:
              name: acme-store-secrets
              key: digitalocean-token
        livenessProbe:
          httpGet:
            path: /api/healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /api/healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
```

#### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: acme-store
  namespace: acme
spec:
  selector:
    app: acme-store
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

#### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: acme-store
  namespace: acme
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - acme.example.com
    secretName: acme-store-tls
  rules:
  - host: acme.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: acme-store
            port:
              number: 80
```

## Building for Different Architectures

### Multi-Architecture Build

```bash
# Enable BuildKit
export DOCKER_BUILDKIT=1

# Build for multiple platforms
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t acme-store:latest \
  --push \
  .
```

### Build Arguments

```bash
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT_REF=abc123 \
  -t acme-store:1.0.0 \
  .
```

## Health Checks

The container includes a built-in health check:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' acme-store

# Manual health check
curl http://localhost:8080/api/healthz
```

## Troubleshooting

### View Container Logs

```bash
docker logs acme-store
docker logs -f acme-store  # Follow logs
```

### Exec into Container

```bash
docker exec -it acme-store sh
```

### Check Environment

```bash
docker exec acme-store env
```

### Verify Configuration

```bash
docker exec acme-store curl http://localhost:8080/api/config
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make docker-build` | Build Docker image |
| `make docker-run` | Start Docker Compose stack |
| `make docker-stop` | Stop Docker Compose stack |
| `make docker-logs` | View container logs |
| `make docker-clean` | Remove containers and images |
| `make docker-push` | Push image to registry |

## Security Considerations

1. **Never commit secrets** to the repository
2. **Use secrets management** (Vault, Kubernetes secrets, etc.)
3. **Enable TLS** for production deployments
4. **Use read-only root filesystem** when possible
5. **Scan images** for vulnerabilities regularly
6. **Keep base images updated** (Alpine Linux)

## Performance Tuning

### Resource Limits

Recommended resource allocation:
- **CPU**: 0.25-0.5 cores (requests), 0.5-1.0 cores (limits)
- **Memory**: 64-128 MB (requests), 128-256 MB (limits)

### Logging

For high-traffic environments, consider:
- Setting `LOG_LEVEL=WARN` or `LOG_LEVEL=ERROR`
- Using structured logging output
- Implementing log aggregation (ELK, Loki, etc.)

## Examples

See the `examples/` directory for:
- `docker-compose.production.yml` - Production-ready compose file
- `kubernetes/` - Complete Kubernetes manifests
- `helm/` - Helm chart for Kubernetes deployment

