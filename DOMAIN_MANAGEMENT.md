# Domain Management Guide

## Overview

As of the latest version, domains are managed via **HashiCorp Vault** instead of the configuration file. This provides:

- **Dynamic management** - Add/remove domains without restarting the application
- **Centralized state** - Single source of truth for domain configuration
- **API-driven** - Manage domains programmatically via REST API or Web UI
- **Audit trail** - Vault provides audit logging for all changes

## Management Methods

You can manage domains using:

1. **Web UI** - User-friendly interface (recommended for manual management)
2. **REST API** - For automation and programmatic access
3. **Direct Vault access** - Advanced users only

## Web UI Management

### Accessing Domain Manager

1. Open the web UI: `http://127.0.0.1:15872/ui`
2. Click the **"Manage Domains"** button in the action bar
3. A modal will open showing:
   - Current managed domains list
   - Form to add new domains
   - Remove buttons for each domain

### Adding a Domain via UI

1. Click **"Manage Domains"** button
2. Enter **domain name** (Common Name) in the first field (e.g., `example.com` or `*.example.com`)
3. **Optional**: Enter **Subject Alternative Names (SANs)** in the second field
   - One domain per line or comma-separated
   - Example: `www.example.com, api.example.com`
4. Click **"Add Domain"** button
5. Success message will appear
6. Domain will be added to the list
7. Certificate will be automatically requested immediately with all SANs

**Subject Alternative Names (SANs):**
- SANs allow one certificate to be valid for multiple domains
- Example: A certificate for `example.com` with SANs `www.example.com, api.example.com`
- The main domain (CN) is automatically included
- All SANs must pass DNS-01 challenges

**Re-adding Previously Removed Domains:**
- If domain was previously managed and still has a certificate in Vault
- Certificate's `managed` flag is automatically flipped back to `true`
- Existing certificate will be renewed on next cycle
- No need to re-issue certificate if still valid

### Removing a Domain via UI

The UI now provides **two removal options** for each domain:

#### Option 1: Unmanage (Soft Delete) 🟠
1. Click **"Manage Domains"** button
2. Find the domain in the list
3. Click the **"Unmanage"** button (orange)
4. Confirm in the dialog
5. Domain removed from managed list
6. Certificate marked as "unmanaged" but remains in Vault
7. Certificate visible in dashboard with "Unmanaged" badge

**Use when:**
- You want to stop renewing but keep certificate for reference
- Temporarily disabling automatic renewal
- Testing or troubleshooting

#### Option 2: Delete (Hard Delete) 🔴
1. Click **"Manage Domains"** button
2. Find the domain in the list
3. Click the **"Delete"** button (red)
4. Read the **permanent deletion warning**
5. Confirm deletion
6. Domain removed from managed list
7. Certificate **completely deleted** from Vault
8. Certificate no longer visible anywhere

**Use when:**
- Permanently removing a domain
- Cleaning up failed/invalid domains
- Certificate no longer needed
- Freeing up Vault storage

**⚠️ Warning**: Hard delete is permanent and cannot be undone!

### Certificate Issuance After Adding Domain

When you add a domain via the Web UI:
1. Domain is added to Vault
2. Certificate issuance is automatically triggered
3. Check logs for ACME progress
4. Certificate should appear in the dashboard within a few minutes

If using the API directly, you can manually trigger certificate issuance:
```bash
curl -X POST http://127.0.0.1:15872/api/trigger-renewal
```

## API Endpoints

### List All Managed Domains

```bash
GET /api/domains
```

**Example:**
```bash
curl http://127.0.0.1:15872/api/domains
```

**Response:**
```json
{
  "domains": ["example.com", "*.example.com", "api.example.com"],
  "count": 3
}
```

### Add a Domain

**Simple domain (no SANs):**
```bash
POST /api/domains
Content-Type: application/json

{
  "domain": "example.com"
}
```

**Example:**
```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}'
```

**Domain with SANs:**
```bash
POST /api/domains
Content-Type: application/json

{
  "domain": "example.com",
  "sans": ["www.example.com", "api.example.com", "app.example.com"]
}
```

**Example:**
```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "example.com",
    "sans": ["www.example.com", "api.example.com"]
  }'
```

**Response (201 Created):**
```json
{
  "message": "domain example.com added successfully",
  "domain": "example.com",
  "sans": ["www.example.com", "api.example.com"]
}
```

