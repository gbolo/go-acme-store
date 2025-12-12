#!/bin/bash
# Example script: Fetch certificates and deploy to nginx
# This script can be run via cron or systemd timer

set -e

# Configuration
CONFIG_FILE="/etc/acme-store/config.yml"
CERT_DIR="/etc/ssl/acme-certs"
FETCHER_BIN="/usr/local/bin/acme-store-fetcher"
NGINX_BIN="/usr/bin/nginx"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    log_error "This script must be run as root"
    exit 1
fi

# Check if fetcher binary exists
if [ ! -f "$FETCHER_BIN" ]; then
    log_error "Fetcher binary not found: $FETCHER_BIN"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
fi

# Create cert directory if it doesn't exist
if [ ! -d "$CERT_DIR" ]; then
    log_info "Creating certificate directory: $CERT_DIR"
    mkdir -p "$CERT_DIR"
    chmod 755 "$CERT_DIR"
fi

# Fetch certificates
log_info "Fetching certificates from Vault..."
if $FETCHER_BIN -config "$CONFIG_FILE" -output-dir "$CERT_DIR"; then
    log_info "Certificates fetched successfully"
else
    log_error "Failed to fetch certificates"
    exit 1
fi

# Test nginx configuration
log_info "Testing nginx configuration..."
if $NGINX_BIN -t; then
    log_info "Nginx configuration is valid"
else
    log_error "Nginx configuration test failed"
    exit 1
fi

# Reload nginx
log_info "Reloading nginx..."
if systemctl reload nginx; then
    log_info "Nginx reloaded successfully"
else
    log_error "Failed to reload nginx"
    exit 1
fi

log_info "Certificate deployment complete!"

