// UI interactions module

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

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Don't trigger shortcuts when typing in input fields
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.isContentEditable) {
        return;
    }

    switch (e.key.toLowerCase()) {
        case 'r':
            // Rotate toggle
            document.getElementById('rotate').click();
            break;
        case 'l':
            // Laser toggle
            document.getElementById('laserToggle').click();
            break;
        case '?':
            // Help overlay
            toggleHelpOverlay(true);
            break;
        case 'escape':
            // Close help overlay
            toggleHelpOverlay(false);
            break;
    }
});

// Rotate button functionality
document.getElementById('rotate').addEventListener('click', function () {
    portrait = !portrait;
    this.classList.toggle('toggled');
    eventWorker.postMessage({ type: 'portrait', portrait: portrait });
    resizeVisibleCanvas();
    redrawScene(portrait, 1);

    // Show confirmation message
    showMessage(`Display ${portrait ? 'portrait' : 'landscape'} mode activated`, MessageDuration.QUICK);
});

// Sidebar hover effect
const sidebar = document.querySelector('.sidebar');
sidebar.addEventListener('mouseover', function () {
    sidebar.classList.add('active');
});
sidebar.addEventListener('mouseout', function () {
    sidebar.classList.remove('active');
});

// Resize the canvas whenever the window is resized
window.addEventListener("resize", resizeVisibleCanvas);
resizeVisibleCanvas();

// Mask drawing button functionality
document.getElementById('switchOrderButton').addEventListener('click', function () {
    // Swap z-index values
    const isLayerSwitched = iFrame.style.zIndex != 1;

    if (isLayerSwitched) {
        iFrame.style.zIndex = 1;
        this.classList.remove('toggled');
        showMessage('Drawing layer on top', MessageDuration.QUICK);
    } else {
        iFrame.style.zIndex = 4;
        this.classList.add('toggled');
        showMessage('Content layer on top', MessageDuration.QUICK);
    }
});

// Laser toggle button functionality
document.getElementById('laserToggle').addEventListener('click', function () {
    laserEnabled = !laserEnabled;
    this.classList.toggle('toggled');
    if (!laserEnabled) {
        clearLaser();
    }
    showMessage(`Laser pointer ${laserEnabled ? 'enabled' : 'disabled'}`, MessageDuration.QUICK);
});

// Funnel toggle and URL copy
document.getElementById('funnelButton').addEventListener('click', async function() {
    try {
        // Get current status
        let response = await fetch('/funnel');
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
            showMessage('Enabling public sharing...', MessageDuration.NORMAL);
        }

        response = await fetch('/funnel', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ enable: newState })
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to toggle funnel: ${errorText}`);
        }
        data = await response.json();

        // Update button visual state
        this.classList.toggle('toggled', data.enabled);

        // Update footer with URL or version
        const footerElement = document.querySelector('.sidebar-footer small');

        if (data.enabled && data.url) {
            // Show QR code
            const qrContainer = document.getElementById('qrCodeContainer');
            if (qrContainer && typeof generateQRCode === 'function') {
                const qrSvg = generateQRCode(data.url, 120);
                qrContainer.innerHTML = qrSvg;
                qrContainer.style.display = 'flex';
            }

            // Show URL in footer
            if (footerElement) {
                footerElement.textContent = data.url;
                footerElement.title = 'Public Funnel URL';
            }
            // Try to copy URL to clipboard (requires secure context)
            try {
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    await navigator.clipboard.writeText(data.url);
                }
            } catch (clipboardErr) {
                console.warn('Clipboard access denied:', clipboardErr);
            }
            // Update status indicator to show paused state
            updateConnectionStatus(ConnectionState.PAUSED);
            // Show persistent status message
            showStatusMessage('Local stream paused - Funnel sharing active');
        } else {
            // Hide QR code
            const qrContainer = document.getElementById('qrCodeContainer');
            if (qrContainer) {
                qrContainer.style.display = 'none';
            }

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
        }
    } catch (error) {
        console.error('Funnel toggle error:', error);
        showMessage('Failed to toggle public sharing', MessageDuration.IMPORTANT);
    }
});

