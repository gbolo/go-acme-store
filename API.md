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
| GET | `/certs/{domain}/private-key` | Get private key for a domain |
| DELETE | `/certs/{domain}` | Delete a certificate |
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

### GET /api/certs/{domain}/private-key

Get the private key for a specific domain's certificate.

**URL Parameters:**
- `domain` (required) - The domain name

**Response:**
```json
{
  "domain": "example.com",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgk..."
}
```

**Status Codes:**
- `200` - Success
- `404` - Certificate not found for domain
- `500` - Failed to retrieve certificate
- `503` - Keystore unavailable

**Security Note:**
This endpoint returns sensitive private key material. Ensure proper access controls are in place.

**Example:**
```bash
# Get private key for a domain
curl http://127.0.0.1:15872/api/certs/example.com/private-key | jq

# Save private key to file
curl -s http://127.0.0.1:15872/api/certs/example.com/private-key | \
  jq -r '.private_key' > example.com.key

# Set proper permissions
chmod 600 example.com.key
```

---

### DELETE /api/certs/{domain}

Delete a certificate for a specific domain.

**URL Parameters:**
- `domain` (required) - The domain name

**Response (Success):**
```json
{
  "message": "certificate for domain example.com deleted successfully",
  "domain": "example.com"
}
```

**Status Codes:**
- `200` - Certificate deleted successfully
- `400` - Cannot delete managed certificate (must unmanage first)
- `404` - Certificate not found
- `500` - Failed to delete certificate
- `503` - Keystore unavailable

**Important:**
- You cannot delete a managed certificate directly
- First remove it from managed domains using `DELETE /api/domains/{domain}`
- Then you can delete the certificate

**Example:**
```bash
# Try to delete a managed certificate (will fail)
curl -X DELETE http://127.0.0.1:15872/api/certs/example.com

# Unmanage the domain first
curl -X DELETE http://127.0.0.1:15872/api/domains/example.com

# Now delete the certificate
curl -X DELETE http://127.0.0.1:15872/api/certs/example.com
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
```