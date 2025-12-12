# Changelog

## [Unreleased] - 2025-12-12

### Added
- **Command-line Flag**: Added `-config` flag to specify custom configuration file path
  - Precedence: flag > CONFIG_FILE env var > default (./config.yml)
  
- **Vault TLS Skip Verify**: Added configuration option to skip TLS certificate verification
  - New config option: `vault.tls_skip_verify` (default: false)
  - Useful for development/testing with self-signed certificates
  - Warning logged when enabled to discourage production use
  
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
- Updated `httpserver.go` to serve web UI on root path (`/`)
- Enhanced `certInfo` struct to include all certificate metadata
- Improved certificate retrieval logic with better error handling
- Updated configuration file with example domains section

### Fixed
- Fixed Vault token configuration bug (was setting literal string instead of value)
- Fixed error handling in `encodeForVault` function to check Marshal result

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
http://127.0.0.1:15872/
```

Or view the demo page at:
```
http://127.0.0.1:15872/static/demo.html
```

## API Endpoints

- `GET /` - Web UI dashboard
- `GET /certs` - JSON API returning all certificates
- `GET /healthz` - Health check
- `GET /version` - Version information
- `GET /metrics` - Server metrics
- `GET /static/*` - Static assets (CSS, JS, etc.)

