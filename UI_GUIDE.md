# Web UI User Guide

## Accessing the Dashboard

Open your web browser and navigate to:
```
http://127.0.0.1:15872/ui
```

Or simply visit `http://127.0.0.1:15872/` and you'll be automatically redirected.

## Dashboard Overview

### Statistics Cards (Top)
- **Total Certificates**: Count of all certificates
- **Valid**: Unexpired, managed certificates
- **Expiring Soon**: Certificates expiring within 30 days
- **Expired**: Expired certificates

### Action Bar
- **Refresh Button**: Manually refresh certificate data
- **Manage Domains Button** (Green): Open domain management modal
- **View Config Button** (Gray): View current application configuration
- **Auto-refresh Indicator**: Shown when pending certificates exist
- **Last Updated**: Timestamp of last data fetch

### Certificate Cards

Each certificate displays:
- **Certificate Name** (Common Name)
- **Management Badge**: "Managed" (green) or "Unmanaged" (gray)
- **Status Badge**: 
  - "Pending" (orange, pulsing) - Being issued
  - "Valid" (green) - Active and valid
  - "Expiring Soon" (orange) - < 30 days
  - "Expired" (red) - Past expiration
  - "Failed" (red) - Issuance error
- **Issuer Information**
- **Certificate Details** (dates, expiration countdown)
- **Subject Alternative Names (SANs)** - List of all domains covered
- **Action Buttons** (for issued certificates):
  - View Certificate
  - View Full Chain

## Viewing Configuration

### Opening Configuration Viewer

Click the **"View Config"** button in the action bar.

**What's displayed:**
- **Log Level** - Current logging verbosity
- **Server Settings** - Bind address and port
- **ACME Configuration** - Directory URL, email, DNS settings
- **Vault Configuration** - Address, mount path, settings

**Security:**
- Sensitive values (tokens, passwords) are **never displayed**
- Only non-sensitive configuration shown
- Safe to screenshot or share

**Features:**
- **Copy Buttons** - Each value has a copy button for easy copying
- **Hover to Expand** - Long values expand on hover to show full text
- **Tooltips** - Hover over values to see full content

**Use Cases:**
- Verify configuration without file access
- Troubleshooting connectivity issues
- Confirm which ACME directory is being used (staging vs production)
- Check Vault connection settings
- Copy values for documentation or support tickets

## Managing Domains

### Opening Domain Manager

Click the green **"Manage Domains"** button in the action bar.

The domain list will show status indicators:
- **🔴 Failed** - Certificate issuance failed
- **🟠 Expired** - Certificate has expired
- **🟠 Expiring Soon** - Certificate expiring within 30 days

### Adding a Domain

1. **Domain (Common Name)** field:
   - Enter the primary domain
   - Example: `example.com` or `*.example.com`
   - Wildcards supported for DNS-01 challenge

2. **Subject Alternative Names (SANs)** field (optional):
   - Add additional domains for the same certificate
   - One per line or comma-separated
   - Example:
     ```
     www.example.com
     api.example.com
     app.example.com
     ```
   - Or: `www.example.com, api.example.com, app.example.com`

3. **Click "Add Domain"**
   - Domain appears immediately with "Pending" status
   - Certificate issuance triggered automatically
   - Success message displayed
   - Auto-refresh starts (updates every 10 seconds)

### Removing a Domain

Each domain in the list has **two removal options**:

#### 🟠 Unmanage Button (Soft Delete)
- **What it does**:
  - Removes from managed domains list
  - Marks certificate as "unmanaged"
  - Certificate stays in Vault for reference
  - Certificate won't be renewed
  - Shows "Unmanaged" badge in dashboard

- **When to use**:
  - Temporarily stop managing
  - Keep certificate for historical reference
  - May want to re-enable later

- **Can be reversed**: Yes, re-add the domain to manage again

#### 🔴 Delete Button (Hard Delete)
- **What it does**:
  - Removes from managed domains list
  - **Permanently deletes** certificate from Vault
  - Certificate no longer visible in dashboard
  - All data lost forever

- **When to use**:
  - Permanently removing domain
  - Cleaning up invalid/failed domains
  - Domain no longer owned/needed
  - Freeing Vault storage

- **Can be reversed**: No, deletion is permanent

- **Warning displayed**:
  ```
  ⚠️ PERMANENT DELETION ⚠️
  
  This will:
  • Remove domain from managed list
  • DELETE the certificate from Vault permanently
  • Remove all certificate data (cannot be recovered)
  
  This action CANNOT be undone!
  ```

## Certificate States

### Pending 🟡
- **Just added** to managed list
- Certificate issuance in progress
- Shows animated hourglass icon
- Auto-refresh enabled
- Wait 30-90 seconds for completion

