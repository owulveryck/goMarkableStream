// UI interactions module

// Rotate button functionality
document.getElementById('rotate').addEventListener('click', function () {
    portrait = !portrait;
    this.classList.toggle('toggled');
    eventWorker.postMessage({ type: 'portrait', portrait: portrait });
    resizeVisibleCanvas();
    redrawScene(portrait, 1);

    // Show confirmation message
    showMessage(`Display ${portrait ? 'portrait' : 'landscape'} mode activated`, 2000);
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
        showMessage('Drawing layer on top', 2000);
    } else {
        iFrame.style.zIndex = 4;
        this.classList.add('toggled');
        showMessage('Content layer on top', 2000);
    }
});

// Laser toggle button functionality
document.getElementById('laserToggle').addEventListener('click', function () {
    laserEnabled = !laserEnabled;
    this.classList.toggle('toggled');
    if (!laserEnabled) {
        clearLaser();
    }
    showMessage(`Laser pointer ${laserEnabled ? 'enabled' : 'disabled'}`, 2000);
});

// Funnel toggle and URL copy
document.getElementById('funnelButton').addEventListener('click', async function() {
    try {
        // Get current status
        let response = await fetch('/funnel');
        if (!response.ok) throw new Error('Failed to fetch funnel status');
        let data = await response.json();

        if (!data.available) {
            showMessage('Tailscale mode not active', 2000);
            return;
        }

        // Toggle Funnel state
        const newState = !data.enabled;
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
                    showMessage(`Funnel enabled! URL copied: ${data.url}`, 4000);
                } else {
                    showMessage(`Funnel enabled! URL: ${data.url}`, 4000);
                }
            } catch (clipboardErr) {
                console.warn('Clipboard access denied:', clipboardErr);
                showMessage(`Funnel enabled! URL: ${data.url}`, 4000);
            }
        } else {
            // Hide QR code
            const qrContainer = document.getElementById('qrCodeContainer');
            if (qrContainer) {
                qrContainer.style.display = 'none';
            }

            // Restore version in footer
            if (footerElement) {
                fetchVersion().then(version => {
                    footerElement.textContent = `goMarkableStream ${version}`;
                    footerElement.title = '';
                });
            }
            showMessage('Funnel disabled', 2000);
        }
    } catch (error) {
        console.error('Funnel toggle error:', error);
        showMessage('Failed to toggle Funnel', 2000);
    }
});

