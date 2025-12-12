# ACME Store Fetcher

## Overview

`acme-store-fetcher` is a CLI tool that fetches certificates from the ACME store (Vault backend) and saves them to disk as PEM files.

## Purpose

This tool is useful for:
- Exporting certificates to use with other applications (nginx, Apache, HAProxy, etc.)
- Creating backups of certificates
- Automating certificate deployment to servers
- Integration with configuration management tools (Ansible, Puppet, etc.)

## Features

- ✅ Fetches all managed domains from Vault
- ✅ Saves full certificate chain (leaf + intermediates) to disk
- ✅ Saves private keys with secure permissions (0600)
- ✅ Skips invalid/pending/failed certificates automatically
- ✅ Handles wildcard domains (replaces `*` with `_wild_`)
- ✅ Configurable output directory
- ✅ Detailed logging and summary report
- ✅ Exit code indicates success/failure

## Installation

### Build from Source

```bash
cd /path/to/go-acme-store
go build -o acme-store-fetcher ./cmd/acme-store-fetcher
```

### Install to System

```bash
sudo cp acme-store-fetcher /usr/local/bin/
sudo chmod +x /usr/local/bin/acme-store-fetcher
```

## Usage

### Basic Usage

```bash
# Fetch all certificates to ./certs directory
./acme-store-fetcher

# Specify config file
./acme-store-fetcher -config /etc/acme-store/config.yml

# Specify output directory (via flag)
./acme-store-fetcher -output-dir /etc/ssl/acme-certs

# Specify output directory (via config file)
# Add to config.yml: fetcher.output_dir: "/etc/ssl/acme-certs"
./acme-store-fetcher -config /etc/acme-store/config.yml

# Generate Traefik config (via flag)
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml

# Generate Traefik config (via config file)
# Add to config.yml:
#   fetcher.output_dir: "/etc/traefik/certs"
#   fetcher.traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
./acme-store-fetcher -config /etc/acme-store/config.yml
```

### Command-Line Flags

| Flag | Default | Description | Config File Key |
|------|---------|-------------|-----------------|
| `-config` | `./config.yml` | Path to configuration file | `CONFIG_FILE` env var |
| `-output-dir` | `./certs` | Directory to save certificate files | `fetcher.output_dir` |
| `-traefik-config` | _(none)_ | Path to write Traefik configuration file | `fetcher.traefik_config` |

**Precedence**: Command-line flag > Config file > Default value

### Configuration File

You can set fetcher options in `config.yml`:

