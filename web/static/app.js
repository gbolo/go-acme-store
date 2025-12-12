// Global state
let certificates = [];
let autoRefreshInterval = null;

// Load certificates on page load
document.addEventListener('DOMContentLoaded', function() {
    loadCertificates();
});

// Configuration modal functions
async function showConfig() {
    const modal = document.getElementById('configModal');
    const loadingEl = document.getElementById('configLoading');
    const contentEl = document.getElementById('configContent');
    
    modal.style.display = 'block';
    loadingEl.style.display = 'flex';
    contentEl.style.display = 'none';
    
    try {
        const response = await fetch('/api/config');
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const config = await response.json();
        
        // Helper function to set config value with copy button
        function setConfigValue(elementId, value, colorStyle) {
            const valueEl = document.getElementById(elementId);
            const displayValue = value || 'N/A';
            const containerEl = valueEl.closest('.config-value-container');
            
            // Update the value text
            valueEl.textContent = displayValue;
            valueEl.title = displayValue;
            
            if (colorStyle) {
                valueEl.style.color = colorStyle;
            }
            
            // Add copy button if not already present
            if (!containerEl.querySelector('.config-copy-btn')) {
                const copyBtn = document.createElement('button');
                copyBtn.className = 'config-copy-btn';
                copyBtn.innerHTML = '<i class="fi fi-rr-copy-alt"></i>';
                copyBtn.onclick = function() { copyConfigValue(displayValue, this); };
                containerEl.appendChild(copyBtn);
            } else {
                // Update existing button's onclick
                const copyBtn = containerEl.querySelector('.config-copy-btn');
                copyBtn.onclick = function() { copyConfigValue(displayValue, this); };
            }
        }
        
        // Populate config values
        setConfigValue('config-log-level', config.log.level);
        setConfigValue('config-server-bind-address', config.server.bind_address);
        setConfigValue('config-server-bind-port', config.server.bind_port);
        setConfigValue('config-acme-directory', config.acme.directory);
        setConfigValue('config-acme-email', config.acme.account_email);
        setConfigValue('config-acme-dns-server', config.acme.dns_server);
        setConfigValue('config-acme-dns-provider', config.acme.dns_provider);
        setConfigValue('config-vault-address', config.vault.address);
        setConfigValue('config-vault-mount', config.vault.kv2_mount);
        setConfigValue('config-vault-path', config.vault.kv2_secret_path);
        
        // Special handling for TLS skip verify (with color coding)
        const tlsSkip = config.vault.tls_skip_verify;
        const tlsSkipValue = tlsSkip ? 'Yes' : 'No';
        const tlsColor = tlsSkip ? 'var(--warning-color)' : 'var(--success-color)';
        setConfigValue('config-vault-tls-skip', tlsSkipValue, tlsColor);
        
        loadingEl.style.display = 'none';
        contentEl.style.display = 'block';
        
    } catch (error) {
        console.error('Error loading config:', error);
        loadingEl.innerHTML = `<div class="error-text">Failed to load configuration: ${error.message}</div>`;
    }
}

function closeConfigModal() {
    const modal = document.getElementById('configModal');
    modal.style.display = 'none';
}

