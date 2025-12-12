// Global state
let certificates = [];

// Load certificates on page load
document.addEventListener('DOMContentLoaded', function() {
    loadCertificates();
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

    const status = getCertificateStatus(cert.expires_on);
    const expiryDate = new Date(cert.expires_on);
    const issuedDate = new Date(cert.issued_on);
    const daysUntilExpiry = Math.floor((expiryDate - new Date()) / (1000 * 60 * 60 * 24));

    card.innerHTML = `
        <div class="cert-header">
            <div class="cert-title">
                <div class="cert-name-row">
                    <div class="cert-name">${escapeHtml(cert.common_name)}</div>
                    ${cert.managed ? '<span class="managed-badge">Managed</span>' : '<span class="unmanaged-badge">Unmanaged</span>'}
                </div>
                <div class="cert-issuer">Issuer: ${cert.issuers && cert.issuers.length > 0 ? escapeHtml(cert.issuers[0]) : 'Unknown'}</div>
            </div>
            <div class="cert-status status-${status.class}">${status.text}</div>
        </div>

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

        ${cert.sans && cert.sans.length > 0 ? `
            <div class="cert-detail">
                <div class="detail-label">Subject Alternative Names (SANs)</div>
                <ul class="sans-list">
                    ${cert.sans.map(san => `<li>${escapeHtml(san)}</li>`).join('')}
                </ul>
            </div>
        ` : ''}

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
    
    // Load domains
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
            const domainItem = document.createElement('div');
            domainItem.className = 'domain-item';
            domainItem.innerHTML = `
                <div class="domain-name">
                    <i class="fi fi-rr-globe"></i>
                    <span>${escapeHtml(domain)}</span>
                </div>
                <button class="btn btn-danger btn-small" onclick="removeDomain('${escapeForJs(domain)}')">
                    <i class="fi fi-rr-trash"></i>
                    <span>Remove</span>
                </button>
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
    const input = document.getElementById('newDomainInput');
    const messageEl = document.getElementById('domainAddMessage');
    const domain = input.value.trim();
    
    if (!domain) {
        showMessage(messageEl, 'Please enter a domain name', 'error');
        return;
    }
    
    // Basic validation
    if (!/^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(domain)) {
        showMessage(messageEl, 'Invalid domain format', 'error');
        return;
    }
    
    try {
        const response = await fetch('/api/domains', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ domain: domain })
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'Failed to add domain');
        }
        
        showMessage(messageEl, `Domain ${domain} added successfully! Triggering certificate issuance...`, 'success');
        input.value = '';
        
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

async function removeDomain(domain) {
    if (!confirm(`Are you sure you want to remove ${domain}?\n\nNote: The certificate will remain in Vault but won't be renewed.`)) {
        return;
    }
    
    try {
        const response = await fetch(`/api/domains/${encodeURIComponent(domain)}`, {
            method: 'DELETE'
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            throw new Error(data.error || 'Failed to remove domain');
        }
        
        // Reload domain list
        await loadDomains();
        
        // Optionally reload certificates
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

