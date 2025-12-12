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
        const response = await fetch('/certs');
        
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
                <div class="cert-name">${escapeHtml(cert.common_name)}</div>
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
    const modal = document.getElementById('certModal');
    if (event.target == modal) {
        modal.style.display = 'none';
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

