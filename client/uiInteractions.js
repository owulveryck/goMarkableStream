// UI interactions module

// ============================================
// State Persistence with localStorage
// ============================================

const STORAGE_KEYS = {
    PORTRAIT_MODE: 'goMarkableStream_portraitMode',
    LASER_ENABLED: 'goMarkableStream_laserEnabled',
    LAYER_ORDER: 'goMarkableStream_layerOrder'
};

// Load saved preferences on initialization
function loadSavedPreferences() {
    // Portrait mode
    const savedPortrait = localStorage.getItem(STORAGE_KEYS.PORTRAIT_MODE);
    if (savedPortrait !== null) {
        const isPortrait = savedPortrait === 'true';
        if (isPortrait !== portrait) {
            portrait = isPortrait;
            const rotateBtn = document.getElementById('rotate');
            rotateBtn.classList.toggle('toggled', portrait);
            rotateBtn.setAttribute('aria-pressed', portrait.toString());
            eventWorker.postMessage({ type: 'portrait', portrait: portrait });
            resizeVisibleCanvas();
            redrawScene(portrait, 1);
        }
    }

    // Laser enabled
    const savedLaser = localStorage.getItem(STORAGE_KEYS.LASER_ENABLED);
    if (savedLaser !== null) {
        const isLaserEnabled = savedLaser === 'true';
        if (isLaserEnabled !== laserEnabled) {
            laserEnabled = isLaserEnabled;
            const laserBtn = document.getElementById('laserToggle');
            laserBtn.classList.toggle('toggled', laserEnabled);
            laserBtn.setAttribute('aria-pressed', laserEnabled.toString());
            if (!laserEnabled) {
                clearLaser();
            }
        }
    }

    // Layer order (only if layers menu is visible)
    const savedLayerOrder = localStorage.getItem(STORAGE_KEYS.LAYER_ORDER);
    if (savedLayerOrder !== null && document.getElementById('layersMenuItem').style.display !== 'none') {
        const isContentOnTop = savedLayerOrder === 'content';
        if (isContentOnTop) {
            iFrame.style.zIndex = 4;
        }
        // Wait for updateLayerOrderUI to be defined
        setTimeout(() => {
            if (typeof updateLayerOrderUI === 'function') {
                updateLayerOrderUI(isContentOnTop);
            }
        }, 200);
    }
}

// Save preference to localStorage
function savePreference(key, value) {
    try {
        localStorage.setItem(key, value.toString());
    } catch (e) {
        console.warn('Failed to save preference:', e);
    }
}

// Initialize preferences after DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    // Small delay to ensure all components are initialized
    setTimeout(loadSavedPreferences, 100);
});

// ============================================
// Toast Notification System
// ============================================