```yaml
fetcher:
  # Directory to save certificate files
  output_dir: "/etc/ssl/acme-certs"
  # Path to write Traefik configuration file (optional)
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Note**: Command-line flags override config file values.

### Environment Variables

- `CONFIG_FILE` - Alternative way to specify config file path
- `VAULT_TOKEN` - Vault authentication token (recommended instead of config file)
- `DIGITALOCEAN_TOKEN` - DigitalOcean API token (not needed by fetcher)

## Output Files

For each managed domain with a valid certificate, two files are created:

### Certificate Chain File
- **Filename**: `{domain}_cert-chain.pem`
- **Contents**: Full certificate chain (leaf certificate + intermediate CA certificates)
- **Permissions**: `0644` (readable by all)
- **Use with**: nginx `ssl_certificate`, Apache `SSLCertificateFile`

### Private Key File
- **Filename**: `{domain}_key.pem`
- **Contents**: Private key for the certificate
- **Permissions**: `0600` (readable only by owner)
- **Use with**: nginx `ssl_certificate_key`, Apache `SSLCertificateKeyFile`

### Wildcard Domain Handling

Wildcard domains have `*` replaced with `_wild_` in filenames:

| Domain | Cert File | Key File |
|--------|-----------|----------|
| `example.com` | `example.com_cert-chain.pem` | `example.com_key.pem` |
| `*.example.com` | `_wild_.example.com_cert-chain.pem` | `_wild_.example.com_key.pem` |
| `*.lab.linuxctl.com` | `_wild_.lab.linuxctl.com_cert-chain.pem` | `_wild_.lab.linuxctl.com_key.pem` |

## Examples

### Example 1: Basic Fetch

```bash
$ ./acme-store-fetcher
2025-12-12T14:30:00.000-0500 INFO acme-store-fetcher/main.go:42 initialized vault keystore at http://127.0.0.1:8200
2025-12-12T14:30:00.100-0500 INFO acme-store-fetcher/main.go:50 output directory: ./certs
2025-12-12T14:30:00.200-0500 INFO acme-store-fetcher/main.go:58 found 3 managed domain(s)
2025-12-12T14:30:00.300-0500 INFO acme-store-fetcher/main.go:67 processing domain: example.com
2025-12-12T14:30:00.400-0500 INFO acme-store-fetcher/main.go:115 saved certificate for example.com:
2025-12-12T14:30:00.400-0500 INFO acme-store-fetcher/main.go:116   - cert chain: ./certs/example.com_cert-chain.pem
2025-12-12T14:30:00.400-0500 INFO acme-store-fetcher/main.go:117   - private key: ./certs/example.com_key.pem
2025-12-12T14:30:00.500-0500 INFO acme-store-fetcher/main.go:67 processing domain: *.example.com
2025-12-12T14:30:00.600-0500 INFO acme-store-fetcher/main.go:115 saved certificate for *.example.com:
2025-12-12T14:30:00.600-0500 INFO acme-store-fetcher/main.go:116   - cert chain: ./certs/_wild_.example.com_cert-chain.pem
2025-12-12T14:30:00.600-0500 INFO acme-store-fetcher/main.go:117   - private key: ./certs/_wild_.example.com_key.pem
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:122 ========================================
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:123 Summary:
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:124   Total domains: 3
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:125   Successfully saved: 2
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:126   Skipped: 1
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:127   Errors: 0
2025-12-12T14:30:00.700-0500 INFO acme-store-fetcher/main.go:128 ========================================
```

### Example 2: Custom Output Directory

```bash
$ ./acme-store-fetcher -output-dir /etc/ssl/acme-certs
2025-12-12T14:30:00.000-0500 INFO acme-store-fetcher/main.go:50 output directory: /etc/ssl/acme-certs
...
```

### Example 3: Directory Structure

```bash
$ tree ./certs
./certs
├── example.com_cert-chain.pem
├── example.com_key.pem
├── _wild_.example.com_cert-chain.pem
└── _wild_.example.com_key.pem

$ ls -lh ./certs
-rw-r--r-- 1 user user 3.2K Dec 12 14:30 example.com_cert-chain.pem
-rw------- 1 user user 1.7K Dec 12 14:30 example.com_key.pem
-rw-r--r-- 1 user user 3.2K Dec 12 14:30 _wild_.example.com_cert-chain.pem
-rw------- 1 user user 1.7K Dec 12 14:30 _wild_.example.com_key.pem
```

## Certificate States

The fetcher handles different certificate states:

| State | Action | Reason |
|-------|--------|--------|
| `issued` | ✅ Fetch | Certificate is valid and ready |
| `pending` | ⏭️ Skip | Certificate not yet issued |
| `failed` | ⏭️ Skip | Certificate issuance failed |
| Unmanaged | ⏭️ Skip | Domain no longer managed |
| Empty data | ⏭️ Skip | No certificate or key data |

## Integration Examples

### Nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name example.com;
    
    ssl_certificate /etc/ssl/acme-certs/example.com_cert-chain.pem;
    ssl_certificate_key /etc/ssl/acme-certs/example.com_key.pem;
    
    # ... rest of config
}
```

### Apache Configuration

```apache
<VirtualHost *:443>
    ServerName example.com
    
    SSLEngine on
    SSLCertificateFile /etc/ssl/acme-certs/example.com_cert-chain.pem
    SSLCertificateKeyFile /etc/ssl/acme-certs/example.com_key.pem
    
    # ... rest of config
</VirtualHost>
```

### HAProxy Configuration

```
frontend https_frontend
    bind *:443 ssl crt /etc/ssl/acme-certs/example.com.pem
    # Note: HAProxy needs cert+key in single file, see automation example below
```

