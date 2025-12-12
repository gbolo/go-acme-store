# API Documentation

## Base URL

```
http://127.0.0.1:15872/api
```

All API endpoints are prefixed with `/api`.

## Endpoints Overview

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/version` | Application version and build info |
| GET | `/healthz` | Health check |
| GET | `/config` | View non-sensitive configuration |
| GET | `/certs` | List all certificates |
| GET | `/domains` | List managed domains |
| POST | `/domains` | Add a domain |
| DELETE | `/domains/{domain}` | Remove a domain |
| POST | `/trigger-renewal` | Trigger certificate renewal |

---

## System Endpoints

### GET /api/version

Returns application version and build information.

**Response:**
```json
{
  "name": "acme-store",
  "version": "1.0.0",
  "build_time": "2025-12-12T10:30:00Z",
  "git_commit": "abc1234"
}
```

**Example:**
```bash
curl http://127.0.0.1:15872/api/version
```

---

### GET /api/healthz

Health check endpoint for monitoring.

**Response:**
```json
{
  "status": "success"
}
```

**Status Codes:**
- `200` - Service is healthy

**Example:**
```bash
curl http://127.0.0.1:15872/api/healthz
```

---

### GET /api/config

Returns the current application configuration (non-sensitive values only).

**Response:**
```json
{
  "log": {
    "level": "INFO"
  },
  "server": {
    "bind_address": "127.0.0.1",
    "bind_port": "15872"
  },
  "acme": {
    "directory": "https://acme-staging-v02.api.letsencrypt.org/directory",
    "account_email": "admin@example.com",
    "dns_server": "8.8.8.8:53",
    "dns_provider": "digitalocean"
  },
  "vault": {
    "address": "http://127.0.0.1:8200",
    "kv2_mount": "kv",
    "kv2_secret_path": "platform/acme",
    "tls_skip_verify": false
  }
}
```

**Security Note:**
Sensitive values are **excluded** from the response:
- `vault.token` - Never exposed
- `acme.digitalocean_token` - Never exposed

**Use Cases:**
- Verify configuration without accessing the file
- Debugging configuration issues
- Monitoring/auditing active settings
- Display current settings in UI

**Example:**
```bash
curl http://127.0.0.1:15872/api/config | jq
```

---

## Certificate Endpoints

### GET /api/certs

List all certificates with full details.

**Response:**
```json
[
  {
    "common_name": "example.com",
    "sans": ["example.com", "www.example.com"],
    "leaf_cert_pem": "-----BEGIN CERTIFICATE-----\n...",
    "cert_chain_pem": "-----BEGIN CERTIFICATE-----\n...",
    "cert_chain_url": "https://acme-v02.api.letsencrypt.org/...",
    "issuers": ["Let's Encrypt Authority X3"],
    "issued_on": "2025-12-12 10:30:00",
    "expires_on": "2026-03-12 10:30:00",
    "managed": true,
    "status": "issued",
    "error": ""
  },
  {
    "common_name": "pending.example.com",
    "sans": ["pending.example.com"],
    "leaf_cert_pem": "",
    "cert_chain_pem": "",
    "cert_chain_url": "",
    "issuers": [],
    "issued_on": "",
    "expires_on": "",
    "managed": true,
    "status": "pending",
    "error": ""
  },
  {
    "common_name": "failed.example.com",
    "sans": ["failed.example.com"],
    "leaf_cert_pem": "",
    "cert_chain_pem": "",
    "cert_chain_url": "",
    "issuers": [],
    "issued_on": "",
    "expires_on": "",
    "managed": true,
    "status": "failed",
    "error": "DNS validation failed: no TXT record found"
  }
]
```

**Status Values:**
- `pending` - Certificate issuance in progress
- `issued` - Certificate successfully issued
- `failed` - Certificate issuance failed (see `error` field)

**Example:**
```bash
# Get all certificates
curl http://127.0.0.1:15872/api/certs | jq

# Get only managed certificates
curl http://127.0.0.1:15872/api/certs | jq '.[] | select(.managed == true)'

# Get failed certificates
curl http://127.0.0.1:15872/api/certs | jq '.[] | select(.status == "failed")'
```

---

## Domain Management Endpoints

### GET /api/domains

List all managed domains.

**Response:**
```json
{
  "domains": ["example.com", "api.example.com", "*.example.com"],
  "count": 3
}
```

**Example:**
```bash
curl http://127.0.0.1:15872/api/domains | jq
```

---

### POST /api/domains

Add a new domain to be managed.

**Request Body:**
```json
{
  "domain": "example.com",
  "sans": ["www.example.com", "api.example.com"]
}
```

**Parameters:**
- `domain` (required) - The primary domain (Common Name)
- `sans` (optional) - Array of Subject Alternative Names

**Response (Success):**
```json
{
  "message": "domain example.com added successfully",
  "domain": "example.com",
  "sans": ["www.example.com", "api.example.com"]
}
```

**Status Codes:**
- `201` - Domain added successfully
- `400` - Invalid request (missing domain, etc.)

**Behavior:**
- Creates placeholder certificate with `status: "pending"`
- Triggers automatic certificate issuance
- If domain previously existed as unmanaged, flips `managed: true`

**Example:**
```bash
# Add domain without SANs
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain": "example.com"}'

# Add domain with SANs
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "example.com",
    "sans": ["www.example.com", "api.example.com"]
  }'

# Add wildcard domain
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "*.example.com",
    "sans": ["example.com"]
  }'
