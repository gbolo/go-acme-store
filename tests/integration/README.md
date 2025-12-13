# Integration Tests

These tests verify the complete acme-store system including API, certificate management, and Vault integration.

## Prerequisites

- Docker and Docker Compose
- Go 1.20 or later
- Make

## Running Tests

### Quick Run (All Steps)

```bash
make test-all
```

This will:
1. Start the Docker test environment
2. Run all integration tests
3. Clean up the environment

### Step by Step

```bash
# 1. Start test environment
make test-setup

# 2. Run integration tests
make test-integration

# 3. Clean up
make test-cleanup
```

### Manual Test Run

```bash
# Start environment
make test-setup

# Run tests directly
CONFIG_FILE=./testdata/config/test.yml go test -v -tags=integration ./tests/integration/...

# Clean up
make test-cleanup
```

## Test Coverage

### API Tests
- ✅ Health check endpoint
- ✅ Version endpoint
- ✅ Configuration endpoint (verify secrets excluded)
- ✅ Root redirect to UI
- ✅ UI accessibility

### Domain Management Tests
- ✅ List domains (initially empty)
- ✅ Add domain (basic)
- ✅ Add domain with SANs
- ✅ List domains after add
- ✅ Remove domain (soft delete)
- ✅ Remove domain (hard delete)
- ✅ Invalid domain rejection

### Certificate Tests
- ✅ List certificates
- ✅ Certificate status tracking (pending/issued/failed)
- ✅ Trigger manual renewal

## Test Environment

The tests use a complete Docker Compose environment:

- **Vault** (dev mode) - Port 8200
- **Pebble** (ACME server) - Port 14000
- **challtestsrv** (DNS mock) - Port 8055, 53
- **acme-dns** (DNS delegation) - Port 8053

## Debugging

### View Logs

```bash
# All services
docker-compose -f docker-compose.test.yml logs -f

# Specific service
docker-compose -f docker-compose.test.yml logs -f pebble
docker-compose -f docker-compose.test.yml logs -f vault
```

### Check Service Status

```bash
docker-compose -f docker-compose.test.yml ps
```

### Run Single Test

```bash
CONFIG_FILE=./testdata/config/test.yml go test -v -tags=integration ./tests/integration -run TestAddDomain
```

### Keep Environment Running

```bash
# Start environment
make test-setup

# Run tests (keep environment running on failure)
CONFIG_FILE=./testdata/config/test.yml go test -v -tags=integration ./tests/integration/...

# Manual testing...
curl http://localhost:15872/api/healthz

# Clean up when done
make test-cleanup
```

## Troubleshooting

### API Not Starting

**Problem:** Tests fail with "API failed to start in time"

**Solution:**
1. Check if acme-store is already running
2. Check if port 15872 is in use
3. Check logs: `docker-compose -f docker-compose.test.yml logs`

### Vault Connection Failed

**Problem:** Tests fail with Vault connection errors

**Solution:**
1. Ensure Vault is running: `curl http://localhost:8200/v1/sys/health`
2. Check Vault token is set correctly in test config
3. Restart environment: `make test-cleanup && make test-setup`

### Certificate Tests Failing

**Problem:** Certificate tests timeout or fail validation

**Solution:**
1. Check Pebble is running: `curl -k https://localhost:14000/dir`
2. Check challtestsrv: `curl http://localhost:8055/`
3. This is expected if DNS validation fails - Pebble won't issue real certificates

### Port Conflicts

**Problem:** "port already in use" errors

**Solution:**
```bash
# Find what's using the port
sudo lsof -i :8200  # Vault
sudo lsof -i :14000 # Pebble
sudo lsof -i :15872 # acme-store

# Stop conflicting services or change ports in docker-compose.test.yml
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      
      - name: Run integration tests
        run: make test-all
```

### GitLab CI Example

```yaml
integration-test:
  image: golang:1.20
  services:
    - docker:dind
  script:
    - apt-get update && apt-get install -y docker-compose
    - make test-all
```

## Adding New Tests

1. Create test function in `api_test.go`
2. Follow naming convention: `TestFeatureName`
3. Use testify assertions for clear error messages
4. Clean up resources after test (use `defer`)

Example:

```go
func TestNewFeature(t *testing.T) {
    // Setup
    domain := "new-feature.example.com"
    defer removeDomain(t, domain, true) // Cleanup
    
    // Test
    addDomain(t, domain, nil)
    
    // Verify
    resp, err := http.Get(baseURL + "/api/domains")
    require.NoError(t, err)
    defer resp.Body.Close()
    
    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)
    
    domains := result["domains"].([]interface{})
    assert.Contains(t, domains, domain)
}
```

## Performance Tests

To add performance/load testing:

```go
func TestLoadManyDomains(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    // Add 100 domains
    for i := 0; i < 100; i++ {
        domain := fmt.Sprintf("load-test-%d.example.com", i)
        addDomain(t, domain, nil)
        defer removeDomain(t, domain, true)
    }
    
    // Verify all added
    // ...
}
```

Run with: `go test -v -tags=integration ./tests/integration -timeout 10m`

## Known Limitations

1. **Certificate Issuance:** Pebble may not successfully issue certificates without proper DNS setup
2. **Timing:** Some tests may be timing-sensitive (certificate issuance can take time)
3. **Cleanup:** Tests should clean up their domains, but manual cleanup may be needed if tests crash

## Future Enhancements

- [ ] Add certificate fetcher integration tests
- [ ] Add Traefik config generation tests
- [ ] Add concurrent domain management tests
- [ ] Add certificate renewal cycle tests
- [ ] Add performance benchmarks
- [ ] Add chaos testing (service failures)