## Automation

### Cron Job (Every 6 Hours)

```bash
# /etc/cron.d/acme-store-fetcher
0 */6 * * * root /usr/local/bin/acme-store-fetcher -config /etc/acme-store/config.yml -output-dir /etc/ssl/acme-certs && systemctl reload nginx
```

### Systemd Timer

**Service file** (`/etc/systemd/system/acme-store-fetcher.service`):
```ini
[Unit]
Description=Fetch ACME certificates from Vault
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/acme-store-fetcher -config /etc/acme-store/config.yml -output-dir /etc/ssl/acme-certs
ExecStartPost=/bin/systemctl reload nginx
User=root
```

**Timer file** (`/etc/systemd/system/acme-store-fetcher.timer`):
```ini
[Unit]
Description=Fetch ACME certificates every 6 hours

[Timer]
OnBootSec=5min
OnUnitActiveSec=6h

[Install]
WantedBy=timers.target
```

Enable and start:
```bash
sudo systemctl enable acme-store-fetcher.timer
sudo systemctl start acme-store-fetcher.timer
```

### Ansible Playbook

```yaml
---
- name: Fetch ACME certificates
  hosts: webservers
  tasks:
    - name: Run acme-store-fetcher
      command: >
        /usr/local/bin/acme-store-fetcher
        -config /etc/acme-store/config.yml
        -output-dir /etc/ssl/acme-certs
      register: fetcher_result
      changed_when: fetcher_result.rc == 0
      
    - name: Reload nginx
      service:
        name: nginx
        state: reloaded
      when: fetcher_result.changed
```

### Traefik Configuration

The fetcher can automatically generate a Traefik-compatible configuration file:

```bash
# Generate Traefik config
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml
```

**Generated file** (`/etc/traefik/dynamic/acme-certs.yml`):
```yaml
tls:
  options:
    default:
      minVersion: VersionTLS12
    mintls13:
      minVersion: VersionTLS13
  certificates:
    - certFile: /etc/traefik/certs/example.com_cert-chain.pem
      keyFile: /etc/traefik/certs/example.com_key.pem
    - certFile: /etc/traefik/certs/_wild_.example.com_cert-chain.pem
      keyFile: /etc/traefik/certs/_wild_.example.com_key.pem
```

**Traefik main config** (`traefik.yml`):
```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

Traefik will automatically reload when the config file changes!

### HAProxy Certificate Combining

HAProxy requires cert and key in a single file:

```bash
#!/bin/bash
# combine-for-haproxy.sh

CERT_DIR="/etc/ssl/acme-certs"
HAPROXY_DIR="/etc/haproxy/certs"

# Fetch certificates
/usr/local/bin/acme-store-fetcher -output-dir "$CERT_DIR"