**Wildcard domains with SANs:**
```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "*.example.com",
    "sans": ["example.com"]
  }'
```

**Note**: When requesting a certificate, the `domain` (Common Name) is automatically included along with all specified SANs.

### Remove a Domain

**Soft Delete (Default)** - Marks certificate as unmanaged but keeps it in Vault:
```bash
DELETE /api/domains/{domain}
```

**Example:**
```bash
curl -X DELETE http://127.0.0.1:15872/api/domains/example.com
```

**Hard Delete** - Completely removes certificate from Vault:
```bash
DELETE /api/domains/{domain}?delete_cert=true
```

**Example:**
```bash
curl -X DELETE "http://127.0.0.1:15872/api/domains/example.com?delete_cert=true"
```

**Response (200 OK):**
```json
{
  "message": "domain example.com removed successfully",
  "domain": "example.com"
}
```

**Behavior:**
- **Soft Delete**: Certificate marked with `managed: false`, visible in dashboard with "Unmanaged" badge
- **Hard Delete**: Certificate completely removed from Vault, no longer visible
- **Renewal**: Unmanaged certificates are NOT renewed automatically

### Trigger Certificate Renewal

Manually trigger certificate issuance/renewal for all managed domains:

```bash
POST /api/trigger-renewal
```

**Example:**
```bash
curl -X POST http://127.0.0.1:15872/api/trigger-renewal
```

**Response (202 Accepted):**
```json
{
  "message": "certificate renewal triggered, check logs for progress"
}
```

**Use cases:**
- After adding a new domain via API (Web UI does this automatically)
- To force immediate renewal check
- Testing certificate issuance
- Recovery after configuration changes

## Migration from Config File

If you're upgrading from a version that used `config.yml` for domain management:

### Step 1: Get your current domains

Check your `config.yml` for the domains list:

```yaml
domains:
  - example.com
  - "*.example.com"
  - api.example.com
```

### Step 2: Add domains via API

For each domain in your config, add it via the API:

```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}'

curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"*.example.com"}'

curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"api.example.com"}'
```

### Step 3: Verify

List domains to confirm they're all added:

```bash
curl http://127.0.0.1:15872/api/domains
```

### Step 4: Remove old config (optional)

You can now remove the `domains:` section from your `config.yml`.

## Bulk Operations

### Add Multiple Domains (Shell Script)

```bash
#!/bin/bash
DOMAINS=(
  "example.com"
  "*.example.com"
  "api.example.com"
  "www.example.com"
)

for domain in "${DOMAINS[@]}"; do
  echo "Adding domain: $domain"
  curl -X POST http://127.0.0.1:15872/api/domains \
    -H "Content-Type: application/json" \
    -d "{\"domain\":\"$domain\"}"
  echo ""
done
```

### Add from File

If you have domains in a file (one per line):

```bash
#!/bin/bash
while IFS= read -r domain; do
  echo "Adding domain: $domain"
  curl -X POST http://127.0.0.1:15872/api/domains \
    -H "Content-Type: application/json" \
    -d "{\"domain\":\"$domain\"}"
  echo ""
done < domains.txt
```

## How It Works

### Storage in Vault

Domains are stored in Vault at:
```
{kv2_mount}/data/{kv2_secret_path}/managed-domains
```

With the default configuration, this would be:
```
kv/data/platform/acme/managed-domains
```

The data structure is:
```json
{
  "data": {
    "domains": ["example.com", "*.example.com", "api.example.com"]
  }
}
```

### ACME Daemon Behavior

- The daemon runs automatically every 24 hours
- It can also be triggered manually via `/api/trigger-renewal`
- It fetches the current list of managed domains from Vault
- For each domain, it checks if the certificate needs renewal
- Certificates are renewed if they expire within 21 days
- New domains get certificates immediately when triggered

### Certificate Lifecycle

1. **Domain Added** → Certificate will be obtained immediately (auto-triggered)
2. **Certificate Obtained** → Stored in Vault at `{path}/domain/{domain}` with `managed: true`
3. **Renewal Check** → Every 24 hours
4. **Certificate Renewed** → Updated in Vault automatically (if managed)
5. **Domain Removed** → Certificate marked as `managed: false` (soft delete)
6. **Hard Delete** → Certificate completely removed from Vault