function showToast(message, duration = 3000) {
    // Remove existing toast if any
    const existingToast = document.querySelector('.toast');
    if (existingToast) {
        existingToast.remove();
    }

    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    toast.setAttribute('role', 'status');
    toast.setAttribute('aria-live', 'polite');
    document.body.appendChild(toast);

    // Trigger animation
    requestAnimationFrame(() => {
        toast.classList.add('visible');
    });

    // Auto-dismiss
    setTimeout(() => {
        toast.classList.remove('visible');
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

// Help overlay functionality
function toggleHelpOverlay(show) {
    const overlay = document.getElementById('helpOverlay');
    if (!overlay) return;

    if (show === undefined) {
        overlay.classList.toggle('visible');
    } else if (show) {
        overlay.classList.add('visible');
    } else {
        overlay.classList.remove('visible');
    }
}

// Help button click handler
document.getElementById('helpButton').addEventListener('click', function() {
    toggleHelpOverlay(true);
});

// Help close button click handler
document.getElementById('helpCloseBtn').addEventListener('click', function() {
    toggleHelpOverlay(false);
});

// Close help overlay when clicking outside content
document.getElementById('helpOverlay').addEventListener('click', function(e) {
    if (e.target === this) {
        toggleHelpOverlay(false);
    }
});

// Fullscreen functionality
let isFullscreen = false;

function toggleFullscreen() {
    const container = document.getElementById('container');
    const fullscreenBtn = document.getElementById('fullscreenButton');
    const fullscreenHint = document.getElementById('fullscreenHint');
    const sidebar = document.querySelector('.sidebar');

    if (!document.fullscreenElement) {
        // Enter fullscreen
        container.requestFullscreen().then(() => {
            isFullscreen = true;
            fullscreenBtn.classList.add('toggled');
            fullscreenBtn.setAttribute('aria-pressed', 'true');

            // Auto-hide sidebar in fullscreen
            sidebar.classList.remove('active');
            const hamburgerMenu = document.getElementById('hamburgerMenu');
            if (hamburgerMenu) {
                hamburgerMenu.classList.remove('active');
                hamburgerMenu.setAttribute('aria-expanded', 'false');
            }

            // Show exit hint
            if (fullscreenHint) {
                fullscreenHint.classList.add('visible');
                // Auto-hide hint after 3 seconds
                setTimeout(() => {
                    fullscreenHint.classList.remove('visible');
                }, 3000);
            }

            showToast('Fullscreen mode activated');
        }).catch(err => {
            console.error('Error attempting to enable fullscreen:', err);
            showMessage('Fullscreen not supported', MessageDuration.QUICK);
        });
    } else {
        // Exit fullscreen
        document.exitFullscreen();
    }
}

// Listen for fullscreen changes
document.addEventListener('fullscreenchange', function() {
    const fullscreenBtn = document.getElementById('fullscreenButton');
    const fullscreenHint = document.getElementById('fullscreenHint');

    if (!document.fullscreenElement) {
        isFullscreen = false;
        fullscreenBtn.classList.remove('toggled');
        fullscreenBtn.setAttribute('aria-pressed', 'false');
        if (fullscreenHint) {
            fullscreenHint.classList.remove('visible');
        }
    }
});

// Fullscreen button click handler
const fullscreenButton = document.getElementById('fullscreenButton');
if (fullscreenButton) {
    fullscreenButton.addEventListener('click', toggleFullscreen);
}

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Don't trigger shortcuts when typing in input fields
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.isContentEditable) {
        return;
    }

    switch (e.key.toLowerCase()) {
        case 'f':
            // Fullscreen toggle
            if (document.getElementById('fullscreenButton')) {
                toggleFullscreen();
            }
            break;
        case 'r':
            // Rotate toggle
            document.getElementById('rotate').click();
            break;
        case 'l':
            // Laser toggle
            document.getElementById('laserToggle').click();
            break;
        case 'h':
        case 's':
            // Sidebar toggle (mobile and desktop)
            const sidebar = document.querySelector('.sidebar');
            const hamburgerMenu = document.getElementById('hamburgerMenu');
            const isActive = sidebar.classList.toggle('active');
            if (hamburgerMenu) {
                hamburgerMenu.classList.toggle('active', isActive);
                hamburgerMenu.setAttribute('aria-expanded', isActive.toString());
            }
            break;
        case 'c':
            // Copy share URL (when funnel is active)
            const funnelBtn = document.getElementById('funnelButton');
            if (funnelBtn && funnelBtn.classList.contains('toggled')) {
                const footerText = document.querySelector('.sidebar-footer small');
                if (footerText && footerText.textContent.startsWith('http')) {
                    navigator.clipboard.writeText(footerText.textContent)
                        .then(() => showToast('URL copied to clipboard'))
                        .catch(() => showToast('Failed to copy URL'));
                }
            }
            break;
        case 'p':
            // Download screenshot
            downloadScreenshot();
            break;
        case '?':
            // Help overlay
            toggleHelpOverlay(true);
            break;
        case 'escape':
            // Close help overlay and sidebar
            toggleHelpOverlay(false);
            // Close sidebar on mobile
            if (window.innerWidth <= 480) {
                const sidebar = document.querySelector('.sidebar');
                const hamburgerMenu = document.getElementById('hamburgerMenu');
                sidebar.classList.remove('active');
                if (hamburgerMenu) {
                    hamburgerMenu.classList.remove('active');
                    hamburgerMenu.setAttribute('aria-expanded', 'false');
                }
            }
            break;
    }
});

// Rotate button functionality
document.getElementById('rotate').addEventListener('click', function () {
    portrait = !portrait;
    this.classList.toggle('toggled');
    this.setAttribute('aria-pressed', portrait.toString());
    eventWorker.postMessage({ type: 'portrait', portrait: portrait });
    resizeVisibleCanvas();
    redrawScene(portrait, 1);

    // Save preference
    savePreference(STORAGE_KEYS.PORTRAIT_MODE, portrait);

    // Show confirmation message
    showMessage(`Display ${portrait ? 'portrait' : 'landscape'} mode activated`, MessageDuration.QUICK);
});