# Combine cert and key for each domain
for cert in "$CERT_DIR"/*_cert-chain.pem; do
    domain=$(basename "$cert" _cert-chain.pem)
    key="$CERT_DIR/${domain}_key.pem"
    
    if [ -f "$key" ]; then
        cat "$cert" "$key" > "$HAPROXY_DIR/${domain}.pem"
        chmod 600 "$HAPROXY_DIR/${domain}.pem"
        echo "Combined $domain"
    fi
done

# Reload HAProxy
systemctl reload haproxy
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success - all certificates fetched |
| `1` | Errors occurred during fetch |

## Troubleshooting

### No certificates fetched

**Check:**
1. Are there managed domains? `curl http://127.0.0.1:15872/api/domains`
2. Do they have valid certificates? `curl http://127.0.0.1:15872/api/certs`
3. Is Vault accessible from this host?
4. Are Vault credentials correct in config?

### Permission denied writing files

**Solution:**
```bash
# Ensure output directory is writable
sudo mkdir -p /etc/ssl/acme-certs
sudo chown $USER:$USER /etc/ssl/acme-certs
```

### Vault connection failed

**Check:**
1. Vault address in config: `vault.address`
2. Vault token: `vault.token` or `VAULT_TOKEN` env var
3. Network connectivity: `curl http://127.0.0.1:8200/v1/sys/health`

### Certificate data empty

**Possible causes:**
- Certificate still pending issuance
- Certificate issuance failed
- Domain was just added (wait a few minutes)

**Check status:**
```bash
curl http://127.0.0.1:15872/api/certs | jq '.[] | {domain: .common_name, status: .status}'
```

## Security Considerations

### File Permissions

- Certificate files: `0644` (world-readable, needed for web servers)
- Private key files: `0600` (owner-only, **critical for security**)

### Output Directory

Ensure the output directory has appropriate permissions:

```bash
# For system-wide certificates
sudo mkdir -p /etc/ssl/acme-certs
sudo chown root:root /etc/ssl/acme-certs
sudo chmod 755 /etc/ssl/acme-certs

# For user-specific
mkdir -p ~/certs
chmod 700 ~/certs
```

### Vault Token

Never commit Vault tokens to version control. Use:
- Environment variable: `VAULT_TOKEN`
- Config file with restricted permissions: `chmod 600 config.yml`
- Vault agent for automatic token management

## Comparison with acme-store

| Feature | acme-store | acme-store-fetcher |
|---------|------------|-------------------|
| Purpose | Obtain & renew certificates | Export certificates to disk |
| Runs as | Daemon (24/7) | CLI tool (on-demand) |
| Certificate issuance | ✅ Yes | ❌ No (read-only) |
| Certificate renewal | ✅ Yes | ❌ No (read-only) |
| Export to files | ❌ No | ✅ Yes |
| Web UI | ✅ Yes | ❌ No |
| API | ✅ Yes | ❌ No |

**Typical workflow:**
1. `acme-store` runs as daemon, obtains and renews certificates
2. `acme-store-fetcher` runs periodically to export certificates to disk
3. Web server (nginx, Apache, etc.) uses the exported files

## Advanced Usage

### Fetch Only Specific Domains

Currently not supported, but you can filter after fetch:

```bash
# Fetch all
./acme-store-fetcher -output-dir /tmp/all-certs

# Copy only specific domain
cp /tmp/all-certs/example.com_* /etc/ssl/acme-certs/
```

### Dry Run Mode

Not currently supported. Feature request welcome!

### Custom Filename Format

Not currently supported. Modify source code if needed.

## Traefik Integration (NEW!)

The fetcher can generate Traefik-compatible configuration automatically.

### Setup

**1. Configure Traefik to watch dynamic configs:**

`/etc/traefik/traefik.yml`:
```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

**2. Run fetcher with Traefik flag:**

```bash
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml
```

**3. Traefik auto-reloads** when the config file changes - no restart needed!

### Generated Configuration

```yaml
tls:
  options:
    default:
      minVersion: VersionTLS12
    mintls13:
      minVersion: VersionTLS13
  certificates:
    - certFile: /etc/traefik/certs/example.com_cert-chain.pem
      keyFile: /etc/traefik/certs/example.com_key.pem
    - certFile: /etc/traefik/certs/_wild_.example.com_cert-chain.pem
      keyFile: /etc/traefik/certs/_wild_.example.com_key.pem
```

### Cron Automation

```bash
# /etc/cron.d/acme-store-fetcher-traefik
0 */6 * * * root /usr/local/bin/acme-store-fetcher \
  -config /etc/acme-store/config.yml \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml
```

### TLS Options Usage

Apply the TLS options in your Traefik routes:

```yaml
# /etc/traefik/dynamic/routes.yml
http:
  routers:
    my-router:
      rule: "Host(`example.com`)"
      tls:
        options: mintls13  # Use TLS 1.3 minimum
      service: my-service
```

## Future Enhancements

Potential features:
- [ ] Dry run mode
- [ ] Filter by domain pattern
- [ ] Custom filename templates
- [ ] Combine cert+key for HAProxy
- [ ] Checksum/hash output
- [ ] Only update if changed
- [ ] Webhook notifications
- [ ] Metrics export
- [x] Traefik configuration generation ✅

## Support

For issues, questions, or feature requests, please check:
- Main README: `README.md`
- API documentation: `API.md`
- Configuration guide: `QUICKSTART.md`

