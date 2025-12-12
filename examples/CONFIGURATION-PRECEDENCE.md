# Configuration Precedence Examples

This document shows how `acme-store-fetcher` resolves configuration from multiple sources.

## Precedence Order

**Highest to Lowest:**
1. Command-line flags
2. Configuration file (`config.yml`)
3. Default values

## Example Scenarios

### Scenario 1: Using Defaults

**Command:**
```bash
./acme-store-fetcher
```

**Result:**
- `output_dir`: `./certs` (default)
- `traefik_config`: _(none)_ (default)

---

### Scenario 2: Config File Only

**Config file** (`config.yml`):
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Command:**
```bash
./acme-store-fetcher -config config.yml
```

**Result:**
- `output_dir`: `/etc/ssl/acme-certs` (from config)
- `traefik_config`: `/etc/traefik/dynamic/acme-certs.yml` (from config)

---

### Scenario 3: Override with Flags

**Config file** (`config.yml`):
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Command:**
```bash
./acme-store-fetcher -config config.yml -output-dir /custom/path
```

**Result:**
- `output_dir`: `/custom/path` ⬅️ **FLAG WINS**
- `traefik_config`: `/etc/traefik/dynamic/acme-certs.yml` (from config)

---

### Scenario 4: Partial Override

**Config file** (`config.yml`):
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  # traefik_config not set
```

**Command:**
```bash
./acme-store-fetcher -config config.yml -traefik-config /tmp/traefik.yml
```

**Result:**
- `output_dir`: `/etc/ssl/acme-certs` (from config)
- `traefik_config`: `/tmp/traefik.yml` (from flag)

---

### Scenario 5: Disable Traefik Config

**Config file** (`config.yml`):
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Command (to disable Traefik config generation):**
```bash
# Comment out or remove traefik_config from config file
# OR just don't use that config file
./acme-store-fetcher -output-dir /etc/ssl/acme-certs
```

**Result:**
- `output_dir`: `/etc/ssl/acme-certs` (from flag)
- `traefik_config`: _(none)_ - Traefik config not generated

---

### Scenario 6: Environment Variable + Config

**Environment:**
```bash
export CONFIG_FILE="/etc/acme-store/config.yml"
```

**Config file** (`/etc/acme-store/config.yml`):
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Command:**
```bash
./acme-store-fetcher
# Uses CONFIG_FILE env var to find config
```

**Result:**
- Config file: `/etc/acme-store/config.yml` (from env var)
- `output_dir`: `/etc/ssl/acme-certs` (from config)
- `traefik_config`: `/etc/traefik/dynamic/acme-certs.yml` (from config)

---

## Decision Tree

```
┌─────────────────────────────────────┐
│ Is flag provided on command line?  │
└───────────┬─────────────────────────┘
            │
            ├─ Yes ──────────────────┐
            │                        ▼
            │                  [Use flag value]
            │
            └─ No ───────────────────┐
                                     ▼
                ┌──────────────────────────────────┐
                │ Is value set in config file?    │
                └───────────┬──────────────────────┘
                            │
                            ├─ Yes ───────┐
                            │              ▼
                            │        [Use config value]
                            │
                            └─ No ────────┐
                                          ▼
                                    [Use default value]
```

## Practical Examples

### Development Setup

**Config file** (`dev-config.yml`):
```yaml
fetcher:
  output_dir: "./dev-certs"
  # No Traefik config for dev
```

**Usage:**
```bash
./acme-store-fetcher -config dev-config.yml
```

---

### Production Setup

**Config file** (`prod-config.yml`):
```yaml
fetcher:
  output_dir: "/etc/traefik/certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"

vault:
  address: "https://vault.prod.example.com:8200"
  kv2_mount: kv
  kv2_secret_path: prod/acme
```

**Cron job:**
```bash
0 */6 * * * /usr/local/bin/acme-store-fetcher -config /etc/acme-store/prod-config.yml
```

**Result:**
- Uses all settings from prod config
- Automatically generates Traefik config
- No need for command-line flags

---

### Testing/Override Setup

**Config file** (`prod-config.yml`):
```yaml
fetcher:
  output_dir: "/etc/traefik/certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
```

**Test with different output:**
```bash
# Test fetch to temporary location
./acme-store-fetcher \
  -config /etc/acme-store/prod-config.yml \
  -output-dir /tmp/test-certs \
  -traefik-config /tmp/test-traefik.yml
```

**Result:**
- Uses prod config for Vault settings
- Overrides output locations for testing
- Doesn't affect production files

---

## Tips

### 1. **Use config file for stable settings**
```yaml
fetcher:
  output_dir: "/etc/ssl/acme-certs"
  traefik_config: "/etc/traefik/dynamic/acme-certs.yml"
vault:
  address: "https://vault.example.com"
```

Then just run:
```bash
./acme-store-fetcher
```

### 2. **Use flags for one-off changes**
```bash
# Temporarily use different output
./acme-store-fetcher -output-dir /tmp/backup
```

### 3. **Combine both for flexibility**
```yaml
# config.yml - stable settings
vault:
  address: "https://vault.example.com"
fetcher:
  output_dir: "/etc/ssl/acme-certs"
```

```bash
# Add Traefik generation only when needed
./acme-store-fetcher -traefik-config /etc/traefik/dynamic/acme-certs.yml
```

### 4. **Different configs for different environments**
```bash
# Development
./acme-store-fetcher -config dev-config.yml

# Staging  
./acme-store-fetcher -config staging-config.yml

# Production
./acme-store-fetcher -config prod-config.yml
```

## Summary

| Setting | Flag | Config Key | Default |
|---------|------|------------|---------|
| Config File | `-config` | `CONFIG_FILE` env | `./config.yml` |
| Output Directory | `-output-dir` | `fetcher.output_dir` | `./certs` |
| Traefik Config | `-traefik-config` | `fetcher.traefik_config` | _(none)_ |

**Remember:** Flags always win! 🏆

