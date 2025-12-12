# Web UI Features

## Overview

The ACME Certificate Store now includes a modern, responsive web interface for monitoring and managing SSL/TLS certificates stored in HashiCorp Vault.

## Screenshots & Preview

### Dashboard View
The main dashboard provides an at-a-glance view of all your certificates with:
- **Statistics Cards**: Total certificates, valid, expiring soon, and expired counts
- **Certificate Cards**: Detailed information for each certificate
- **Real-time Updates**: Refresh button to fetch latest data
- **Responsive Design**: Works on desktop, tablet, and mobile devices

### Color Scheme
- **Dark Theme**: Modern dark background with gradient accents
- **Color-Coded Status**:
  - 🟢 Green: Valid certificates (>30 days remaining)
  - 🟠 Orange: Expiring soon (≤30 days remaining)
  - 🔴 Red: Expired certificates

## Key Features

### 1. Certificate Dashboard
- View all managed certificates in a clean, organized layout
- Each certificate card displays:
  - Common Name (domain)
  - Management status ("Managed" or "Unmanaged" badge)
  - Issuance status ("Pending", "Issued", or "Failed")
  - Issuer information
  - Issue date and expiration date
  - Days until expiration
  - Certificate status badge (Valid/Expiring/Expired)
  - Subject Alternative Names (SANs)
  - Error messages for failed issuance

**Certificate States:**
- **Pending**: Certificate issuance in progress (animated indicator)
- **Issued**: Certificate successfully issued and valid
- **Failed**: Certificate issuance failed (shows error message)

### 2. Statistics Overview
Real-time statistics showing:
- Total number of certificates
- Number of valid certificates
- Number of certificates expiring within 30 days
- Number of expired certificates

### 3. Certificate Viewer
- **View Certificate**: Display the PEM-encoded leaf certificate in a modal
- **View Full Chain**: Display the complete certificate chain
- **Copy to Clipboard**: One-click copy functionality for certificates

### 4. Configuration Viewer (NEW)
- **View Config Button**: Opens configuration viewer modal
- **Non-Sensitive Display**: Only shows safe configuration values
- **Organized Sections**: 
  - Logging configuration
  - Server settings
  - ACME provider settings
  - Vault connection details
- **Security Note**: Tokens and secrets are never displayed
- **Color Coding**: Warning color for insecure settings (e.g., TLS skip verify)
- **Use Cases**: 
  - Verify settings without file access
  - Troubleshoot configuration issues
  - Confirm ACME directory (staging vs production)

### 5. Domain Management
- **Manage Domains Button**: Opens domain management modal
- **View Managed Domains**: List of all domains being managed
- **Add Domain**: Form to add new domains with SANs and validation
- **Remove Options**:
  - "Unmanage" button (orange) - Soft delete, keeps certificate
  - "Delete" button (red) - Hard delete, removes from Vault permanently
- **Confirmation Dialogs**: Different warnings for soft vs hard delete
- **Real-time Updates**: Domain list updates immediately after changes
- **Domain Count**: Shows total number of managed domains

### 5. Status Indicators
Visual badges indicating certificate health:
- **VALID**: Certificate is valid and not expiring soon
- **EXPIRING SOON**: Certificate expires within 30 days
- **EXPIRED**: Certificate has already expired

### 6. Responsive Design
- Mobile-friendly interface
- Adapts to different screen sizes
- Touch-friendly buttons and controls
- Optimized for both desktop and mobile browsers

### 7. User Experience
- **Smooth Animations**: Transitions and hover effects
- **Loading States**: Spinner during data fetch
- **Error Handling**: Clear error messages if API fails
- **Last Updated**: Timestamp showing when data was last refreshed
- **Intuitive Navigation**: Easy-to-use interface requiring no training

## Technical Implementation

### Frontend Stack
- **Pure JavaScript**: No frameworks required, lightweight and fast
- **Modern CSS3**: Grid, Flexbox, animations, and transitions
- **HTML5**: Semantic markup for accessibility

### API Integration
- RESTful API endpoint: `GET /certs`
- JSON response format
- Automatic error handling
- Real-time data fetching

### Performance
- **Compression**: Gzip/Brotli compression for faster loading
- **Caching**: ETag support for efficient resource loading
- **Optimized Assets**: Minified and optimized CSS/JS
- **Fast Rendering**: Efficient DOM manipulation

## Accessibility Features
- Semantic HTML structure
- Clear visual hierarchy
- Readable font sizes
- High contrast color scheme
- Keyboard navigation support

## Browser Compatibility
Tested and working on:
- Chrome/Chromium (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Security Considerations

### Current Implementation
The web UI is designed for internal/trusted network use. For production deployment, consider:

1. **Authentication**: Add authentication middleware
2. **HTTPS**: Use a reverse proxy (nginx, Caddy) with TLS
3. **Access Control**: Implement IP whitelisting or VPN access
4. **Rate Limiting**: Prevent abuse of API endpoints
5. **CORS**: Configure appropriate CORS policies

### Recommended Production Setup

```nginx
# Example nginx reverse proxy configuration
server {
    listen 443 ssl http2;
    server_name acme-manager.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Basic authentication
    auth_basic "ACME Certificate Manager";
    auth_basic_user_file /etc/nginx/.htpasswd;
    
    location / {
        proxy_pass http://127.0.0.1:15872;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Future Enhancements

Potential features for future versions:
- [ ] Certificate renewal trigger from UI
- [ ] Certificate download functionality
- [ ] Search and filter capabilities
- [ ] Sorting options (by expiry, name, status)
- [ ] Email notifications for expiring certificates
- [ ] Certificate history and audit log
- [ ] Multi-user support with role-based access
- [ ] Dark/light theme toggle
- [ ] Export certificates in various formats
- [ ] Certificate comparison tool
- [ ] Integration with monitoring systems (Prometheus, Grafana)

## Demo Mode

A demo page is available at `/static/demo.html` showing the UI with sample data. This is useful for:
- Previewing the interface without running Vault
- Testing UI changes
- Demonstrating features to stakeholders
- Development and design work

## Customization

### Changing Colors
Edit `web/static/style.css` and modify the CSS variables:

```css
:root {
    --primary-color: #6366f1;      /* Primary accent color */
    --secondary-color: #8b5cf6;    /* Secondary accent color */
    --success-color: #10b981;      /* Valid status color */
    --warning-color: #f59e0b;      /* Warning status color */
    --danger-color: #ef4444;       /* Error/expired color */
    --bg-color: #0f172a;           /* Background color */
    --surface-color: #1e293b;      /* Card background */
    --text-color: #f1f5f9;         /* Primary text */
    --text-secondary: #94a3b8;     /* Secondary text */
    --border-color: #334155;       /* Border color */
}
```

### Modifying Layout
The UI uses CSS Grid for the main layout. Adjust grid templates in `style.css`:

```css
.stats-container {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;
}
```

### Adding Custom Features
The JavaScript code in `web/static/app.js` is well-commented and modular. Key functions:
- `loadCertificates()`: Fetches data from API
- `renderCertificates()`: Renders certificate cards
- `updateStats()`: Updates statistics
- `viewCertificate()`: Opens certificate viewer modal

## Support

For issues, questions, or feature requests related to the web UI:
1. Check the browser console for JavaScript errors
2. Verify the `/certs` API endpoint returns valid JSON
3. Ensure static files are being served correctly
4. Check application logs for server-side errors

## Conclusion

The web UI transforms the ACME Certificate Store from a backend service into a complete certificate management solution with an intuitive, modern interface. It provides visibility into certificate health and makes certificate management accessible to both technical and non-technical users.

