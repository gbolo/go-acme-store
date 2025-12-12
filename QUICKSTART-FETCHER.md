# Quick Start: acme-store-fetcher

## What is it?

A CLI tool that exports certificates from Vault to disk as PEM files, ready to use with nginx, Apache, HAProxy, etc.

## Quick Start

### 1. Build

```bash
make build-fetcher
# or
go build -o acme-store-fetcher ./cmd/acme-store-fetcher
```

### 2. Configure (Optional)

Add fetcher settings to `config.yml`:

```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"  # optional
```

### 3. Run

```bash
# Basic usage (saves to ./certs or config file value)
./acme-store-fetcher

# Override with command-line flags
./acme-store-fetcher -output-dir /etc/ssl/acme-certs

# Use config file settings
./acme-store-fetcher -config /etc/acme-store/config.yml

# Generate Traefik configuration (via flags)
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml

# Or set in config file and just run
./acme-store-fetcher
```

### 3. Use the Certificates

```bash
# List exported files
ls -lh ./certs/

# Example output:
# example.com_cert-chain.pem  (certificate chain)
# example.com_key.pem         (private key)
# _wild_.example.com_cert-chain.pem
# _wild_.example.com_key.pem
```

## Web Server Examples

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name example.com;
    
    ssl_certificate /etc/ssl/acme-certs/example.com_cert-chain.pem;
    ssl_certificate_key /etc/ssl/acme-certs/example.com_key.pem;
    
    # ... rest of config
}
```

### Traefik (with auto-generated config)

```bash
# Fetch certs and generate Traefik config
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml

# Traefik auto-reloads - no restart needed!
```

**Traefik main config** (`/etc/traefik/traefik.yml`):
```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

## Automation

### Cron (every 6 hours)

```bash
# Add to /etc/cron.d/acme-store-fetcher
0 */6 * * * root /usr/local/bin/acme-store-fetcher -output-dir /etc/ssl/acme-certs && systemctl reload nginx
```

### Systemd Timer

```bash
# Copy example files
sudo cp examples/systemd/acme-store-fetcher.service /etc/systemd/system/
sudo cp examples/systemd/acme-store-fetcher.timer /etc/systemd/system/

# Enable and start
sudo systemctl enable acme-store-fetcher.timer
sudo systemctl start acme-store-fetcher.timer

# Check status
sudo systemctl status acme-store-fetcher.timer
```

## What Gets Exported?

| Certificate Status | Exported? | Reason |
|-------------------|-----------|--------|
| ✅ Issued & Managed | Yes | Ready to use |
| ⏭️ Pending | No | Not yet issued |
| ⏭️ Failed | No | Issuance failed |
| ⏭️ Unmanaged | No | No longer managed |

## File Naming

| Domain | Certificate File | Key File |
|--------|------------------|----------|
| `example.com` | `example.com_cert-chain.pem` | `example.com_key.pem` |
| `*.example.com` | `_wild_.example.com_cert-chain.pem` | `_wild_.example.com_key.pem` |

**Note:** Wildcards (`*`) are replaced with `_wild_` in filenames.

## Permissions

- **Certificate files**: `0644` (readable by all)
- **Private key files**: `0600` (owner only) ⚠️

## Troubleshooting

### No certificates exported

**Check if certificates exist:**
```bash
curl http://127.0.0.1:15872/api/certs | jq '.[] | {domain: .common_name, status: .status, managed: .managed}'
```

**Expected output:**
```json
{
  "domain": "example.com",
  "status": "issued",
  "managed": true
}
```

### Permission denied

```bash
# Ensure output directory is writable
sudo mkdir -p /etc/ssl/acme-certs
sudo chown $USER:$USER /etc/ssl/acme-certs
```

### Vault connection failed

```bash
# Test Vault connectivity
curl http://127.0.0.1:8200/v1/sys/health

# Check config
cat config.yml | grep -A5 vault
```

## Complete Documentation

See [FETCHER.md](FETCHER.md) for:
- Detailed usage examples
- Integration guides (nginx, Apache, HAProxy)
- Automation examples (Ansible, systemd)
- Security considerations
- Advanced configuration

## Workflow

```
┌─────────────────┐
│  acme-store     │  ← Obtains & renews certificates
│  (daemon)       │  ← Stores in Vault
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ Vault  │  ← Central certificate storage
    └────┬───┘
         │
         ▼
┌─────────────────┐
│ acme-store-     │  ← Exports certificates to disk
│ fetcher (CLI)   │  ← Runs periodically (cron/timer)
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ Files  │  ← example.com_cert-chain.pem
    │ on     │  ← example.com_key.pem
    │ Disk   │
    └────┬───┘
         │
         ▼
┌─────────────────┐
│ Web Server      │  ← nginx, Apache, HAProxy, etc.
│ (nginx, etc.)   │  ← Uses the PEM files
└─────────────────┘
```

## Next Steps

1. ✅ Build and test the fetcher
2. ✅ Verify certificates are exported correctly
3. ✅ Configure your web server to use the certificates
4. ✅ Set up automation (cron or systemd timer)
5. ✅ Test certificate renewal workflow

## Support

- Full documentation: [FETCHER.md](FETCHER.md)
- Main README: [README.md](README.md)
- API documentation: [API.md](API.md)