// Copy config value to clipboard
function copyConfigValue(value, button) {
    navigator.clipboard.writeText(value).then(() => {
        const originalHTML = button.innerHTML;
        button.innerHTML = '<i class="fi fi-rr-check"></i>';
        button.classList.add('copied');
        
        setTimeout(() => {
            button.innerHTML = originalHTML;
            button.classList.remove('copied');
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy:', err);
        alert('Failed to copy to clipboard');
    });
}

// Close config modal when clicking outside
window.addEventListener('click', function(event) {
    const modal = document.getElementById('configModal');
    if (event.target === modal) {
        closeConfigModal();
    }
});

// Load certificates from API
async function loadCertificates() {
    const loadingEl = document.getElementById('loading');
    const errorEl = document.getElementById('error');
    const certsContainer = document.getElementById('certificates');
    const refreshBtn = document.getElementById('refreshBtn');

    // Show loading state
    loadingEl.style.display = 'block';
    errorEl.style.display = 'none';
    certsContainer.innerHTML = '';
    refreshBtn.classList.add('loading');

    try {
        const response = await fetch('/api/certs');
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        certificates = await response.json();
        
        // Hide loading
        loadingEl.style.display = 'none';
        refreshBtn.classList.remove('loading');

        // Update stats
        updateStats();

        // Render certificates
        renderCertificates();

        // Update last updated time
        updateLastUpdatedTime();
        
        // Setup auto-refresh if there are pending certificates
        setupAutoRefresh();

    } catch (error) {
        console.error('Error loading certificates:', error);
        loadingEl.style.display = 'none';
        refreshBtn.classList.remove('loading');
        errorEl.textContent = `Failed to load certificates: ${error.message}`;
        errorEl.style.display = 'block';
    }
}

// Update statistics
function updateStats() {
    const now = new Date();
    let validCount = 0;
    let expiringSoonCount = 0;
    let expiredCount = 0;

    certificates.forEach(cert => {
        const expiryDate = new Date(cert.expires_on);
        const daysUntilExpiry = Math.floor((expiryDate - now) / (1000 * 60 * 60 * 24));

        if (daysUntilExpiry < 0) {
            expiredCount++;
        } else if (daysUntilExpiry <= 30) {
            expiringSoonCount++;
        } else {
            validCount++;
        }
    });

    document.getElementById('totalCerts').textContent = certificates.length;
    document.getElementById('validCerts').textContent = validCount;
    document.getElementById('expiringSoon').textContent = expiringSoonCount;
    document.getElementById('expiredCerts').textContent = expiredCount;
}

// Render certificates
function renderCertificates() {
    const container = document.getElementById('certificates');
    container.innerHTML = '';

    if (certificates.length === 0) {
        container.innerHTML = '<div style="text-align: center; padding: 40px; color: var(--text-secondary);">No certificates found</div>';
        return;
    }

    certificates.forEach((cert, index) => {
        const certCard = createCertificateCard(cert, index);
        container.appendChild(certCard);
    });
}

// Create certificate card
function createCertificateCard(cert, index) {
    const card = document.createElement('div');
    card.className = 'cert-card';

    // Check certificate status
    const certStatus = cert.status || 'issued';
    let statusBadge = '';
    let detailsSection = '';
    
    if (certStatus === 'pending') {
        statusBadge = '<div class="cert-status status-pending">Pending</div>';
        detailsSection = `
            <div class="cert-details">
                <div class="cert-detail pending-notice">
                    <i class="fi fi-rr-hourglass"></i>
                    <span>Certificate issuance in progress... This may take a few minutes.</span>
                </div>
            </div>
        `;
    } else if (certStatus === 'failed') {
        statusBadge = '<div class="cert-status status-failed">Failed</div>';
        detailsSection = `
            <div class="cert-details">
                <div class="cert-detail error-notice">
                    <i class="fi fi-rr-cross-circle"></i>
                    <div>
                        <strong>Error:</strong>
                        <p>${escapeHtml(cert.error || 'Unknown error')}</p>
                    </div>
                </div>
            </div>
        `;
    } else {
        const status = getCertificateStatus(cert.expires_on);
        const expiryDate = new Date(cert.expires_on);
        const issuedDate = new Date(cert.issued_on);
        const daysUntilExpiry = Math.floor((expiryDate - new Date()) / (1000 * 60 * 60 * 24));
        
        statusBadge = `<div class="cert-status status-${status.class}">${status.text}</div>`;
        detailsSection = `
            <div class="cert-details">
                <div class="cert-detail">
                    <div class="detail-label">Issued On</div>
                    <div class="detail-value">${formatDate(issuedDate)}</div>
                </div>
                <div class="cert-detail">
                    <div class="detail-label">Expires On</div>
                    <div class="detail-value">${formatDate(expiryDate)}</div>
                </div>
                <div class="cert-detail">
                    <div class="detail-label">Days Until Expiry</div>
                    <div class="detail-value" style="color: ${daysUntilExpiry < 0 ? 'var(--danger-color)' : daysUntilExpiry <= 30 ? 'var(--warning-color)' : 'var(--success-color)'}">
                        ${daysUntilExpiry} days
                    </div>
                </div>
                <div class="cert-detail">
                    <div class="detail-label">Certificate Chain</div>
                    <div class="detail-value">${cert.issuers ? cert.issuers.length : 0} issuer(s)</div>
                </div>
            </div>
        `;
    }

    card.innerHTML = `
        <div class="cert-header">
            <div class="cert-title">
                <div class="cert-name-row">
                    <div class="cert-name">${escapeHtml(cert.common_name)}</div>
                    ${cert.managed ? '<span class="managed-badge">Managed</span>' : '<span class="unmanaged-badge">Unmanaged</span>'}
                </div>
                <div class="cert-issuer">Issuer: ${cert.issuers && cert.issuers.length > 0 ? escapeHtml(cert.issuers[0]) : (certStatus === 'issued' ? 'Unknown' : 'Pending issuance')}</div>
            </div>
            ${statusBadge}
        </div>

        ${detailsSection}

        ${cert.sans && cert.sans.length > 0 ? `
            <div class="cert-detail">
                <div class="detail-label">Subject Alternative Names (SANs)</div>
                <ul class="sans-list">
                    ${cert.sans.map(san => `<li>${escapeHtml(san)}</li>`).join('')}
                </ul>
            </div>
        ` : ''}

        ${certStatus === 'issued' ? `
            <div class="cert-actions">
                <button class="btn btn-primary" onclick="viewCertificate(${index})">
                    <i class="fi fi-rr-diploma"></i>
                    <span>View Certificate</span>
                </button>
                <button class="btn btn-secondary" onclick="viewCertificateChain(${index})">
                    <i class="fi fi-rr-link-alt"></i>
                    <span>View Full Chain</span>
                </button>
            </div>
        ` : ''}
    `;

    return card;
}

// Get certificate status
function getCertificateStatus(expiresOn) {
    const expiryDate = new Date(expiresOn);
    const now = new Date();
    const daysUntilExpiry = Math.floor((expiryDate - now) / (1000 * 60 * 60 * 24));

    if (daysUntilExpiry < 0) {
        return { class: 'expired', text: 'Expired' };
    } else if (daysUntilExpiry <= 30) {
        return { class: 'warning', text: 'Expiring Soon' };
    } else {
        return { class: 'valid', text: 'Valid' };
    }
}

// Format date
function formatDate(date) {
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Update last updated time
function updateLastUpdatedTime() {
    const now = new Date();
    document.getElementById('lastUpdated').textContent = `Last updated: ${formatDate(now)}`;
}

// View certificate
function viewCertificate(index) {
    const cert = certificates[index];
    const modal = document.getElementById('certModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalBody = document.getElementById('modalBody');

    modalTitle.textContent = `Certificate: ${cert.common_name}`;
    modalBody.innerHTML = `
        <div class="pem-content">${escapeHtml(cert.leaf_cert_pem)}</div>
        <button class="btn btn-primary copy-btn" onclick="copyToClipboard('${escapeForJs(cert.leaf_cert_pem)}')">
            <i class="fi fi-rr-copy-alt"></i>
            <span>Copy to Clipboard</span>
        </button>
    `;

    modal.style.display = 'block';
}

// View certificate chain
function viewCertificateChain(index) {
    const cert = certificates[index];
    const modal = document.getElementById('certModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalBody = document.getElementById('modalBody');

    modalTitle.textContent = `Certificate Chain: ${cert.common_name}`;
    modalBody.innerHTML = `
        <div class="pem-content">${escapeHtml(cert.cert_chain_pem)}</div>
        <button class="btn btn-primary copy-btn" onclick="copyToClipboard('${escapeForJs(cert.cert_chain_pem)}')">
            <i class="fi fi-rr-copy-alt"></i>
            <span>Copy to Clipboard</span>
        </button>
    `;

    modal.style.display = 'block';
}

// Close modal
function closeModal() {
    const modal = document.getElementById('certModal');
    modal.style.display = 'none';
}

// Close modal when clicking outside
window.onclick = function(event) {
    const certModal = document.getElementById('certModal');
    const domainModal = document.getElementById('domainModal');
    
    if (event.target == certModal) {
        certModal.style.display = 'none';
    }
    if (event.target == domainModal) {
        domainModal.style.display = 'none';
    }
}

// Copy to clipboard
function copyToClipboard(text) {
    // Decode the text (it was escaped for JS)
    const textarea = document.createElement('textarea');
    textarea.value = text;
    document.body.appendChild(textarea);
    textarea.select();
    
    try {
        document.execCommand('copy');
        alert('Certificate copied to clipboard!');
    } catch (err) {
        console.error('Failed to copy:', err);
        alert('Failed to copy certificate to clipboard');
    }
    
    document.body.removeChild(textarea);
}

// Escape for JavaScript
function escapeForJs(text) {
    return text.replace(/\\/g, '\\\\')
               .replace(/'/g, "\\'")
               .replace(/"/g, '\\"')
               .replace(/\n/g, '\\n')
               .replace(/\r/g, '\\r');
}

// Domain Management Functions

async function showDomainManager() {
    const modal = document.getElementById('domainModal');
    modal.style.display = 'block';
    
    // Clear previous message
    document.getElementById('domainAddMessage').style.display = 'none';
    document.getElementById('newDomainInput').value = '';
    
    // Ensure we have latest certificate data before loading domains
    await loadCertificates();
    // Load domains with status pills
    await loadDomains();
}

function closeDomainModal() {
    const modal = document.getElementById('domainModal');
    modal.style.display = 'none';
}

async function loadDomains() {
    const loadingEl = document.getElementById('domainLoading');
    const listEl = document.getElementById('domainList');
    const countEl = document.getElementById('domainCount');
    
    loadingEl.style.display = 'flex';
    listEl.innerHTML = '';
    
    try {
        const response = await fetch('/api/domains');
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        const domains = data.domains || [];
        
        loadingEl.style.display = 'none';
        countEl.textContent = domains.length;
        
        if (domains.length === 0) {
            listEl.innerHTML = '<div class="no-domains">No domains configured. Add your first domain above.</div>';
            return;
        }
        
        domains.forEach(domain => {
            // Find certificate status for this domain
            const cert = certificates.find(c => c.common_name === domain);
            let statusPill = '';
            
            if (cert) {
                if (cert.status === 'failed') {
                    statusPill = '<span class="domain-status-pill status-failed-pill">Failed</span>';
                } else if (cert.status === 'issued') {
                    // Check if expired or expiring
                    const expiryDate = new Date(cert.expires_on);
                    const daysUntilExpiry = Math.floor((expiryDate - new Date()) / (1000 * 60 * 60 * 24));
                    if (daysUntilExpiry < 0) {
                        statusPill = '<span class="domain-status-pill status-expired-pill">Expired</span>';
                    } else if (daysUntilExpiry <= 30) {
                        statusPill = '<span class="domain-status-pill status-expiring-pill">Expiring Soon</span>';
                    }
                }
            }
            
            const domainItem = document.createElement('div');
            domainItem.className = 'domain-item';
            domainItem.innerHTML = `
                <div class="domain-name">
                    <i class="fi fi-rr-globe"></i>
                    <span>${escapeHtml(domain)}</span>
                    ${statusPill}
                </div>
                <div class="domain-actions">
                    <button class="btn btn-warning btn-small" onclick="removeDomain('${escapeForJs(domain)}', false)">
                        <i class="fi fi-rr-minus-circle"></i>
                        <span>Unmanage</span>
                    </button>
                    <button class="btn btn-danger btn-small" onclick="removeDomain('${escapeForJs(domain)}', true)">
                        <i class="fi fi-rr-trash"></i>
                        <span>Delete</span>
                    </button>
                </div>
            `;
            listEl.appendChild(domainItem);
        });
        
    } catch (error) {
        console.error('Error loading domains:', error);
        loadingEl.style.display = 'none';
        listEl.innerHTML = `<div class="error-text">Failed to load domains: ${error.message}</div>`;
    }
}

async function addDomain() {
    const domainInput = document.getElementById('newDomainInput');
    const sansInput = document.getElementById('newSANsInput');
    const messageEl = document.getElementById('domainAddMessage');
    const domain = domainInput.value.trim();
    
    if (!domain) {
        showMessage(messageEl, 'Please enter a domain name', 'error');
        return;
    }
    
    // Basic validation
    if (!/^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(domain)) {
        showMessage(messageEl, 'Invalid domain format', 'error');
        return;
    }
    
    // Parse SANs (newline or comma-separated)
    const sansText = sansInput.value.trim();
    let sans = [];
    if (sansText) {
        sans = sansText.split(/[\n,]+/)
            .map(s => s.trim())
            .filter(s => s.length > 0);
        
        // Validate each SAN
        for (const san of sans) {
            if (!/^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(san)) {
                showMessage(messageEl, `Invalid SAN format: ${san}`, 'error');
                return;
            }
        }
    }
    
    try {
        const requestBody = { domain: domain };
        if (sans.length > 0) {
            requestBody.sans = sans;
        }
        
        const response = await fetch('/api/domains', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody)
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'Failed to add domain');
        }
        
        let message = `Domain ${domain} added successfully!`;
        if (sans.length > 0) {
            message += ` (with ${sans.length} SAN${sans.length > 1 ? 's' : ''})`;
        }
        message += ' Triggering certificate issuance...';
        
        showMessage(messageEl, message, 'success');
        domainInput.value = '';
        sansInput.value = '';
        
        // Reload domain list
        await loadDomains();
        
        // Trigger certificate renewal/issuance
        await triggerRenewal();
        
        // Reload certificates after a delay (give time for cert to be issued)
        setTimeout(() => {
            loadCertificates();
        }, 5000);
        
    } catch (error) {
        console.error('Error adding domain:', error);
        showMessage(messageEl, error.message, 'error');
    }
}

async function removeDomain(domain, deleteCert) {
    let confirmMessage;
    
    if (deleteCert) {
        confirmMessage = `⚠️ PERMANENT DELETION ⚠️\n\nAre you sure you want to COMPLETELY DELETE ${domain}?\n\n` +
                        `This will:\n` +
                        `• Remove domain from managed list\n` +
                        `• DELETE the certificate from Vault permanently\n` +
                        `• Remove all certificate data (cannot be recovered)\n\n` +
                        `This action CANNOT be undone!`;
    } else {
        confirmMessage = `Unmanage ${domain}?\n\n` +
                        `This will:\n` +
                        `• Remove domain from managed list\n` +
                        `• Mark certificate as "unmanaged"\n` +
                        `• Certificate will NOT be renewed\n` +
                        `• Certificate data remains in Vault for reference\n\n` +
                        `You can re-add this domain later.`;
    }
    
    if (!confirm(confirmMessage)) {
        return;
    }
    
    try {
        let url = `/api/domains/${encodeURIComponent(domain)}`;
        if (deleteCert) {
            url += '?delete_cert=true';
        }
        
        const response = await fetch(url, {
            method: 'DELETE'
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'Failed to remove domain');
        }
        
        // Reload domain list
        await loadDomains();
        
        // Reload certificates
        loadCertificates();
        
    } catch (error) {
        console.error('Error removing domain:', error);
        alert(`Failed to remove domain: ${error.message}`);
    }
}

function showMessage(element, message, type) {
    element.textContent = message;
    element.className = `message message-${type}`;
    element.style.display = 'block';
    
    // Auto-hide after 5 seconds
    setTimeout(() => {
        element.style.display = 'none';
    }, 5000);
}

async function triggerRenewal() {
    try {
        const response = await fetch('/api/trigger-renewal', {
            method: 'POST'
        });
        
        if (!response.ok) {
            console.warn('Failed to trigger renewal:', response.status);
        } else {
            console.log('Certificate renewal triggered successfully');
        }
    } catch (error) {
        console.error('Error triggering renewal:', error);
    }
}

function setupAutoRefresh() {
    const indicator = document.getElementById('autoRefreshIndicator');
    
    // Clear existing interval
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
    
    // Check if there are any pending certificates
    const hasPending = certificates.some(cert => cert.status === 'pending');
    
    if (hasPending) {
        console.log('Pending certificates detected, enabling auto-refresh every 10 seconds');
        if (indicator) {
            indicator.style.display = 'flex';
        }
        autoRefreshInterval = setInterval(() => {
            console.log('Auto-refreshing due to pending certificates...');
            loadCertificates();
        }, 10000); // Refresh every 10 seconds
    } else {
        if (indicator) {
            indicator.style.display = 'none';
        }
    }
}

