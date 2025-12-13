# Integration Tests

This document describes the integration testing setup for `go-acme-store`.

## Overview

The integration tests verify the complete functionality of the ACME store API by:
- Running a real instance of the application
- Using HashiCorp Vault for storage
- Testing all API endpoints
- Verifying domain management and certificate operations

## Test Environment

The test environment uses Docker Compose to spin up the following services:

### Services

1. **Vault** (HashiCorp Vault)
   - Port: `8200`
   - Mode: Development (unsealed, in-memory)
   - Root token: `root`
   - Purpose: Certificate and domain storage

2. **Pebble** (ACME Test Server)
   - Port: `14000` (ACME API), `15000` (Management)
   - Purpose: Mock ACME server for testing certificate issuance
   - Note: Uses self-signed certificates

3. **challtestsrv** (Challenge Test Server)
   - Port: `8055` (HTTP API)
   - DNS: Port `53` (internal only)
   - Purpose: Mock DNS for ACME DNS-01 challenges

4. **acme-dns** (DNS-01 Delegation Server)
   - Port: `8053` (HTTP API)
   - DNS: Port `53533` (internal only)
   - Purpose: Alternative DNS-01 challenge provider

## Quick Start

### 1. Start Test Environment

```bash
make test-setup
```

This will:
- Pull required Docker images
- Start all services
- Wait for services to be ready
- Verify connectivity

### 2. Run Integration Tests

```bash
make test-integration
```

This will:
- Build the `acme-store` binary
- Clean previous test data from Vault
- Start the API server in the background
- Run all integration tests
- Stop the API server
- Display test results

### 3. Clean Up

```bash
make test-cleanup
```

This will:
- Stop all Docker containers
- Remove volumes
- Clean up test data

### 4. Run Full Test Suite

```bash
make test-all
```

This runs: `test-setup` → `test-integration` → `test-cleanup`

## Test Configuration

The test configuration is located at `testdata/config/test.yml`:

```yaml
log:
  level: DEBUG

server:
  bind_address: 127.0.0.1
  bind_port: 15872

acme:
  directory: "https://localhost:14000/dir"  # Pebble ACME server
  account_email: test@example.com
  dns_server: "challtestsrv:53"
  dns_provider: "digitalocean"

vault:
  address: "http://localhost:8200"
  token: "root"
  kv2_mount: kv
  kv2_secret_path: test/acme
  tls_skip_verify: false

fetcher:
  output_dir: "./testdata/certs"
```

## Test Coverage

The integration tests cover:

### API Endpoints
- ✅ `GET /api/healthz` - Health check
- ✅ `GET /api/version` - Version information
- ✅ `GET /api/config` - Configuration (non-sensitive)
- ✅ `GET /api/domains` - List managed domains
- ✅ `POST /api/domains` - Add domain
- ✅ `DELETE /api/domains/:domain` - Remove domain
- ✅ `GET /api/certs` - List certificates
- ✅ `POST /api/trigger-renewal` - Trigger certificate renewal

### Functionality
- ✅ Domain management (add, list, remove)
- ✅ Subject Alternative Names (SANs)
- ✅ Soft delete (unmanage) vs hard delete
- ✅ Certificate status tracking (pending, issued, failed)
- ✅ Error handling and validation
- ⏭️ Frontend routes (skipped)

## Test Results

All API tests pass successfully:

```
=== RUN   TestHealthCheck
--- PASS: TestHealthCheck (0.00s)
=== RUN   TestVersion
--- PASS: TestVersion (0.00s)
=== RUN   TestConfig
--- PASS: TestConfig (0.00s)
=== RUN   TestListDomainsInitiallyEmpty
--- PASS: TestListDomainsInitiallyEmpty (0.00s)
=== RUN   TestAddDomain
--- PASS: TestAddDomain (0.00s)
=== RUN   TestAddDomainWithSANs
--- PASS: TestAddDomainWithSANs (0.00s)
=== RUN   TestListDomainsAfterAdd
--- PASS: TestListDomainsAfterAdd (0.00s)
=== RUN   TestListCertificates
--- PASS: TestListCertificates (2.00s)
=== RUN   TestTriggerRenewal
--- PASS: TestTriggerRenewal (0.00s)
=== RUN   TestRemoveDomainSoft
--- PASS: TestRemoveDomainSoft (0.01s)
=== RUN   TestRemoveDomainHard
--- PASS: TestRemoveDomainHard (0.01s)
=== RUN   TestInvalidDomainRejection
--- PASS: TestInvalidDomainRejection (0.00s)

PASS
ok  	go-acme-store/tests/integration	2.039s
```

## Troubleshooting

### Services Not Starting

Check Docker logs:
```bash
docker logs acme-vault
docker logs acme-pebble
docker logs acme-challtestsrv
docker logs acme-dns
```

### Port Conflicts

If ports are already in use, you can modify `docker-compose.test.yml` to use different host ports.

### Test Failures

1. Ensure all services are running:
   ```bash
   docker compose -f docker-compose.test.yml ps
   ```

2. Check the API server logs:
   ```bash
   tail -f /tmp/acme-store-test.log
   ```

3. Verify Vault is accessible:
   ```bash
   curl -s http://localhost:8200/v1/sys/health
   ```

4. Verify Pebble is accessible:
   ```bash
   curl -sk https://localhost:14000/dir
   ```

## Manual Testing

You can also run the test environment and manually test the API:

```bash
# Start environment
make test-setup

# Start API server manually
VAULT_SKIP_VERIFY=true \
CONFIG_FILE=./testdata/config/test.yml \
VAULT_TOKEN=root \
DIGITALOCEAN_TOKEN=test \
ACME_TLS_INSECURE=1 \
./acme-store

# In another terminal, test the API
curl http://127.0.0.1:15872/api/healthz
curl http://127.0.0.1:15872/api/version
curl http://127.0.0.1:15872/api/domains

# Add a domain
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain": "test.example.com"}'

# Clean up
make test-cleanup
```

## CI/CD Integration

The integration tests are designed to run in CI/CD pipelines. See `.github/workflows/integration-test.yml` for GitHub Actions configuration.

## Notes

- The test environment uses **development mode** for all services - not suitable for production
- Pebble uses self-signed certificates, so TLS verification must be disabled
- Test data is cleaned up between runs to ensure test isolation
- The DNS server (`challtestsrv`) is only accessible within the Docker network

