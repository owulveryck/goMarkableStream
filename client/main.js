const rawCanvas = new OffscreenCanvas(screenWidth, screenHeight); // Define width and height as needed
let laserEnabled = true;
let portrait = getQueryParam('portrait');
portrait = portrait !== null ? portrait === 'true' : false;

defaultFlip = false;
// If this is the Paper Pro, we don't need to flip the image.
if (DeviceModel === 'RemarkablePaperPro') {
	defaultFlip = false;
}
let flip = getBoolQueryParam('flip', defaultFlip);

let rate = parseInt(getQueryParamOrDefault('rate', '200'), 10);

// Use BGRA format flag from server (Paper Pro or RM2 firmware 3.24+)
let useBGRA = UseBGRA;

//let portrait = false;
// Get the 'present' parameter from the URL
//const presentURL = getQueryParam('present');// Assuming rawCanvas is an OffscreenCanvas that's already been defined
const ctx = rawCanvas.getContext('2d');
const visibleCanvas = document.getElementById("canvas");
const iFrame = document.getElementById("content");
const messageDiv = document.getElementById('message');


// Initialize the worker
let streamWorker = new Worker('worker_stream_processing.js');
const eventWorker = new Worker('worker_event_processing.js');
const gestureWorker = new Worker('worker_gesture_processing.js');
function getQueryParamOrDefault(param, defaultValue) {
    const urlParams = new URLSearchParams(window.location.search);
    const value = urlParams.get(param);
    return value !== null ? value : defaultValue;
}

//let imageData = ctx.createImageData(screenWidth, screenHeight); // width and height of your canvas
function getQueryParam(name) {
	const urlParams = new URLSearchParams(window.location.search);
	return urlParams.get(name);
}

function getBoolQueryParam(param, defaultValue = false) {
    value = getQueryParam(param);

    if (value === null) {
        return defaultValue;
    }

    return value === 'true';
}

window.onload = async function() {
	// Initialize auth UI
	initAuthUI();

	// Check if JWT auth is enabled and user is not authenticated
	if (JWTEnabled && !isAuthenticated()) {
		showLoginModal();
		return; // Don't proceed until user logs in
	}

	// Function to get the value of a query parameter by name
	// Get the 'present' parameter from the URL
	const presentURL = getQueryParam('present');

	// Set the iframe source if the URL is available
	if (presentURL) {
		document.getElementById('content').src = presentURL;
		document.getElementById('layersMenuItem').style.display = '';  // Show layers menu
	}

	// Show onboarding hint for first-time users
	showOnboardingHint();

	// Check Funnel availability and status
	try {
		const funnelResponse = await authFetch('/funnel');
		if (funnelResponse.ok) {
			const funnelData = await funnelResponse.json();
			if (funnelData.available) {
				document.getElementById('funnelMenuItem').style.display = '';
				if (funnelData.enabled) {
					document.getElementById('funnelButton').classList.add('toggled');
					document.getElementById('funnelButton').setAttribute('aria-pressed', 'true');

					const footerElement = document.querySelector('.sidebar-footer small');
					// Show Funnel URL in footer
					if (footerElement && funnelData.url) {
						footerElement.textContent = funnelData.url;
						footerElement.title = 'Public Funnel URL';

						// Generate and show QR code
						const qrContainer = document.getElementById('qrCodeContainer');
						if (qrContainer && typeof generateQRCode === 'function') {
							const qrSvg = generateQRCode(funnelData.url, 120);
							qrContainer.innerHTML = qrSvg;
							qrContainer.style.display = 'flex';
						}
					}

					// Display temporary credentials if available
					if (funnelData.tempCredentials && typeof displayFunnelCredentials === 'function') {
						displayFunnelCredentials(funnelData.tempCredentials);
					}
				}
			}
		}
	} catch (error) {
		console.error('Error checking funnel status:', error);
	}
};

// Add an event listener for the 'beforeunload' event, which is triggered when the page is refreshed or closed
window.addEventListener('beforeunload', () => {
	// Send a termination signal to the worker before the page is unloaded
	streamWorker.postMessage({ type: 'terminate' });
	eventWorker.postMessage({ type: 'terminate' });
	gestureWorker.postMessage({ type: 'terminate' });
});
