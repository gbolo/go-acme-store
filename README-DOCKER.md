# Docker Quick Start

## Build and Run

```bash
# 1. Build the image
make docker-build

# 2. Create environment file
cp env.example .env
nano .env  # Edit with your configuration

# 3. Start the stack (includes Vault)
make docker-run

# 4. Access the application
# API: http://localhost:8080/api
# UI:  http://localhost:8080/ui
```

## Image Details

- **Base Image**: Alpine Linux 3.21
- **Size**: ~27 MB
- **User**: Non-root (acme:1000)
- **Security**: Static binary, minimal dependencies
- **Health Check**: Built-in at `/api/healthz`

## Configuration

All settings can be configured via:
1. **Environment variables** (recommended for Docker)
2. **Config file** mounted at `/etc/acme-store/config.yml`

See `env.example` for all available environment variables.

## Make Targets

| Command | Description |
|---------|-------------|
| `make docker-build` | Build Docker image |
| `make docker-run` | Start with Docker Compose |
| `make docker-stop` | Stop the stack |
| `make docker-logs` | View container logs |
| `make docker-clean` | Clean up everything |

## Production Deployment

For production deployment examples, see `DOCKER.md` which includes:
- Kubernetes manifests
- Docker Compose production setup
- Resource limits and security best practices

## Verify

```bash
# Check health
curl http://localhost:8080/api/healthz

# View version
curl http://localhost:8080/api/version

# Check configuration
curl http://localhost:8080/api/config
```
