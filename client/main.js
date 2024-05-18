const width = 1872;
const height = 1404;

const rawCanvas = new OffscreenCanvas(width, height); // Define width and height as needed
let portrait = getQueryParam('portrait');
portrait = portrait !== null ? portrait === 'true' : false;
let flip = getQueryParam('flip');
flip = flip !== null ? flip === 'true' : false;
let withColor = getQueryParam('color', 'true');
withColor = withColor !== null ? withColor === 'true' : true;
let rate = parseInt(getQueryParamOrDefault('rate', '200'), 10);


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
//let imageData = ctx.createImageData(width, height); // width and height of your canvas
function getQueryParam(name) {
	const urlParams = new URLSearchParams(window.location.search);
	return urlParams.get(name);
}


window.onload = function() {
	// Function to get the value of a query parameter by name
	// Get the 'present' parameter from the URL
	const presentURL = getQueryParam('present');

	// Set the iframe source if the URL is available
	if (presentURL) {
		document.getElementById('content').src = presentURL;
	}
};

// Add an event listener for the 'beforeunload' event, which is triggered when the page is refreshed or closed
window.addEventListener('beforeunload', () => {
	// Send a termination signal to the worker before the page is unloaded
	streamWorker.postMessage({ type: 'terminate' });
	eventWorker.postMessage({ type: 'terminate' });
	gestureWorker.postMessage({ type: 'terminate' });
});
