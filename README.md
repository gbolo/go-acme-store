# ACME Certificate Store

A Go application that automates ACME certificate management and stores certificates in HashiCorp Vault with a beautiful web UI for monitoring.

## Features

- 🔐 **Automated ACME Certificate Management**: Automatically obtains and renews SSL/TLS certificates using the ACME protocol
- 🏦 **HashiCorp Vault Integration**: Securely stores certificates and private keys in Vault's KV v2 secrets engine
- 🌐 **Modern Web UI**: Beautiful, responsive web interface to monitor and manage certificates
- ⚙️ **Domain Management**: Add/remove domains via Web UI or REST API without restarting
- 🔄 **Automatic Renewal**: Monitors certificate expiration and automatically renews certificates
- 📊 **Real-time Statistics**: Dashboard showing certificate status, expiration dates, and health metrics
- 🔄 **Status Tracking**: Real-time status updates (Pending → Issued/Failed) with error reporting
- 🔍 **DNS Challenge Support**: Currently supports DigitalOcean DNS provider for DNS-01 challenges
- 📱 **Responsive Design**: Works seamlessly on desktop and mobile devices

## Web UI

The application includes a modern web interface accessible at the root URL (e.g., `http://localhost:15872/`).

### Features:
- **Certificate Dashboard**: View all managed certificates at a glance
- **Status Indicators**: Visual indicators for valid, expiring soon, and expired certificates
- **Certificate Details**: View detailed information including:
  - Common Name and SANs (Subject Alternative Names)
  - Issuer information
  - Issue and expiration dates
  - Days until expiration
- **Certificate Viewer**: View and copy PEM-encoded certificates
- **Chain Viewer**: View complete certificate chains
- **Real-time Updates**: Refresh button to fetch latest certificate data

## Configuration

Edit `config.yml` to configure the application:

```yaml
log:
  level: INFO

server:
  bind_address: 127.0.0.1
  bind_port: 15872

acme:
  # ACME directory server to use
  directory: "https://acme-staging-v02.api.letsencrypt.org/directory"
  # ACME account to use (private key will get created if it does not exist)
  account_email: your-email@example.com
  # DNS server to use for validating dns challenge
  dns_server: 8.8.8.8:53
  # currently ONLY digitalocean is supported
  dns_provider: digitalocean

vault:
  address: "http://127.0.0.1:8200"
  # path where a KV (v2) secrets engine is mounted
  kv2_mount: kv
  # path (relative to mount) where we want to store our secrets
  kv2_secret_path: platform/acme
  # skip TLS certificate verification (useful for self-signed certs in dev/test)
  tls_skip_verify: false
```

**Note**: Domains are now managed via Vault using the API endpoints (see API section below).

### Environment Variables

Set these environment variables for sensitive data:

- `VAULT_TOKEN`: HashiCorp Vault authentication token
- `DIGITALOCEAN_TOKEN`: DigitalOcean API token for DNS challenges
- `CONFIG_FILE`: Path to configuration file (default: `./config.yml`)

## API Endpoints

### Web UI
- `GET /` - Redirects to `/ui`
- `GET /ui` - Web UI dashboard
- `GET /ui/static/*` - Static assets (CSS, JS, images)

### API - Certificates
- `GET /api/certs` - JSON API returning all certificates with details

### API - Domain Management
- `GET /api/domains` - List all managed domains
- `POST /api/domains` - Add a domain to manage
  - Body: `{"domain": "example.com", "sans": ["www.example.com", "api.example.com"]}`
  - SANs are optional
- `DELETE /api/domains/{domain}` - Remove domain (soft delete - marks as unmanaged)
- `DELETE /api/domains/{domain}?delete_cert=true` - Remove domain and delete certificate

### API - ACME Operations
- `POST /api/trigger-renewal` - Manually trigger certificate issuance/renewal for all managed domains

### API - System
- `GET /api/healthz` - Health check endpoint
- `GET /api/version` - Application version information
- `GET /api/config` - View non-sensitive configuration

### Monitoring
- `GET /metrics` - Server metrics and monitoring

## Building and Running

### Prerequisites

- Go 1.21 or later
- HashiCorp Vault instance (with KV v2 secrets engine enabled)
- DNS provider credentials (currently DigitalOcean)

### Build

```bash
go build -o acme-store ./cmd/acme-store
```

### Run

```bash
export VAULT_TOKEN="your-vault-token"
export DIGITALOCEAN_TOKEN="your-do-token"
./acme-store
```

Or specify a custom config file:

```bash
./acme-store -config /path/to/custom-config.yml
```

Command-line flags:
- `-config` - Path to configuration file (overrides CONFIG_FILE env var and default)

The application will:
1. Initialize connection to HashiCorp Vault
2. Load domains from configuration
3. Start the ACME daemon for certificate management
4. Start the HTTP server with web UI

Access the web UI at: `http://127.0.0.1:15872/ui`

## Project Structure

```
.
├── cmd/
│   └── acme-store/          # Main application
│       ├── main.go          # Entry point
│       ├── httpserver.go    # HTTP server and handlers
│       ├── keystore.go      # Keystore initialization
│       └── worker.go        # ACME daemon worker
├── pkg/
│   ├── acme/                # ACME protocol implementation
│   ├── config/              # Configuration management
│   ├── crypto/              # Certificate utilities
│   ├── httpserver/          # HTTP server setup
│   ├── keystore/            # Vault and storage abstractions
│   ├── log/                 # Logging utilities
│   └── meta/                # Application metadata
├── web/
│   └── static/              # Web UI assets
│       ├── index.html       # Main UI page
│       ├── style.css        # Styles
│       └── app.js           # JavaScript application
└── config.yml               # Configuration file
```

## Security Considerations

- Store sensitive credentials in environment variables, not in `config.yml`
- Use HTTPS in production environments
- Restrict access to the web UI using firewall rules or reverse proxy authentication
- Regularly rotate Vault tokens
- Use production ACME servers (not staging) for production certificates

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