### Issued - Valid ✅
- Certificate successfully obtained
- Currently valid (not expiring soon)
- Full details available
- Can view/copy certificate

### Issued - Expiring Soon ⚠️
- Less than 30 days until expiration
- Will be auto-renewed soon
- Orange warning badge

### Issued - Expired ❌
- Certificate has expired
- Should be renewed
- Red expired badge

### Failed ❌
- Certificate issuance failed
- Error message displayed
- Review error and fix issue
- Remove and re-add to retry

### Unmanaged 📦
- Previously managed certificate
- No longer being renewed
- Kept for reference
- Can be re-enabled by adding domain again

## Viewing Certificates

### View Certificate Button
- Opens modal with PEM-encoded **leaf certificate**
- Copy to clipboard button included
- Use for:
  - Installing on servers
  - Verification
  - Backup

### View Full Chain Button
- Opens modal with **complete certificate chain**
- Includes leaf cert + intermediate CA certs
- Copy to clipboard button included
- Use for:
  - Complete chain installation
  - Troubleshooting trust issues
  - Full chain verification

## Tips and Best Practices

### 1. Start with Staging
For testing, use Let's Encrypt staging environment in `config.yml`:
```yaml
acme:
  directory: "https://acme-staging-v02.api.letsencrypt.org/directory"
```

### 2. Monitor Pending Certificates
- Auto-refresh runs while certificates are pending
- Check logs if pending > 5 minutes
- Failed certificates show error messages

### 3. Use SANs Wisely
- Group related domains in one certificate
- Example: `example.com` + SANs: `www.example.com`, `api.example.com`
- Saves on Let's Encrypt rate limits
- One renewal covers all domains

### 4. Regular Monitoring
- Check dashboard for expiring certificates
- Review failed issuances
- Remove unneeded domains

### 5. Soft Delete First
- Use "Unmanage" before "Delete"
- Keeps certificate for reference
- Can always re-enable if needed
- Only hard delete when certain

### 6. Wildcard Certificates
For wildcard domains:
```
Domain: *.example.com
SANs: example.com  (to also cover the root domain)
```

## Keyboard Shortcuts

- **R** - Refresh certificates (when focused on page)
- **M** - Open Manage Domains modal (when focused on page)
- **Esc** - Close modal

## Mobile Usage

The UI is fully responsive:
- Touch-friendly buttons
- Optimized layouts for small screens
- All features available on mobile
- Swipe-friendly scrolling

## Common Workflows

### Adding Multiple Domains

**One certificate per domain:**
1. Add `example.com` (no SANs)
2. Add `api.example.com` (no SANs)
3. Add `app.example.com` (no SANs)
Result: 3 separate certificates

**One certificate for all:**
1. Add `example.com` with SANs: `api.example.com, app.example.com`
Result: 1 certificate covering all 3 domains

### Cleaning Up Failed Domains

1. Identify failed certificates (red "Failed" badge)
2. Read error message
3. Click "Manage Domains"
4. Find domain in list
5. Click "Delete" (hard delete)
6. Fix the underlying issue (DNS, config, etc.)
7. Re-add with correct configuration

### Temporarily Disabling Management

1. Click "Manage Domains"
2. Find domain
3. Click "Unmanage"
4. Certificate stops renewing but remains visible
5. Later: Re-add same domain to resume management

## Troubleshooting

### Domain doesn't appear after adding
- Check if error message displayed
- Verify browser console for errors
- Check API: `curl http://127.0.0.1:15872/api/domains`

### Certificate stuck in "Pending"
- Wait 5 minutes (ACME can be slow)
- Check application logs
- If >10 minutes, remove and re-add
- Verify DNS and API credentials

### Can't remove domain
- Check browser console for errors
- Try hard refresh (Ctrl+Shift+R)
- Use API directly if UI fails
- Check Vault connectivity

### Auto-refresh not working
- Only works when pending certificates exist
- Check browser console
- Manually click Refresh button

## Security Notes

### Current Setup
- No authentication by default
- Suitable for internal/trusted networks only

### Production Recommendations
1. **Add authentication** (nginx basic auth, OAuth, etc.)
2. **Use HTTPS** with reverse proxy
3. **Firewall rules** to restrict access
4. **VPN access** for remote management
5. **Audit logging** enabled in Vault

## Need Help?

- Check application logs for detailed information
- Review error messages in UI
- Consult documentation:
  - `README.md` - Overview
  - `QUICKSTART.md` - Setup guide
  - `DOMAIN_MANAGEMENT.md` - Domain API details
  - `STATUS_TRACKING.md` - Status system explained

