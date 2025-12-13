# Testing Environment Setup

This document describes how to set up a complete testing environment for `acme-store`.

## Overview

The testing environment includes:
1. **HashiCorp Vault** (dev mode) - Certificate storage
2. **Pebble** - Let's Encrypt ACME test server  
3. **challtestsrv** - Mock DNS for Pebble validation
4. **acme-dns** - DNS-01 challenge delegation server

## Quick Start

### 1. Start Test Environment

```bash
docker-compose -f docker-compose.test.yml up -d
```

### 2. Verify Services

```bash
# Check Vault
curl http://localhost:8200/v1/sys/health

# Check Pebble ACME directory
curl -k https://localhost:14000/dir

# Check acme-dns
curl http://localhost:8053/health
```

### 3. Configure acme-store

Create `config.test.yml`:

```yaml
log:
  level: DEBUG

server:
  bind_address: 127.0.0.1
  bind_port: 15872

acme:
  # Use Pebble instead of Let's Encrypt
  directory: "https://localhost:14000/dir"
  account_email: test@example.com
  dns_server: "localhost:53"
  
  # Choose DNS provider
  dns_provider: "acmedns"  # or "digitalocean"
  
  # ACME-DNS configuration
  acmedns_server_url: "http://localhost:8053"
  acmedns_storage: "./acme-dns-accounts.json"

vault:
  address: "http://localhost:8200"
  token: "root"  # Dev mode token
  kv2_mount: kv
  kv2_secret_path: test/acme
  tls_skip_verify: false

fetcher:
  output_dir: "./test-certs"
```

### 4. Run acme-store

```bash
./acme-store -config config.test.yml
```

## Service Details

### Vault (Port 8200)

**Dev Mode** - INSECURE, for testing only!

- Root token: `root`
- KV v2 engine automatically enabled at `kv/`
- No persistence (data lost on restart)

**Access:**
```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
vault kv list kv/test/acme/domain/
```

### Pebble (Ports 14000, 15000)

Let's Encrypt-compatible ACME server for testing.

- ACME API: https://localhost:14000
- Management API: http://localhost:15000

**Features:**
- Fast certificate issuance (no rate limits)
- Uses challtestsrv for DNS validation
- Self-signed CA (not trusted by browsers)

**Get Pebble CA cert:**
```bash
curl -k https://localhost:15000/roots/0 > pebble-ca.crt
```

### challtestsrv (Port 8055)

Mock DNS server for Pebble validation.

- Management API: http://localhost:8055
- DNS: Port 53 (internal Docker network only, not exposed to host)

**Add DNS records:**
```bash
# Add A record
curl -X POST http://localhost:8055/add-a \
  -d '{"host": "example.com.", "addresses": ["172.20.0.10"]}'

# Add TXT record for DNS-01 challenge
curl -X POST http://localhost:8055/set-txt \
  -d '{"host": "_acme-challenge.example.com.", "value": "challenge-token"}'
```

### acme-dns (Port 8053)

DNS delegation server for DNS-01 challenges.

- HTTP API: http://localhost:8053
- DNS: Port 53533 (internal Docker network only, not exposed to host)

**Register account:**
```bash
curl -X POST http://localhost:8053/register
```

**Response:**
```json
{
  "username": "eabcdb41-d89f-4580-826f-3e62e9755ef2",
  "password": "pbkdf2-sha256$6400$...",
  "fulldomain": "d420c923-bbd7-4056-ab64-c3ca54c9b3cf.acme.test",
  "subdomain": "d420c923-bbd7-4056-ab64-c3ca54c9b3cf",
  "allowfrom": []
}
```

## Testing Workflows

### Test 1: Basic Certificate Issuance

```bash
# 1. Start environment
docker-compose -f docker-compose.test.yml up -d

# 2. Add domain to acme-store
curl -X POST http://localhost:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain": "test.example.com"}'

# 3. Check certificate status
curl http://localhost:15872/api/certs | jq

# 4. Fetch certificate files
./acme-store-fetcher -config config.test.yml -output-dir ./test-certs
```

### Test 2: Wildcard Certificate

```bash
# Add wildcard domain
curl -X POST http://localhost:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain": "*.example.com", "sans": ["example.com"]}'
```

### Test 3: Certificate Renewal

```bash
# Trigger manual renewal
curl -X POST http://localhost:15872/api/trigger-renewal

# Watch logs
docker-compose -f docker-compose.test.yml logs -f acme-store
```

## Troubleshooting

### Pebble can't validate challenges

**Problem:** DNS validation fails

**Solution:** Ensure challtestsrv is running and Pebble can reach it:
```bash
docker-compose -f docker-compose.test.yml logs challtestsrv
docker-compose -f docker-compose.test.yml logs pebble
```

### Vault connection refused

**Problem:** Can't connect to Vault

**Solution:** Check Vault is healthy:
```bash
docker-compose -f docker-compose.test.yml ps vault
curl http://localhost:8200/v1/sys/health
```

### acme-dns registration fails

**Problem:** Can't register with acme-dns

**Solution:** Check acme-dns logs:
```bash
docker-compose -f docker-compose.test.yml logs acme-dns
```

## Cleanup

```bash
# Stop all services
docker-compose -f docker-compose.test.yml down

# Remove volumes (deletes all data)
docker-compose -f docker-compose.test.yml down -v

# Clean test files
rm -rf test-certs/ acme-dns-accounts.json
```

## Network Architecture

```
┌─────────────────┐
│  acme-store     │
│  (host)         │
└────────┬────────┘
         │
         ├─────────────────┐
         │                 │
         ▼                 ▼
    ┌────────┐        ┌─────────┐
    │ Vault  │        │ Pebble  │
    │ :8200  │        │ :14000  │
    └────────┘        └────┬────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ challtestsrv │
                    │ :8055, :53   │
                    └──────────────┘

    ┌──────────┐
    │ acme-dns │
    │ :8053    │
    └──────────┘
```

## Next Steps

1. ✅ Start test environment
2. ✅ Verify all services are running
3. ✅ Configure acme-store for testing
4. ✅ Test certificate issuance
5. ✅ Test certificate fetching
6. ✅ Test Traefik integration

## Production Considerations

**DO NOT use this setup in production!**

- Vault dev mode has no persistence
- Pebble certificates are not trusted
- No authentication on any service
- All services use default/weak credentials

For production:
- Use production Vault with proper auth
- Use Let's Encrypt production servers
- Use real DNS provider
- Enable authentication and TLS everywhere

## References

- [Pebble](https://github.com/letsencrypt/pebble)
- [challtestsrv](https://github.com/letsencrypt/challtestsrv)
- [acme-dns](https://github.com/joohoi/acme-dns)
- [HashiCorp Vault](https://www.vaultproject.io/)