// Sidebar hover effect (desktop)
const sidebar = document.querySelector('.sidebar');
sidebar.addEventListener('mouseover', function () {
    sidebar.classList.add('active');
});
sidebar.addEventListener('mouseout', function () {
    sidebar.classList.remove('active');
});

// Hamburger menu toggle (mobile)
const hamburgerMenu = document.getElementById('hamburgerMenu');
if (hamburgerMenu) {
    hamburgerMenu.addEventListener('click', function(e) {
        e.stopPropagation();
        const isActive = sidebar.classList.contains('active');
        sidebar.classList.toggle('active');
        hamburgerMenu.classList.toggle('active');
        hamburgerMenu.setAttribute('aria-expanded', (!isActive).toString());
    });

    // Close sidebar when clicking outside on mobile
    document.addEventListener('click', function(e) {
        if (window.innerWidth <= 480) {
            if (!sidebar.contains(e.target) && !hamburgerMenu.contains(e.target)) {
                sidebar.classList.remove('active');
                hamburgerMenu.classList.remove('active');
                hamburgerMenu.setAttribute('aria-expanded', 'false');
            }
        }
    });

    // Prevent sidebar clicks from closing it
    sidebar.addEventListener('click', function(e) {
        e.stopPropagation();
    });
}

// Resize the canvas whenever the window is resized
window.addEventListener("resize", resizeVisibleCanvas);
resizeVisibleCanvas();

// Helper function to update layer order UI
function updateLayerOrderUI(isContentOnTop) {
    const layerOrderText = document.getElementById('layerOrderText');
    const layerOrderStatus = document.getElementById('layerOrderStatus');
    const switchBtn = document.getElementById('switchOrderButton');

    if (isContentOnTop) {
        if (layerOrderText) layerOrderText.textContent = 'Content on Top';
        if (layerOrderStatus) layerOrderStatus.textContent = 'Content on Top';
        switchBtn.classList.add('toggled');
        switchBtn.setAttribute('aria-pressed', 'true');
        switchBtn.setAttribute('aria-label', 'Content layer is on top. Click to switch to drawing on top');
    } else {
        if (layerOrderText) layerOrderText.textContent = 'Drawing on Top';
        if (layerOrderStatus) layerOrderStatus.textContent = 'Drawing on Top';
        switchBtn.classList.remove('toggled');
        switchBtn.setAttribute('aria-pressed', 'false');
        switchBtn.setAttribute('aria-label', 'Drawing layer is on top. Click to switch to content on top');
    }
}

// Mask drawing button functionality
document.getElementById('switchOrderButton').addEventListener('click', function () {
    // Swap z-index values
    const isLayerSwitched = iFrame.style.zIndex != 1;

    if (isLayerSwitched) {
        iFrame.style.zIndex = 1;
        savePreference(STORAGE_KEYS.LAYER_ORDER, 'drawing');
        updateLayerOrderUI(false);
        showMessage('Drawing layer on top', MessageDuration.QUICK);
    } else {
        iFrame.style.zIndex = 4;
        savePreference(STORAGE_KEYS.LAYER_ORDER, 'content');
        updateLayerOrderUI(true);
        showMessage('Content layer on top', MessageDuration.QUICK);
    }
});

// Laser toggle button functionality
document.getElementById('laserToggle').addEventListener('click', function () {
    laserEnabled = !laserEnabled;
    this.classList.toggle('toggled');
    this.setAttribute('aria-pressed', laserEnabled.toString());
    if (!laserEnabled) {
        clearLaser();
    }

    // Save preference
    savePreference(STORAGE_KEYS.LASER_ENABLED, laserEnabled);

    showMessage(`Laser pointer ${laserEnabled ? 'enabled' : 'disabled'}`, MessageDuration.QUICK);
});

