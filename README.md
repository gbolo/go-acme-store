# ACME Certificate Store

Automated certificate management with ACME protocol support, secure storage, and a modern web UI.

## Purpose

ACME Certificate Store automates the entire lifecycle of TLS certificates:
- Obtains certificates from ACME providers (Let's Encrypt, etc.) using DNS-01 challenge.
- Stores certificates and keys securely (Vault, Filesystem, or Memory)
- Monitors expiration and automatically renews certificates
- Provides a web UI and REST API for management
- Includes a fetcher tool for deploying certificates and keys to other systems

## Configuration

Configuration can be provided via:
1. Configuration file (YAML)
2. Environment variables

### Environment Variable Format

Environment variables use the prefix `ACMESTORE_` followed by the config path with underscores:

```bash
# Config file:
acme:
  directory: https://acme-v02.api.letsencrypt.org/directory
  
# Environment variable:
ACMESTORE_ACME_DIRECTORY=https://acme-v02.api.letsencrypt.org/directory
```

**Precedence:** Environment variables override config file values.

### Configuration Options

#### Server
```yaml
server:
  bind_address: 0.0.0.0  # Listen address
  bind_port: 15872       # HTTP port
```

Environment variables:
- `ACMESTORE_SERVER_BIND_ADDRESS`
- `ACMESTORE_SERVER_BIND_PORT`

#### Logging
```yaml
log:
  level: INFO            # DEBUG, INFO, WARN, ERROR
```

Environment variable:
- `ACMESTORE_LOG_LEVEL`

#### ACME
```yaml
acme:
  directory: https://acme-v02.api.letsencrypt.org/directory
  account_email: admin@example.com
  dns_provider: digitalocean
  dns_server: 8.8.8.8:53              # Optional: Custom DNS resolver
  tls_insecure_skip_verify: false     # Only for testing!
  
  # DigitalOcean DNS
  digitalocean_token: your-token
  
  # PowerDNS
  powerdns_server_url: http://pdns:8081
  powerdns_api_token: secret
  powerdns_server_id: localhost
  
  # ACME-DNS
  acmedns_server_url: http://acme-dns:8053
```

Environment variables:
- `ACMESTORE_ACME_DIRECTORY`
- `ACMESTORE_ACME_ACCOUNT_EMAIL`
- `ACMESTORE_ACME_DNS_PROVIDER`
- `ACMESTORE_ACME_DNS_SERVER`
- `ACMESTORE_ACME_TLS_INSECURE_SKIP_VERIFY`
- `ACMESTORE_ACME_DIGITALOCEAN_TOKEN`
- `ACMESTORE_ACME_POWERDNS_SERVER_URL`
- `ACMESTORE_ACME_POWERDNS_API_TOKEN`
- `ACMESTORE_ACME_POWERDNS_SERVER_ID`
- `ACMESTORE_ACME_ACMEDNS_SERVER_URL`

#### Keystore Backend

The keystore backend determines where certificates and keys are stored. Three backends are supported:

```yaml
keystore:
  backend: vault  # Options: vault, filesystem, memory
```

**Vault** (default): Production-ready, secure storage in HashiCorp Vault
- Best for: Production deployments, multi-instance setups, HA requirements
- Requires: Running Vault instance with KV v2 secrets engine

**Filesystem**: Simple file-based storage
- Best for: Single-node deployments, development, testing
- Requires: Persistent storage volume
- Data stored in: `<base_path>/account.json`, `<base_path>/domains/*.json`, `<base_path>/managed-domains.json`

```yaml
keystore:
  backend: filesystem

filesystem:
  base_path: /var/lib/acme-store  # Directory for storing certificates and keys
```

**Memory**: In-memory storage (ephemeral)
- Best for: Testing only - all data lost on restart
- Requires: No external dependencies

Environment variable:
- `ACMESTORE_KEYSTORE_BACKEND`
- `ACMESTORE_FILESYSTEM_BASE_PATH`

#### Vault Backend Configuration

Required only when using `keystore.backend: vault`
```yaml
vault:
  address: http://vault:8200
  token: your-token
  kv2_mount: kv
  kv2_secret_path: acme
  tls_skip_verify: false
```

Environment variables:
- `ACMESTORE_VAULT_ADDRESS`
- `ACMESTORE_VAULT_TOKEN`
- `ACMESTORE_VAULT_KV2_MOUNT`
- `ACMESTORE_VAULT_KV2_SECRET_PATH`
- `ACMESTORE_VAULT_TLS_SKIP_VERIFY`

### DNS Provicer Configuration

See [DNS_PROVIDERS.md](DNS_PROVIDERS.md) for DNS provider configuration.

### Example Configuration

See [config.yml](config.yml) for a complete example.

## API Documentation

See [API.md](API.md) for complete REST API documentation.

### Quick API Examples

```bash
# Health check
curl http://localhost:15872/api/healthz

# List certificates
curl http://localhost:15872/api/certs

# Get private key for a domain
curl http://localhost:15872/api/certs/example.com/private-key

# Add domain
curl -X POST http://localhost:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com","sans":["www.example.com"]}'

# Remove domain
curl -X DELETE http://localhost:15872/api/domains/example.com

# Trigger renewal
curl -X POST http://localhost:15872/api/trigger-renewal
```

## Web UI

Access the web interface at `http://localhost:15872/`

Features:
- Certificate dashboard with status indicators
- View certificate details and chains
- Add/remove domains
- Manual renewal triggers
- Real-time status updates

## Building from Source

```bash
# Build both binaries
make build

# Build daemon only
make build-daemon

# Build fetcher only
make build-fetcher

# Run tests
make test

# Run integration tests
make test-all
```

## Docker

### Build Image
```bash
docker build -t acme-store:latest .
```

### Run Container
```bash
docker run -d \
  -p 15872:15872 \
  -v $(pwd)/config.yml:/etc/acme-store/config.yml:ro \
  -e ACMESTORE_VAULT_TOKEN=your-token \
  acme-store:latest
```

## Vault Backend Setup

If using the Vault backend (`keystore.backend: vault`), you need Vault with KV v2 secrets engine:

```bash
# Enable KV v2
vault secrets enable -version=2 -path=kv kv

# Verify
vault secrets list
```

The keystore will store:
- ACME account key and registration
- Domain list (managed domains)
- Certificates and private keys

## Filesystem Backend Setup

If using the Filesystem backend (`keystore.backend: filesystem`), ensure the base path directory exists and has appropriate permissions:

```bash
# Create directory
sudo mkdir -p /var/lib/acme-store

# Set permissions (adjust user as needed)
sudo chown acme-store:acme-store /var/lib/acme-store
sudo chmod 700 /var/lib/acme-store
```

The filesystem keystore will create:
- `account.json` - ACME account information
- `managed-domains.json` - List of managed domains
- `domains/` - Directory containing certificate files (one JSON file per domain)


## acme-store-fetcher (Optional Companion CLI)
- Fetches certificates from the acme-store API
- Deploys to local filesystem
- Generates Traefik dynamic configuration
- Runs as a cron job or one-shot command
- No direct keystore access required

See [FETCHER.md](FETCHER.md) for fetcher documentation.
## License

MIT
