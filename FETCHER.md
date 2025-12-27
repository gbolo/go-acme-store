# ACME Store Fetcher

## Overview

`acme-store-fetcher` is a CLI tool that fetches certificates from the ACME store API and saves them to disk as PEM files.

## Purpose

This tool is useful for:
- Exporting certificates to use with other applications (nginx, Apache, HAProxy, etc.)
- Creating backups of certificates
- Automating certificate deployment to servers
- Integration with configuration management tools (Ansible, Puppet, etc.)

## Features

- ✅ Fetches all managed domains from acme-store API
- ✅ Saves full certificate chain (leaf + intermediates) to disk
- ✅ Saves private keys with secure permissions (0600)
- ✅ Skips invalid/pending/failed certificates automatically
- ✅ Handles wildcard domains (replaces `*` with `_wild_`)
- ✅ Configurable output directory
- ✅ No direct keystore access required (uses API)
- ✅ **Daemon mode** - continuously monitor and update certificates
- ✅ Graceful shutdown on SIGTERM/SIGINT
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
# Fetch all certificates once (one-shot mode)
./acme-store-fetcher

# Run as daemon, continuously checking for updates
./acme-store-fetcher -daemon

# Specify config file
./acme-store-fetcher -config /etc/acme-store/config.yml

# Specify output directory (via flag)
./acme-store-fetcher -output-dir /etc/tls/acme-certs

# Run as daemon with custom check interval (via config)
# Add to config.yml:
#   fetcher.daemon: true
#   fetcher.check_interval: 10m
./acme-store-fetcher -config /etc/acme-store/config.yml

# Generate Traefik config (via flag)
./acme-store-fetcher \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml

# Run as daemon with Traefik config generation
./acme-store-fetcher \
  -daemon \
  -output-dir /etc/traefik/certs \
  -traefik-config /etc/traefik/dynamic/acme-certs.yml
```

### Command-Line Flags

| Flag | Default | Description | Config File Key |
|------|---------|-------------|-----------------|
| `-config` | `./config.yml` | Path to configuration file | `CONFIG_FILE` env var |
| `-output-dir` | `./certs` | Directory to save certificate files | `fetcher.output_dir` |
| `-traefik-config` | _(none)_ | Path to write Traefik configuration file | `fetcher.traefik_config` |
| `-daemon` | `false` | Run as daemon, continuously checking for updates | `fetcher.daemon` |

**Precedence**: Command-line flag > Config file > Default value

**Note**: When running in daemon mode, the check interval is controlled by `fetcher.check_interval` in the config file (default: 5 minutes).

### Configuration File

You can set fetcher options in `config.yml`:

```yaml
fetcher:
  # URL of the acme-store API (required)
  api_url: "http://127.0.0.1:15872/api"
  # Directory to save certificate files
  output_dir: "/etc/tls/acme-certs"
  # Path to write Traefik configuration file (optional)
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
  # Run as daemon (default: false)
  daemon: true
  # Check interval when running in daemon mode (default: 5m)
  check_interval: 10m
