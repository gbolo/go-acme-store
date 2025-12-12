# Changelog

## [Unreleased] - 2025-12-12

### Added
- **Command-line Flag**: Added `-config` flag to specify custom configuration file path
  - Precedence: flag > CONFIG_FILE env var > default (./config.yml)
  
- **Vault TLS Skip Verify**: Added configuration option to skip TLS certificate verification
  - New config option: `vault.tls_skip_verify` (default: false)
  - Useful for development/testing with self-signed certificates
  - Warning logged when enabled to discourage production use

- **Route Organization**: Reorganized routes for better API structure
  - API endpoints moved to `/api` prefix
  - Web UI moved to `/ui` prefix
  - Root `/` redirects to `/ui`

- **Vault-based Domain Management**: Domains now managed via Vault instead of config file
  - Added `GET /api/domains` - List managed domains
  - Added `POST /api/domains` - Add a domain with optional SANs
  - Added `DELETE /api/domains/{domain}` - Remove domain (soft delete)
  - Added `DELETE /api/domains/{domain}?delete_cert=true` - Remove and delete certificate
  - Added `POST /api/trigger-renewal` - Manually trigger certificate issuance/renewal
  - Domains stored in Vault for centralized state management
  - Dynamic addition/removal without application restart
  - Web UI includes "Manage Domains" modal for easy domain management
  - Real-time domain list updates in UI
  - Automatic certificate issuance when adding domains via Web UI
  - **Subject Alternative Names (SANs) Support**: 
    - UI allows entering multiple SANs when adding domains
    - SANs can be entered one per line or comma-separated
    - API accepts `sans` array in domain add request
    - ACME now requests certificates for all domains (CN + SANs)
    - SANs stored in Vault with domain configuration
  
- **Certificate Management States**:
  - Added `managed` boolean field to certificates
  - Soft delete: marks certificate as unmanaged (default)
  - Hard delete: completely removes certificate from Vault
  - Unmanaged certificates NOT renewed automatically
  - UI shows "Managed" or "Unmanaged" badge for each certificate
  - ACME worker skips renewal for unmanaged certificates
  - Re-adding a previously removed domain automatically flips `managed` back to `true`

- **Flaticon Icons**: Replaced emoji icons with professional Flaticon uicons
  - Diploma icon for certificate viewing
  - Link icon for chain viewing
  - Copy icon for clipboard operations
  - Favicon added using shield logo
  
- **Web UI Dashboard**: Beautiful, modern web interface for certificate monitoring
  - Real-time certificate status display
  - Statistics dashboard showing total, valid, expiring soon, and expired certificates
  - Responsive design that works on desktop and mobile devices
  - Dark theme with gradient accents
  
- **Enhanced Certificate Display**:
  - Certificate cards with detailed information
  - Common Name and Subject Alternative Names (SANs)
  - Issuer information
  - Issue and expiration dates
  - Days until expiration with color-coded indicators
  - Status badges (Valid, Expiring Soon, Expired)
  
- **Certificate Viewer**:
  - Modal dialogs to view PEM-encoded certificates
  - View individual leaf certificates
  - View complete certificate chains
  - Copy to clipboard functionality
  
- **API Enhancements**:
  - Enhanced `/certs` endpoint to return full certificate details
  - Added comprehensive certificate metadata in JSON responses
  - Improved error handling and logging
  
- **Static File Serving**:
  - Integrated static file server for web assets
  - Optimized delivery with compression and caching
  - ETag support for efficient resource loading
  
- **Documentation**:
  - Comprehensive README with feature descriptions
  - Quick Start Guide for easy setup
  - Demo HTML page to preview the UI
  - API endpoint documentation

### Changed
- Updated `httpserver.go` to serve web UI on `/ui` path with redirect from `/`
- Enhanced `certInfo` struct to include all certificate metadata
- Improved certificate retrieval logic with better error handling
- Removed `domains` section from configuration file (now managed via Vault/API)
- Reduced header banner size by ~20% for more compact design
- API calls now use `/api/certs` endpoint instead of `/certs`
- Worker daemon now loads domains from Vault instead of config file
- Worker daemon refactored to use ticker + trigger channel for on-demand execution
- Keystore interface extended with domain management methods

### Fixed
- Fixed Vault token configuration bug (was setting literal string instead of value)
- Fixed error handling in `encodeForVault` function to check Marshal result
- Fixed 500 error when no certificates exist in Vault (nil pointer dereference)
- Fixed certificate API to return empty array instead of null when no certificates
- Fixed logging bug in `StoreCertAndKey` (was logging on error instead of success)
- Fixed nil pointer dereference in `GetAllDomains()` when Vault domain path is empty
- Fixed `readDataIntoInterface()` to return nil error (not undefined) when no data exists
- Fixed re-adding previously removed domains - now automatically marks certificate as managed
- Fixed ACME function not using SANs parameter - now properly requests multi-domain certificates

### Technical Details
- **Frontend Stack**: Vanilla JavaScript, CSS3, HTML5 (no frameworks required)
- **UI Features**:
  - CSS Grid and Flexbox for responsive layouts
  - CSS animations and transitions
  - Modal dialogs for certificate viewing
  - Automatic refresh functionality
  - Last updated timestamp
  
- **Color Coding**:
  - Green: Valid certificates (>30 days until expiration)
  - Orange: Expiring soon (≤30 days until expiration)
  - Red: Expired certificates (<0 days)

### Files Added
- `web/static/index.html` - Main web UI page
- `web/static/style.css` - Stylesheet with modern dark theme
- `web/static/app.js` - JavaScript application logic
- `web/static/demo.html` - Demo page with sample data
- `README.md` - Project documentation
- `QUICKSTART.md` - Quick start guide
- `CHANGELOG.md` - This changelog

### Files Modified
- `cmd/acme-store/httpserver.go` - Added web UI routes and enhanced API
- `config.yml` - Added domains configuration section

## Usage

After building and running the application, access the web UI at:
```
http://127.0.0.1:15872/ui
```

Or just visit the root URL and be redirected:
```
http://127.0.0.1:15872/
```

## API Endpoints

### Web UI
- `GET /` - Redirects to `/ui`
- `GET /ui` - Web UI dashboard
- `GET /ui/static/*` - Static assets (CSS, JS, images)

### API - Certificates
- `GET /api/certs` - JSON API returning all certificates

### API - Domain Management
- `GET /api/domains` - List all managed domains
- `POST /api/domains` - Add a domain (body: `{"domain": "example.com"}`)
- `DELETE /api/domains/{domain}` - Remove a domain

### API - System
- `GET /api/healthz` - Health check
- `GET /api/version` - Version information

### Monitoring
- `GET /metrics` - Server metrics

