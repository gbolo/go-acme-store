# Integration Tests Quick Start

## TL;DR

```bash
# Run everything
make test-all

# Or step by step
make test-setup       # Start Docker environment
make test-integration # Run tests
make test-cleanup     # Clean up
```

## What Gets Tested

✅ **API Endpoints**
- Health check
- Version info
- Configuration (secrets excluded)
- Root redirect
- UI accessibility

✅ **Domain Management**
- List domains
- Add domain (basic + with SANs)
- Remove domain (soft + hard delete)
- Invalid input rejection

✅ **Certificate Management**
- List certificates
- Status tracking (pending/issued/failed)
- Manual renewal trigger

## Test Output Example

```
=== RUN   TestHealthCheck
--- PASS: TestHealthCheck (0.01s)
=== RUN   TestVersion
--- PASS: TestVersion (0.01s)
=== RUN   TestConfig
--- PASS: TestConfig (0.01s)
=== RUN   TestAddDomain
--- PASS: TestAddDomain (0.05s)
=== RUN   TestListCertificates
    api_test.go:215: Certificate status for test.example.com: pending
--- PASS: TestListCertificates (2.03s)
PASS
ok      go-acme-store/tests/integration 5.123s
```

## Quick Commands

```bash
# Check test environment status
docker-compose -f docker-compose.test.yml ps

# View logs
docker-compose -f docker-compose.test.yml logs -f

# Run specific test
CONFIG_FILE=./testdata/config/test.yml \
  go test -v -tags=integration ./tests/integration -run TestAddDomain

# Keep environment running for debugging
make test-setup
# ... manual testing ...
make test-cleanup
```

## Troubleshooting

### Tests fail immediately

**Check if environment is running:**
```bash
make test-setup
docker-compose -f docker-compose.test.yml ps
```

### Port conflicts

**Find what's using ports:**
```bash
sudo lsof -i :8200  # Vault
sudo lsof -i :14000 # Pebble
sudo lsof -i :15872 # acme-store
```

### Certificate tests fail

This is expected! Pebble requires proper DNS setup for real certificate issuance. The tests verify the API and workflow, not actual certificate validation.

## CI/CD

GitHub Actions workflow included at `.github/workflows/integration-test.yml`

Runs automatically on:
- Push to `main` or `develop`
- Pull requests

## Next Steps

1. ✅ Run tests locally
2. ✅ Verify all pass
3. ✅ Add to CI/CD pipeline
4. ✅ Extend tests as needed

See `tests/integration/README.md` for detailed documentation.