```

**Note**: Command-line flags override config file values.

### Environment Variables

- `CONFIG_FILE` - Alternative way to specify config file path
- `ACMESTORE_FETCHER_API_URL` - Alternative way to specify API URL
- `ACMESTORE_FETCHER_DAEMON` - Alternative way to enable daemon mode
- `ACMESTORE_FETCHER_CHECK_INTERVAL` - Alternative way to set check interval

## Output Files

For each managed domain with a valid certificate, two files are created:

### Certificate Chain File
- **Filename**: `{domain}_cert-chain.pem`
- **Contents**: Full certificate chain (leaf certificate + intermediate CA certificates)
- **Permissions**: `0644` (readable by all)
- **Use with**: nginx `ssl_certificate`, Apache `TLSCertificateFile`

### Private Key File
- **Filename**: `{domain}_key.pem`
- **Contents**: Private key for the certificate
- **Permissions**: `0600` (readable only by owner)
- **Use with**: nginx `ssl_certificate_key`, Apache `TLSCertificateKeyFile`

### Wildcard Domain Handling

Wildcard domains have `*` replaced with `_wild_` in filenames:

| Domain | Cert File | Key File |
|--------|-----------|----------|
| `example.com` | `example.com_cert-chain.pem` | `example.com_key.pem` |
| `*.example.com` | `_wild_.example.com_cert-chain.pem` | `_wild_.example.com_key.pem` |
| `*.lab.linuxctl.com` | `_wild_.lab.linuxctl.com_cert-chain.pem` | `_wild_.lab.linuxctl.com_key.pem` |

## Examples

### Example 1: Basic Fetch (One-Shot)

```bash
$ ./acme-store-fetcher
2025-12-27T11:02:39.743-0500 INFO config/viper.go:54 using config file: config.yml
2025-12-27T11:02:39.743-0500 INFO config/viper.go:64 initializing app: acme-store-fetcher v:devel(ref-unknown), platform: go1.25.5 [linux/amd64]
2025-12-27T11:02:39.744-0500 INFO acme-store-fetcher/main.go:87 connected to acme-store API at http://127.0.0.1:15872/api
2025-12-27T11:02:39.744-0500 INFO acme-store-fetcher/main.go:94 output directory: ./certs
2025-12-27T11:02:39.745-0500 INFO acme-store-fetcher/main.go:153 found 3 managed domain(s)
2025-12-27T11:02:39.745-0500 INFO acme-store-fetcher/main.go:166 processing domain: example.com
2025-12-27T11:02:39.747-0500 INFO acme-store-fetcher/main.go:233 saved certificate for domain=example.com cert=certs/example.com_cert-chain.pem key=certs/example.com_key.pem
2025-12-27T11:02:39.747-0500 INFO acme-store-fetcher/main.go:166 processing domain: *.example.com
2025-12-27T11:02:39.748-0500 INFO acme-store-fetcher/main.go:233 saved certificate for domain=*.example.com cert=certs/_wild_.example.com_cert-chain.pem key=certs/_wild_.example.com_key.pem
2025-12-27T11:02:39.749-0500 INFO acme-store-fetcher/main.go:253 fetch complete: total=3 saved=2 skipped=1 errors=0
```

### Example 2: Custom Output Directory

```bash
$ ./acme-store-fetcher -output-dir /etc/tls/acme-certs
2025-12-12T14:30:00.000-0500 INFO acme-store-fetcher/main.go:50 output directory: /etc/tls/acme-certs
...
```

### Example 3: Daemon Mode

```bash
$ ./acme-store-fetcher -daemon
2025-12-27T11:03:10.459-0500 INFO config/viper.go:54 using config file: config.yml
2025-12-27T11:03:10.459-0500 INFO config/viper.go:64 initializing app: acme-store-fetcher v:devel(ref-unknown), platform: go1.25.5 [linux/amd64]
2025-12-27T11:03:10.460-0500 INFO acme-store-fetcher/main.go:87 connected to acme-store API at http://127.0.0.1:15872/api
2025-12-27T11:03:10.460-0500 INFO acme-store-fetcher/main.go:94 output directory: ./certs
2025-12-27T11:03:10.460-0500 INFO acme-store-fetcher/main.go:98 running in daemon mode, checking every 5m0s
2025-12-27T11:03:10.460-0500 INFO acme-store-fetcher/main.go:119 performing initial certificate fetch
2025-12-27T11:03:10.461-0500 INFO acme-store-fetcher/main.go:153 found 1 managed domain(s)
2025-12-27T11:03:10.461-0500 INFO acme-store-fetcher/main.go:166 processing domain: test.linuxctl.com
2025-12-27T11:03:10.464-0500 INFO acme-store-fetcher/main.go:233 saved certificate for domain=test.linuxctl.com cert=certs/test.linuxctl.com_cert-chain.pem key=certs/test.linuxctl.com_key.pem
2025-12-27T11:03:10.465-0500 INFO acme-store-fetcher/main.go:253 fetch complete: total=1 saved=1 skipped=0 errors=0
2025-12-27T11:03:10.465-0500 INFO acme-store-fetcher/main.go:126 daemon started, will check for updates every 5m0s
2025-12-27T11:03:10.465-0500 INFO acme-store-fetcher/main.go:127 press Ctrl+C to stop gracefully
# ... waits 5 minutes ...
2025-12-27T11:08:10.465-0500 INFO acme-store-fetcher/main.go:138 checking for certificate updates
2025-12-27T11:08:10.466-0500 INFO acme-store-fetcher/main.go:153 found 1 managed domain(s)
2025-12-27T11:08:10.467-0500 INFO acme-store-fetcher/main.go:253 fetch complete: total=1 saved=1 skipped=0 errors=0
```

### Example 4: Graceful Shutdown

```bash
$ ./acme-store-fetcher -daemon
# ... running ...
^C
2025-12-27T10:58:54.196-0500 INFO acme-store-fetcher/main.go:135 received signal terminated, shutting down gracefully
```

### Example 5: Directory Structure

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
    
    TLSEngine on
    TLSCertificateFile /etc/tls/acme-certs/example.com_cert-chain.pem
    TLSCertificateKeyFile /etc/tls/acme-certs/example.com_key.pem
    
    # ... rest of config
</VirtualHost>
```

