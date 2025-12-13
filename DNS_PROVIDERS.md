# DNS Providers

`acme-store` supports multiple DNS providers for ACME DNS-01 challenges via the [libdns](https://github.com/libdns) ecosystem.

## Supported Providers

### 1. DigitalOcean

Uses the DigitalOcean DNS API for DNS-01 challenges.

**Configuration:**

```yaml
acme:
  dns_provider: digitalocean
  digitalocean_token: "your-token-here"
```

**Environment Variables:**

```bash
export ACMESTORE_ACME_DNS_PROVIDER=digitalocean
export ACMESTORE_ACME_DIGITALOCEAN_TOKEN="dop_v1_xxxxx"
```

**Requirements:**
- DigitalOcean API token with DNS write permissions
- Domain must be managed by DigitalOcean DNS

---

### 2. PowerDNS

Uses the PowerDNS API for DNS-01 challenges.

**Configuration:**

```yaml
acme:
  dns_provider: powerdns
  powerdns_server_url: "https://pdns.example.com"
  powerdns_api_token: "your-api-token"
  powerdns_server_id: "localhost"  # Optional, defaults to "localhost"
```

**Environment Variables:**

```bash
export ACMESTORE_ACME_DNS_PROVIDER=powerdns
export ACMESTORE_ACME_POWERDNS_SERVER_URL="https://pdns.example.com"
export ACMESTORE_ACME_POWERDNS_API_TOKEN="your-api-token"
export ACMESTORE_ACME_POWERDNS_SERVER_ID="localhost"
```

**Requirements:**
- PowerDNS server with API enabled
- API key with zone write permissions
- PowerDNS version 4.x or later recommended

**PowerDNS API Setup:**

1. Enable the API in `pdns.conf`:
   ```ini
   api=yes
   api-key=your-secret-api-key
   webserver=yes
   webserver-address=0.0.0.0
   webserver-port=8081
   ```

2. Restart PowerDNS:
   ```bash
   systemctl restart pdns
   ```

3. Test the API:
   ```bash
   curl -H "X-API-Key: your-secret-api-key" \
        http://localhost:8081/api/v1/servers/localhost
   ```

---

### 3. ACME-DNS

Uses ACME-DNS for DNS-01 delegation. This is useful when you don't have API access to your primary DNS provider.

**Configuration:**

```yaml
acme:
  dns_provider: acmedns
  acmedns_server_url: "http://acme-dns.example.com"
```

**Environment Variables:**

```bash
export ACMESTORE_ACME_DNS_PROVIDER=acmedns
export ACMESTORE_ACME_ACMEDNS_SERVER_URL="http://acme-dns.example.com"
```

**Requirements:**
- Running ACME-DNS server
- CNAME delegation from your domain to ACME-DNS

**ACME-DNS Setup:**

1. Deploy ACME-DNS server (see [joohoi/acme-dns](https://github.com/joohoi/acme-dns))

2. Register a new account:
   ```bash
   curl -X POST http://acme-dns.example.com/register
   ```

3. Add CNAME record to your DNS:
   ```
   _acme-challenge.example.com. CNAME <subdomain>.acme-dns.example.com.
   ```

---

## Switching Providers

To switch DNS providers, update the `dns_provider` setting and provide the appropriate credentials:

```yaml
acme:
  # Change this to: digitalocean, powerdns, or acmedns
  dns_provider: powerdns
  
  # Provide credentials for the selected provider
  powerdns_server_url: "https://pdns.example.com"
  powerdns_api_token: "your-api-token"
```

## Adding New Providers

`acme-store` uses [libdns](https://github.com/libdns) providers. To add support for a new DNS provider:

1. Add the libdns provider dependency:
   ```bash
   go get github.com/libdns/yourprovider@latest
   ```

2. Add solver function in `pkg/acme/solvers.go`:
   ```go
   func getYourProviderSolver() *certmagic.DNS01Solver {
       solver := &certmagic.DNS01Solver{}
       solver.DNSManager.DNSProvider = &yourprovider.Provider{
           APIToken: viper.GetString("acme.yourprovider_api_token"),
       }
       return solver
   }
   ```

3. Add case in `pkg/acme/acme.go`:
   ```go
   case "yourprovider":
       dnsSolver = getYourProviderSolver()
   ```

4. Update configuration files and documentation

## Troubleshooting

### DNS Propagation Issues

If challenges are failing, check DNS propagation:

```bash
# Check if TXT record is visible
dig +short TXT _acme-challenge.example.com @8.8.8.8
```

### PowerDNS API Issues

```bash
# Test PowerDNS API connectivity
curl -H "X-API-Key: your-api-token" \
     https://pdns.example.com/api/v1/servers

# Check zone exists
curl -H "X-API-Key: your-api-token" \
     https://pdns.example.com/api/v1/servers/localhost/zones/example.com
```

### Provider-Specific Logs

Enable DEBUG logging to see provider-specific errors:

```yaml
log:
  level: DEBUG
```

## Provider Comparison

| Provider | Pros | Cons | Best For |
|----------|------|------|----------|
| **DigitalOcean** | Easy setup, reliable API | Requires DO DNS | DO users |
| **PowerDNS** | Self-hosted, full control | Requires setup | Self-hosted DNS |
| **ACME-DNS** | Works with any DNS | Requires CNAME delegation | Limited DNS API access |

## Security Considerations

1. **API Tokens**: Store tokens in environment variables, never commit to git
2. **Least Privilege**: Use API tokens with minimal required permissions
3. **Token Rotation**: Regularly rotate API tokens
4. **Network Security**: Restrict API access to trusted networks when possible

## References

- [libdns Providers](https://github.com/libdns)
- [ACME DNS-01 Challenge](https://letsencrypt.org/docs/challenge-types/#dns-01-challenge)
- [PowerDNS API Documentation](https://doc.powerdns.com/authoritative/http-api/)
- [ACME-DNS Documentation](https://github.com/joohoi/acme-dns)
