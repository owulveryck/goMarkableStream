const rawCanvas = new OffscreenCanvas(screenWidth, screenHeight); // Define width and height as needed
let portrait = getQueryParam('portrait');
portrait = portrait !== null ? portrait === 'true' : false;

defaultFlip = false;
// If this is the Paper Pro, we don't need to flip the image.
if (DeviceModel === 'RemarkablePaperPro') {
	defaultFlip = false;
}
let flip = getBoolQueryParam('flip', defaultFlip);

let withColor = getQueryParam('color', 'true');
withColor = withColor !== null ? withColor === 'true' : true;
let rate = parseInt(getQueryParamOrDefault('rate', '200'), 10);

// Remarkable Paper Pro uses BGRA format.
let useBGRA = false;
if (DeviceModel === 'RemarkablePaperPro') {
	useBGRA = true;
};

//let portrait = false;
// Get the 'present' parameter from the URL
//const presentURL = getQueryParam('present');// Assuming rawCanvas is an OffscreenCanvas that's already been defined
const ctx = rawCanvas.getContext('2d');
const visibleCanvas = document.getElementById("canvas");
const iFrame = document.getElementById("content");
const messageDiv = document.getElementById('message');


// Initialize the worker
const streamWorker = new Worker('worker_stream_processing.js');
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
	// Function to get the value of a query parameter by name
	// Get the 'present' parameter from the URL
	const presentURL = getQueryParam('present');

	// Set the iframe source if the URL is available
	if (presentURL) {
		document.getElementById('content').src = presentURL;
	}
	
	// Update version in the sidebar footer
	const version = await fetchVersion();
	const versionElement = document.querySelector('.sidebar-footer small');
	if (versionElement) {
		versionElement.textContent = `goMarkableStream ${version}`;
	}
};

// Add an event listener for the 'beforeunload' event, which is triggered when the page is refreshed or closed
window.addEventListener('beforeunload', () => {
	// Send a termination signal to the worker before the page is unloaded
	streamWorker.postMessage({ type: 'terminate' });
	eventWorker.postMessage({ type: 'terminate' });
	gestureWorker.postMessage({ type: 'terminate' });
});