### HAProxy Configuration

```
frontend https_frontend
    bind *:443 ssl crt /etc/tls/acme-certs/example.com.pem
    # Note: HAProxy needs cert+key in single file, see automation example below
```

## Daemon Mode vs One-Shot Mode

The fetcher can run in two modes:

### One-Shot Mode (Default)
- Fetches certificates once and exits
- Suitable for cron jobs or manual runs
- Exit code indicates success/failure

### Daemon Mode (New!)
- Runs continuously in the background
- Periodically checks for certificate updates
- Automatically writes updated certificates to disk
- Graceful shutdown on SIGTERM/SIGINT
- Suitable for systemd services or Docker containers

**When to use daemon mode:**
- Production deployments where certificates update frequently
- Kubernetes/Docker environments
- Systems where you want real-time certificate updates
- Environments where you want to minimize external scheduling

**When to use one-shot mode:**
- Simple cron-based automation
- Environments with existing job schedulers
- When you want explicit control over execution timing

## Automation

### Option 1: Daemon Mode with Systemd (Recommended)

**Service file** (`/etc/systemd/system/acme-store-fetcher.service`):
```ini
[Unit]
Description=ACME Store Certificate Fetcher Daemon
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/acme-store-fetcher -config /etc/acme-store/config.yml -daemon
Restart=always
RestartSec=10
User=root

# Reload nginx when certificates are updated
# Note: The daemon continuously updates certs, nginx needs periodic reloads
ExecStartPost=/bin/sh -c 'while true; do sleep 3600; systemctl reload nginx; done' &

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable acme-store-fetcher.service
sudo systemctl start acme-store-fetcher.service
sudo systemctl status acme-store-fetcher.service
```

View logs:
```bash
sudo journalctl -u acme-store-fetcher -f
```

### Option 2: Cron Job (One-Shot Mode)

```bash
# /etc/cron.d/acme-store-fetcher
0 */6 * * * root /usr/local/bin/acme-store-fetcher -config /etc/acme-store/config.yml -output-dir /etc/ssl/acme-certs && systemctl reload nginx
```

### Option 3: Systemd Timer (One-Shot Mode)

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
sudo systemctl status acme-store-fetcher.timer
```

### Docker/Kubernetes Deployment (Daemon Mode)

**Dockerfile:**
```dockerfile
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary
COPY acme-store-fetcher /usr/local/bin/

# Copy config
COPY config.yml /etc/acme-store/config.yml