// Funnel toggle and URL copy
document.getElementById('funnelButton').addEventListener('click', async function() {
    const funnelBtn = this;

    // Prevent double-click during async operation
    if (funnelBtn.disabled || funnelBtn.classList.contains('loading')) {
        return;
    }

    // Add loading state
    funnelBtn.disabled = true;
    funnelBtn.classList.add('loading');

    try {
        // Get current status
        let response = await authFetch('/funnel');
        if (!response.ok) throw new Error('Failed to fetch funnel status');
        let data = await response.json();

        if (!data.available) {
            showMessage('Tailscale mode not active', MessageDuration.NORMAL);
            return;
        }

        // Toggle Funnel state
        const newState = !data.enabled;

        // If enabling Funnel, stop local stream first to free the connection
        if (newState) {
            streamWorker.postMessage({ type: 'terminate' });
            updateConnectionStatus(ConnectionState.PAUSED);
            showMessage('Setting up public sharing...', MessageDuration.NORMAL);
        } else {
            showMessage('Disabling public sharing...', MessageDuration.NORMAL);
        }

        // Build headers with auth token
        const headers = { 'Content-Type': 'application/json' };
        const token = typeof getAuthToken === 'function' ? getAuthToken() : null;
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        response = await fetch('/funnel', {
            method: 'POST',
            headers: headers,
            body: JSON.stringify({ enable: newState })
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to toggle funnel: ${errorText}`);
        }
        data = await response.json();

        // Update button visual state
        funnelBtn.classList.toggle('toggled', data.enabled);
        funnelBtn.setAttribute('aria-pressed', data.enabled.toString());

        // Update footer with URL or version
        const footerElement = document.querySelector('.sidebar-footer small');

        if (data.enabled && data.url) {
            // Show QR code
            const qrContainer = document.getElementById('qrCodeContainer');
            if (qrContainer && typeof generateQRCode === 'function') {
                const qrSvg = generateQRCode(data.url, 120);
                qrContainer.innerHTML = qrSvg + `
                    <div class="qr-label">
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M16,1H4C2.9,1 2,1.9 2,3V17H4V3H16V1M19,5H8C6.9,5 6,5.9 6,7V21C6,22.1 6.9,23 8,23H19C20.1,23 21,22.1 21,21V7C21,5.9 20.1,5 19,5M19,21H8V7H19V21Z"/>
                        </svg>
                        Click to copy URL
                    </div>
                `;
                qrContainer.style.display = 'flex';

                // Add click handler for copying URL
                qrContainer.onclick = async function() {
                    try {
                        await navigator.clipboard.writeText(data.url);

                        // Visual feedback
                        qrContainer.classList.add('copied');
                        const label = qrContainer.querySelector('.qr-label');
                        if (label) {
                            label.innerHTML = `
                                <svg viewBox="0 0 24 24" fill="currentColor" style="animation: checkmark 0.3s ease">
                                    <path d="M21,7L9,19L3.5,13.5L4.91,12.09L9,16.17L19.59,5.59L21,7Z"/>
                                </svg>
                                URL copied!
                            `;
                        }

                        showToast('Public URL copied to clipboard');

                        // Reset after 2 seconds
                        setTimeout(() => {
                            qrContainer.classList.remove('copied');
                            if (label) {
                                label.innerHTML = `
                                    <svg viewBox="0 0 24 24" fill="currentColor">
                                        <path d="M16,1H4C2.9,1 2,1.9 2,3V17H4V3H16V1M19,5H8C6.9,5 6,5.9 6,7V21C6,22.1 6.9,23 8,23H19C20.1,23 21,22.1 21,21V7C21,5.9 20.1,5 19,5M19,21H8V7H19V21Z"/>
                                    </svg>
                                    Click to copy URL
                                `;
                            }
                        }, 2000);
                    } catch (err) {
                        console.error('Failed to copy URL:', err);
                        showToast('Failed to copy URL');
                    }
                };
            }

            // Show URL in footer
            if (footerElement) {
                footerElement.textContent = data.url;
                footerElement.title = 'Public Funnel URL';
            }

            // Show temporary credentials if available
            if (data.tempCredentials) {
                displayFunnelCredentials(data.tempCredentials);
            }

            // Try to copy URL to clipboard (requires secure context)
            try {
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    await navigator.clipboard.writeText(data.url);
                    showToast('Funnel URL copied to clipboard');
                }
            } catch (clipboardErr) {
                console.warn('Clipboard access denied:', clipboardErr);
            }
            // Update status indicator to show paused state
            updateConnectionStatus(ConnectionState.PAUSED);
            // Show persistent status message
            showStatusMessage('Local stream paused - Funnel sharing active');
            showMessage('Public sharing enabled! Share the QR code or URL', MessageDuration.IMPORTANT);
        } else {
            // Hide QR code
            const qrContainer = document.getElementById('qrCodeContainer');
            if (qrContainer) {
                qrContainer.style.display = 'none';
            }

            // Hide temporary credentials
            hideFunnelCredentials();

            // Restore footer text
            if (footerElement) {
                footerElement.textContent = 'goMarkableStream';
                footerElement.title = '';
            }

            // Hide persistent status message before restarting stream
            hideStatusMessage();

            // Update status to connecting
            updateConnectionStatus(ConnectionState.CONNECTING);

            // Recreate and reinitialize stream worker
            streamWorker = new Worker('worker_stream_processing.js');
            initStreamWorker();
            showMessage('Public sharing disabled - Resuming local stream', MessageDuration.NORMAL);
        }
    } catch (error) {
        console.error('Funnel toggle error:', error);
        showMessage('Failed to toggle public sharing', MessageDuration.IMPORTANT);
    } finally {
        // Remove loading state
        funnelBtn.disabled = false;
        funnelBtn.classList.remove('loading');
    }
});

// ============================================
// Funnel Credentials Display
// ============================================

// Display temporary credentials in the UI
function displayFunnelCredentials(credentials) {
    const container = document.getElementById('credentialsContainer');
    const usernameEl = document.getElementById('tempUsername');
    const passwordEl = document.getElementById('tempPassword');

    if (!container || !usernameEl || !passwordEl) return;

    if (credentials && credentials.username && credentials.password) {
        usernameEl.textContent = credentials.username;
        passwordEl.textContent = credentials.password;
        container.style.display = 'block';
    } else {
        container.style.display = 'none';
    }
}

// Hide credentials from UI
function hideFunnelCredentials() {
    const container = document.getElementById('credentialsContainer');
    if (container) {
        container.style.display = 'none';
    }
}

// Initialize copy buttons for credentials
function initCredentialsCopyButtons() {
    const copyButtons = document.querySelectorAll('#credentialsContainer .copy-btn');
    copyButtons.forEach(btn => {
        btn.addEventListener('click', async function(e) {
            e.stopPropagation();
            const copyType = this.dataset.copy;
            let textToCopy = '';

            if (copyType === 'username') {
                textToCopy = document.getElementById('tempUsername')?.textContent || '';
            } else if (copyType === 'password') {
                textToCopy = document.getElementById('tempPassword')?.textContent || '';
            }

            if (!textToCopy) return;

            try {
                await navigator.clipboard.writeText(textToCopy);

                // Visual feedback
                this.classList.add('copied');
                showToast(`${copyType === 'username' ? 'Username' : 'Password'} copied`);

                // Reset after animation
                setTimeout(() => {
                    this.classList.remove('copied');
                }, 1500);
            } catch (err) {
                console.error('Failed to copy:', err);
                showToast('Failed to copy');
            }
        });
    });
}

// Initialize copy buttons on page load
document.addEventListener('DOMContentLoaded', initCredentialsCopyButtons);

// Screenshot download functionality
async function downloadScreenshot() {
    const screenshotBtn = document.getElementById('screenshotBtn');
    if (!screenshotBtn || screenshotBtn.classList.contains('loading')) {
        return;
    }

    // Add loading state
    screenshotBtn.classList.add('loading');

    try {
        const response = await authFetch('/screenshot');

        if (!response.ok) {
            throw new Error(`Failed to download screenshot: ${response.status}`);
        }

        // Get the blob from the response
        const blob = await response.blob();

        // Create a download link
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;

        // Extract filename from Content-Disposition header if available
        const contentDisposition = response.headers.get('Content-Disposition');
        let filename = 'remarkable_screenshot.png';
        if (contentDisposition) {
            const match = contentDisposition.match(/filename="(.+)"/);
            if (match) {
                filename = match[1];
            }
        }

        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        showToast('Screenshot downloaded');
    } catch (error) {
        console.error('Screenshot download error:', error);
        showToast('Failed to download screenshot');
    } finally {
        screenshotBtn.classList.remove('loading');
    }
}

// Screenshot button click handler
const screenshotBtn = document.getElementById('screenshotBtn');
if (screenshotBtn) {
    screenshotBtn.addEventListener('click', downloadScreenshot);
}

