# Certificate Status Tracking

## Overview

The application now provides real-time status tracking for certificate issuance. When you add a domain, it appears immediately in the dashboard with its current status.

## Certificate Status States

### 1. **Pending** 🟡
- **When**: Immediately after adding a domain
- **Appearance**: Orange "Pending" badge with pulsing animation
- **Details**: Shows message "Certificate issuance in progress... This may take a few minutes."
- **Actions**: No action buttons available (certificate doesn't exist yet)
- **Icon**: Animated hourglass icon

**What's happening:**
- ACME challenge initiated
- DNS records being verified
- Certificate being requested from Let's Encrypt
- Usually takes 1-5 minutes

### 2. **Issued** 🟢
- **When**: Certificate successfully obtained
- **Appearance**: Status badge (Valid/Expiring Soon/Expired) based on expiry
- **Details**: Shows all certificate information (dates, issuers, SANs)
- **Actions**: "View Certificate" and "View Full Chain" buttons available
- **Data**: Full certificate data available for viewing/copying

### 3. **Failed** 🔴
- **When**: Certificate issuance encountered an error
- **Appearance**: Red "Failed" badge
- **Details**: Shows error message from ACME provider
- **Actions**: No action buttons (no certificate exists)
- **Error**: Full error message displayed

**Common errors:**
- DNS validation failed
- Domain ownership cannot be verified
- API rate limits exceeded
- DNS provider API issues

## Visual Representation

### Pending Certificate Card
```
┌─────────────────────────────────────────────────┐
│ example.com              [Managed] [Pending]    │
│ Issuer: Pending issuance                        │
│                                                 │
│ ⏳ Certificate issuance in progress...          │
│    This may take a few minutes.                 │
│                                                 │
│ SANs:                                           │
│ • www.example.com                               │
│ • api.example.com                               │
└─────────────────────────────────────────────────┘
```

### Issued Certificate Card
```
┌─────────────────────────────────────────────────┐
│ example.com              [Managed] [Valid]      │
│ Issuer: Let's Encrypt Authority X3              │
│                                                 │
│ Issued On: Dec 12, 2025                         │
│ Expires On: Mar 12, 2026                        │
│ Days Until Expiry: 90 days                      │
│ Certificate Chain: 2 issuer(s)                  │
│                                                 │
│ SANs:                                           │
│ • example.com                                   │
│ • www.example.com                               │
│ • api.example.com                               │
│                                                 │
│ [🎓 View Certificate] [🔗 View Full Chain]      │
└─────────────────────────────────────────────────┘
```

### Failed Certificate Card
```
┌─────────────────────────────────────────────────┐
│ example.com              [Managed] [Failed]     │
│ Issuer: Pending issuance                        │
│                                                 │
│ ❌ Error:                                       │
│    DNS validation failed: no TXT record found   │
│    for _acme-challenge.example.com              │
│                                                 │
│ SANs:                                           │
│ • www.example.com                               │
└─────────────────────────────────────────────────┘
```

## User Workflow

### Adding a New Domain

1. **Click "Manage Domains"**
2. **Enter domain and SANs**
3. **Click "Add Domain"**
4. **Immediate feedback**: "Domain added successfully! Triggering certificate issuance..."
5. **Domain appears** in dashboard with "Pending" status
6. **Watch status**: Refresh to see progress
7. **Certificate issued**: Status changes to "Valid" with full details

### Timeline

```
0s    Domain added → Placeholder created (status: "pending")
1s    ACME process triggered
5-30s DNS challenges being validated
30-90s Certificate issued → Status: "issued"
      OR
      Error occurred → Status: "failed" with error message
```

## Status Transitions

```
[Add Domain]
     ↓
[Pending] ──(Success)──→ [Issued] ──(Expiring)──→ [Expiring Soon]
     ↓                      ↓
     └──(Error)──→ [Failed] └──(Expired)──→ [Expired]
```

## Error Recovery

### If Certificate Issuance Fails

1. **Read the error message** in the UI
2. **Fix the underlying issue** (DNS configuration, API credentials, etc.)
3. **Remove the domain** (soft delete)
4. **Re-add the domain** to retry
5. **Watch the status** - should transition to "Issued"

**Common fixes:**
- Verify DNS records are correct
- Check DigitalOcean API token is valid
- Ensure domain ownership is verified
- Wait if rate limits are hit

## API Response Format

### Pending Certificate
```json
{
  "common_name": "example.com",
  "sans": ["www.example.com"],
  "status": "pending",
  "managed": true,
  "leaf_cert_pem": "",
  "issuers": [],
  "issued_on": "",
  "expires_on": ""
}
```

### Issued Certificate
```json
{
  "common_name": "example.com",
  "sans": ["example.com", "www.example.com"],
  "status": "issued",
  "managed": true,
  "leaf_cert_pem": "-----BEGIN CERTIFICATE-----...",
  "issuers": ["Let's Encrypt Authority X3"],
  "issued_on": "2025-12-12 10:30:00",
  "expires_on": "2026-03-12 10:30:00"
}
```

### Failed Certificate
```json
{
  "common_name": "example.com",
  "sans": ["www.example.com"],
  "status": "failed",
  "managed": true,
  "error": "DNS validation failed: no TXT record found for _acme-challenge.example.com",
  "leaf_cert_pem": "",
  "issuers": [],
  "issued_on": "",
  "expires_on": ""
}
```

## Monitoring Status

### Watch Status in Real-Time

**Option 1: Web UI**
- Open dashboard
- Click "Refresh" button periodically
- Watch "Pending" badges change to "Valid" or "Failed"

**Option 2: API Polling**
```bash
# Poll every 10 seconds
watch -n 10 'curl -s http://127.0.0.1:15872/api/certs | jq ".[] | {domain: .common_name, status: .status, error: .error}"'
```

**Option 3: Logs**
```bash
# Follow logs to see issuance progress
tail -f /var/log/acme-store.log
```

## Best Practices

1. **Don't spam refresh** - Certificate issuance takes time (30-90 seconds typically)
2. **Check logs** for detailed progress information
3. **Fix errors immediately** - Failed certificates should be investigated and retried
4. **Monitor pending domains** - If stuck in pending >5 minutes, check logs
5. **Test with staging** - Use Let's Encrypt staging for testing to avoid rate limits

## Statistics Impact

The statistics dashboard counts certificates by status:

- **Total Certificates**: All certificates (pending + issued + failed)
- **Valid**: Only successfully issued, unexpired certificates
- **Expiring Soon**: Successfully issued certificates expiring within 30 days
- **Expired**: Successfully issued but expired certificates

**Note**: Pending and failed certificates don't count toward valid/expiring/expired.

## Troubleshooting

### Domain stuck in "Pending" status

**Possible causes:**
1. ACME process still running (be patient, can take 5+ minutes)
2. DNS challenge validation slow
3. Process crashed/errored without updating status

**Solutions:**
1. Check application logs: `journalctl -u acme-store -f`
2. Wait 5 minutes and refresh
3. If stuck >10 minutes, check logs for errors
4. Try removing and re-adding the domain

### Certificate shows "Failed" status

**Steps:**
1. Read the error message in the UI
2. Common issues:
   - DNS not configured correctly
   - API credentials invalid
   - Rate limit exceeded
   - Domain ownership cannot be verified
   - **Invalid domain/TLD** (e.g., `.invalid`, `.local`, `.test`)
3. Fix the underlying issue
4. Remove the domain (soft delete or hard delete)
5. Re-add to retry issuance

**Invalid Domain Example:**
```
Error: Invalid identifiers requested :: Cannot issue for "test.example.invalid": 
Domain name does not end with a valid public suffix (TLD)
```

**Solution**: Only add domains with valid public TLDs (.com, .org, .net, etc.)

### Status not updating in UI

**Solutions:**
- Click the "Refresh" button
- Check browser console for errors
- Verify API endpoint: `curl http://127.0.0.1:15872/api/certs`
- Hard refresh browser (Ctrl+Shift+R)

## Technical Implementation

### Backend Flow

1. **Domain Added** → Create placeholder CertAndKey with `status: "pending"`
2. **Store in Vault** → Domain visible immediately
3. **Trigger ACME** → Certificate issuance starts
4. **On Success** → Update with full cert data, `status: "issued"`
5. **On Error** → Update `status: "failed"`, populate `error` field

### Data Storage

All status is stored in Vault at:
```
{kv2_mount}/data/{kv2_secret_path}/domain/{domain}
```

Each certificate contains:
```json
{
  "common_name": "...",
  "status": "pending|issued|failed",
  "error": "error message if failed",
  "managed": true,
  ...other fields...
}
```

## Future Enhancements

Potential improvements:
- [ ] Retry button for failed certificates
- [ ] Progress percentage indicator
- [ ] Timestamp for when status last changed
- [ ] Notification system for status changes
- [ ] History of previous issuance attempts
- [ ] Auto-refresh when pending certificates exist

