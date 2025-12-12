# Quick Start Guide

## Prerequisites

1. **HashiCorp Vault** running and accessible
   ```bash
   # Example: Run Vault in dev mode (for testing only)
   vault server -dev
   ```

2. **Vault KV v2 Secrets Engine** enabled
   ```bash
   vault secrets enable -path=kv kv-v2
   ```

3. **DigitalOcean API Token** (for DNS challenges)
   - Get your token from: https://cloud.digitalocean.com/account/api/tokens

## Setup Steps

### 1. Configure the Application

Edit `config.yml`:

```yaml
server:
  bind_address: 127.0.0.1
  bind_port: 15872

acme:
  # Use staging for testing, production for real certs
  directory: "https://acme-staging-v02.api.letsencrypt.org/directory"
  account_email: your-email@example.com
  dns_server: 8.8.8.8:53
  dns_provider: digitalocean

vault:
  address: "http://127.0.0.1:8200"
  kv2_mount: kv
  kv2_secret_path: platform/acme
  tls_skip_verify: false  # Set to true for self-signed certs

# Add your domains here
domains:
  - example.com
  - "*.example.com"
```

### 2. Set Environment Variables

```bash
# Vault authentication token
export VAULT_TOKEN="your-vault-token"

# DigitalOcean API token for DNS challenges
export DIGITALOCEAN_TOKEN="dop_v1_xxxxxxxxxxxxx"
```

### 3. Build the Application

```bash
go build -o acme-store ./cmd/acme-store
```

### 4. Run the Application

```bash
./acme-store
```

Or specify a custom config file:

```bash
./acme-store -config /path/to/config.yml
```

You should see output like:
```
INFO vault client was initialized
INFO loaded 2 domain(s) from configuration: [example.com *.example.com]
INFO server listening on 127.0.0.1:15872
INFO web UI available at http://127.0.0.1:15872
```

### 5. Access the Web UI

Open your browser and navigate to:
```
http://127.0.0.1:15872
```

## Web UI Features

### Dashboard
- **Total Certificates**: Shows the number of managed certificates
- **Valid**: Certificates that are currently valid and not expiring soon
- **Expiring Soon**: Certificates expiring within 30 days
- **Expired**: Certificates that have already expired

### Certificate Cards
Each certificate displays:
- Common Name (CN)
- Issuer information
- Issue date and expiration date
- Days until expiration (color-coded)
- Subject Alternative Names (SANs)
- Status badge (Valid/Expiring Soon/Expired)

### Actions
- **View Certificate**: Display the PEM-encoded leaf certificate
- **View Full Chain**: Display the complete certificate chain
- **Copy to Clipboard**: Copy certificates for use in other applications
- **Refresh**: Reload certificate data from Vault

## Testing with Staging Environment

For testing, use Let's Encrypt staging environment:

```yaml
acme:
  directory: "https://acme-staging-v02.api.letsencrypt.org/directory"
```

**Note**: Staging certificates are not trusted by browsers but are useful for testing the ACME flow without hitting rate limits.

## Production Use

For production certificates, update to:

```yaml
acme:
  directory: "https://acme-v02.api.letsencrypt.org/directory"
```

**Important**: 
- Ensure your DNS provider credentials are correct
- Verify domains are properly configured in DNS
- Monitor certificate expiration dates
- Let's Encrypt has rate limits (50 certificates per domain per week)

## Troubleshooting

### Application won't start
- Check Vault is running and accessible
- Verify `VAULT_TOKEN` is set and valid
- Ensure Vault KV v2 engine is enabled at the configured path

### Certificates not being issued
- Verify `DIGITALOCEAN_TOKEN` is set and valid
- Check domain ownership and DNS configuration
- Review application logs for ACME challenge errors
- Ensure DNS provider supports the domains you're requesting

### Web UI shows no certificates
- Check if domains are configured in `config.yml`
- Verify certificates have been successfully issued (check logs)
- Try the `/certs` API endpoint directly: `curl http://127.0.0.1:15872/certs`

### Certificate renewal not working
- The daemon checks certificates every 24 hours
- Certificates are renewed when they have 21 days or less until expiration
- Check application logs for renewal attempts and errors

### Vault TLS certificate verification issues
- If using Vault with HTTPS and self-signed certificates, set `vault.tls_skip_verify: true`
- Warning: Only use this in development/testing environments
- For production, use properly signed certificates or add your CA to the system trust store

## API Endpoints

Test the API directly:

```bash
# Get all certificates
curl http://127.0.0.1:15872/certs | jq

# Health check
curl http://127.0.0.1:15872/healthz

# Version info
curl http://127.0.0.1:15872/version

# Metrics
curl http://127.0.0.1:15872/metrics
```

## Next Steps

1. Add your production domains to `config.yml`
2. Switch to production ACME directory
3. Set up monitoring for certificate expiration
4. Configure automated deployment of renewed certificates
5. Set up proper authentication/authorization for the web UI
6. Use HTTPS with a reverse proxy (nginx, Caddy, etc.)

## Support

For issues and questions:
- Check application logs for detailed error messages
- Review Vault audit logs if authentication issues occur
- Verify DNS provider API status
- Check Let's Encrypt status page: https://letsencrypt.status.io/