**Certificate States:**
- **Managed (`managed: true`)**: 
  - Appears in managed domains list
  - Automatically renewed when expiring
  - Shows "Managed" badge in UI
  
- **Unmanaged (`managed: false`)**: 
  - Removed from managed domains list
  - NOT renewed automatically
  - Remains visible in dashboard with "Unmanaged" badge
  - Certificate data preserved for reference

**Issuance Status:**
- **Pending (`status: "pending"`)**: 
  - Certificate issuance in progress
  - Domain appears immediately in UI
  - Animated "Pending" badge shown
  - No certificate data yet
  
- **Issued (`status: "issued"`)**: 
  - Certificate successfully obtained
  - Full certificate details available
  - Can be viewed and downloaded
  
- **Failed (`status: "failed"`)**: 
  - Certificate issuance failed
  - Error message displayed in UI
  - Shows "Failed" badge
  - Can retry by removing and re-adding domain

## Error Handling

### Domain Already Exists

```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}'
```

**Response (400 Bad Request):**
```json
{
  "error": "failed to add domain: domain example.com is already managed"
}
```

### Domain Not Found

```bash
curl -X DELETE http://127.0.0.1:15872/api/domains/nonexistent.com
```

**Response (404 Not Found):**
```json
{
  "error": "failed to remove domain: domain nonexistent.com is not managed"
}
```

### Invalid Request

```bash
curl -X POST http://127.0.0.1:15872/api/domains \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response (400 Bad Request):**
```json
{
  "error": "domain field is required"
}
```

## Best Practices

1. **Start Small**: Add one domain first and verify certificate issuance
2. **Wildcard Certs**: Use `*.example.com` for wildcard certificates
3. **DNS-01 Challenge**: Required for wildcard certificates
4. **Monitoring**: Check logs for certificate issuance/renewal status
5. **Backup**: Vault should be backed up regularly for disaster recovery
6. **Validation**: Ensure DNS is properly configured before adding domains

## Troubleshooting

### No domains loaded on startup

**Log message:**
```
WARN failed to load managed domains from vault: ...
WARN if this is the first run, add domains via the API: POST /api/domains
```

**Solution**: This is normal on first run. Add domains via the API.

### Domain added but certificate not issued

**Check:**
1. Wait for next daemon cycle (up to 24 hours) or restart the application
2. Check logs for ACME errors: `grep -i "failed to obtain" /var/log/acme-store.log`
3. Verify DNS configuration for the domain
4. Check DigitalOcean API token is valid

### Certificate not renewing

**Check:**
1. Domain is still in managed list: `curl http://127.0.0.1:15872/api/domains`
2. Check logs for renewal attempts
3. Verify certificate is approaching expiration (< 21 days)

## Security Considerations

### API Access

Currently, the API endpoints are **not authenticated**. For production use:

1. **Use a Reverse Proxy** with authentication (nginx, Caddy, etc.)
2. **Firewall Rules** to restrict access
3. **VPN/Private Network** for internal-only access
4. **TLS** to encrypt communications

Example nginx configuration:

```nginx
location /api/domains {
    auth_basic "Domain Management";
    auth_basic_user_file /etc/nginx/.htpasswd;
    proxy_pass http://127.0.0.1:15872;
}
```

### Vault Token Rotation

Ensure your Vault token has appropriate permissions and rotate regularly:

```bash
vault token renew
vault token lookup
```

## Advanced Usage

### Integrate with CI/CD

Add domain management to your deployment pipeline:

```yaml
# GitLab CI example
deploy:
  script:
    - |
      curl -X POST https://acme-manager.example.com/api/domains \
        -H "Content-Type: application/json" \
        -d "{\"domain\":\"${CI_ENVIRONMENT_SLUG}.example.com\"}"
```

### Automation with Terraform

```hcl
resource "null_resource" "add_domain" {
  provisioner "local-exec" {
    command = <<EOF
      curl -X POST http://127.0.0.1:15872/api/domains \
        -H "Content-Type: application/json" \
        -d '{"domain":"${var.domain_name}"}'
    EOF
  }
}
```

### Monitoring with Prometheus

Export metrics for managed domains count:

```bash
curl -s http://127.0.0.1:15872/api/domains | jq '.count'
```

## Support

For issues or questions:
- Check application logs
- Review Vault audit logs
- Verify API responses
- Check certificate status via `/api/certs`