# Run as daemon
CMD ["/usr/local/bin/acme-store-fetcher", "-config", "/etc/acme-store/config.yml", "-daemon"]
```

**Kubernetes Deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: acme-store-fetcher
spec:
  replicas: 1
  selector:
    matchLabels:
      app: acme-store-fetcher
  template:
    metadata:
      labels:
        app: acme-store-fetcher
    spec:
      containers:
      - name: fetcher
        image: acme-store-fetcher:latest
        args:
        - "-daemon"
        - "-config"
        - "/etc/acme-store/config.yml"
        volumeMounts:
        - name: certs
          mountPath: /etc/tls/acme-certs
        - name: config
          mountPath: /etc/acme-store
      volumes:
      - name: certs
        emptyDir: {}
      - name: config
        configMap:
          name: acme-store-fetcher-config
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

CERT_DIR="/etc/tls/acme-certs"
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
1. Is the acme-store API accessible? `curl http://127.0.0.1:15872/api/healthz`
2. Are there managed domains? `curl http://127.0.0.1:15872/api/domains`
3. Do they have valid certificates? `curl http://127.0.0.1:15872/api/certs`
4. Is the API URL correct in config? `fetcher.api_url`

### Permission denied writing files

**Solution:**
```bash
# Ensure output directory is writable
sudo mkdir -p /etc/tls/acme-certs
sudo chown $USER:$USER /etc/tls/acme-certs
```

### API connection failed

**Check:**
1. API URL in config: `fetcher.api_url`
2. Is acme-store daemon running?
3. Network connectivity: `curl http://127.0.0.1:15872/api/healthz`
4. Firewall rules allowing access to API port

### Certificate data empty

**Possible causes:**
- Certificate still pending issuance
- Certificate issuance failed
- Domain was just added (wait a few minutes)

**Check status:**
```bash
curl http://127.0.0.1:15872/api/certs | jq '.[] | {domain: .common_name, status: .status}'
```

### Daemon mode not updating certificates

**Check:**
1. Is the daemon actually running? `ps aux | grep acme-store-fetcher`
2. Check the logs for errors
3. Verify check interval: look for "checking for certificate updates" in logs
4. Ensure API is accessible: `curl http://127.0.0.1:15872/api/healthz`

**Systemd service:**
```bash
sudo systemctl status acme-store-fetcher
sudo journalctl -u acme-store-fetcher -f
```

### Daemon exits unexpectedly

**Check:**
1. Review logs for error messages
2. Ensure API URL is correct and accessible
3. Check file permissions on output directory
4. Verify systemd service configuration (Restart=always)

**Debug mode:**
```bash
# Run in foreground to see all output
./acme-store-fetcher -daemon -config /etc/acme-store/config.yml
```

## Security Considerations

### File Permissions

- Certificate files: `0644` (world-readable, needed for web servers)
- Private key files: `0600` (owner-only, **critical for security**)

### Output Directory

Ensure the output directory has appropriate permissions:

```bash
# For system-wide certificates
sudo mkdir -p /etc/tls/acme-certs
sudo chown root:root /etc/tls/acme-certs
sudo chmod 755 /etc/tls/acme-certs

# For user-specific
mkdir -p ~/certs
chmod 700 ~/certs
```

### API Access

The fetcher connects to the acme-store API without authentication. Ensure:
- The API is only accessible from trusted networks
- Use firewall rules to restrict API access
- Consider adding authentication if exposing the API externally

## Comparison with acme-store

| Feature | acme-store | acme-store-fetcher |
|---------|------------|-------------------|
| Purpose | Obtain & renew certificates | Export certificates to disk |
| Runs as | Daemon (24/7) | CLI tool (on-demand) |
| Certificate issuance | ✅ Yes | ❌ No (read-only) |
| Certificate renewal | ✅ Yes | ❌ No (read-only) |
| Export to files | ❌ No | ✅ Yes |
| Web UI | ✅ Yes | ❌ No |
| API | ✅ Provides API | ✅ Consumes API |
| Keystore access | ✅ Direct | ❌ Via API only |

**Typical workflow:**
1. `acme-store` runs as daemon, obtains and renews certificates
2. `acme-store-fetcher` runs periodically to fetch certificates via API and export to disk
3. Web server (nginx, Apache, etc.) uses the exported files

## Advanced Usage

### Fetch Only Specific Domains

Currently not supported, but you can filter after fetch:

```bash
# Fetch all
./acme-store-fetcher -output-dir /tmp/all-certs

# Copy only specific domain
cp /tmp/all-certs/example.com_* /etc/tls/acme-certs/
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