```

---

### DELETE /api/domains/{domain}

Remove a domain from management.

**URL Parameters:**
- `domain` (required) - The domain to remove

**Query Parameters:**
- `delete_cert` (optional, default: `false`) - Whether to delete certificate from Vault
  - `false` - Soft delete (mark as unmanaged, keep certificate)
  - `true` - Hard delete (remove completely from Vault)

**Response (Soft Delete):**
```json
{
  "message": "domain example.com removed successfully",
  "domain": "example.com"
}
```

**Response (Hard Delete):**
```json
{
  "message": "domain example.com removed and certificate deleted",
  "domain": "example.com"
}
```

**Status Codes:**
- `200` - Domain removed successfully
- `400` - Missing domain parameter
- `404` - Domain not found

**Example:**
```bash
# Soft delete (mark as unmanaged)
curl -X DELETE http://127.0.0.1:15872/api/domains/example.com

# Hard delete (remove from Vault)
curl -X DELETE "http://127.0.0.1:15872/api/domains/example.com?delete_cert=true"
```

---

## Operations Endpoints

### POST /api/trigger-renewal

Manually trigger a certificate renewal check for all managed domains.

**Response:**
```json
{
  "message": "certificate renewal triggered, check logs for progress"
}
```

**Status Codes:**
- `202` - Renewal triggered (async operation started)

**Behavior:**
- Runs asynchronously (doesn't block)
- Checks all managed domains
- Renews certificates expiring soon
- Skips unmanaged certificates
- Check logs for detailed progress

**Example:**
```bash
curl -X POST http://127.0.0.1:15872/api/trigger-renewal

# Monitor logs
journalctl -u acme-store -f
```

---

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "error message describing what went wrong"
}
```

**Common Status Codes:**
- `200` - Success
- `201` - Created successfully
- `202` - Accepted (async operation)
- `400` - Bad request (invalid input)
- `404` - Not found
- `500` - Internal server error

---

## Security Considerations

### Current State
- **No authentication** by default
- Suitable for **internal/trusted networks** only
- All endpoints are **publicly accessible** on bind address

### Production Recommendations

1. **Restrict Bind Address**
   ```yaml
   server:
     bind_address: 127.0.0.1  # localhost only
   ```

2. **Use Reverse Proxy with Auth**
   ```nginx
   location /api {
       auth_basic "Restricted";
       auth_basic_user_file /etc/nginx/.htpasswd;
       proxy_pass http://127.0.0.1:15872;
   }
   ```

3. **Firewall Rules**
   ```bash
   # Only allow from specific IPs
   iptables -A INPUT -p tcp --dport 15872 -s 10.0.0.0/8 -j ACCEPT
   iptables -A INPUT -p tcp --dport 15872 -j DROP
   ```

4. **VPN Access**
   - Run on private network
   - Access via VPN only

5. **API Key Authentication** (future enhancement)
   - Add `X-API-Key` header requirement
   - Implement key validation middleware

---

## Rate Limiting

**Current State:**
- No rate limiting implemented
- Use reverse proxy for rate limiting if needed

**Nginx Example:**
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

location /api {
    limit_req zone=api burst=20;
    proxy_pass http://127.0.0.1:15872;
}
```

---

## Monitoring Examples

### Check Service Health
```bash
#!/bin/bash
response=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:15872/api/healthz)
if [ "$response" = "200" ]; then
    echo "Service is healthy"
else
    echo "Service is down!"
    exit 1
fi
```

### List Expiring Certificates
```bash
curl -s http://127.0.0.1:15872/api/certs | \
  jq '.[] | select(.managed == true) | 
    {domain: .common_name, expires: .expires_on, status: .status}'
```

### Count Certificates by Status
```bash
curl -s http://127.0.0.1:15872/api/certs | \
  jq 'group_by(.status) | 
    map({status: .[0].status, count: length})'
```

### Monitor Failed Certificates
```bash
curl -s http://127.0.0.1:15872/api/certs | \
  jq '.[] | select(.status == "failed") | 
    {domain: .common_name, error: .error}'
```

---

## Integration Examples

### Prometheus Monitoring
```python
import requests
from prometheus_client import Gauge, start_http_server

certs_total = Gauge('acme_certificates_total', 'Total certificates')
certs_valid = Gauge('acme_certificates_valid', 'Valid certificates')
certs_failed = Gauge('acme_certificates_failed', 'Failed certificates')

def collect_metrics():
    resp = requests.get('http://127.0.0.1:15872/api/certs')
    certs = resp.json()
    
    certs_total.set(len(certs))
    certs_valid.set(len([c for c in certs if c['status'] == 'issued']))
    certs_failed.set(len([c for c in certs if c['status'] == 'failed']))

start_http_server(8000)
while True:
    collect_metrics()
    time.sleep(60)
```

### Alert on Failed Certificates
```bash
#!/bin/bash
failed=$(curl -s http://127.0.0.1:15872/api/certs | jq '[.[] | select(.status == "failed")] | length')

if [ "$failed" -gt 0 ]; then
    echo "⚠️ Alert: $failed certificate(s) failed issuance!"
    curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK \
      -d "{\"text\": \"$failed ACME certificates failed!\"}"
fi
```

---

## Future Enhancements

Planned API improvements:
- [ ] Authentication (API keys)
- [ ] Rate limiting
- [ ] Pagination for large certificate lists
- [ ] Filtering and search for certificates
- [ ] Certificate history/audit log
- [ ] Webhook notifications
- [ ] ACME account management endpoints
- [ ] Bulk domain operations

