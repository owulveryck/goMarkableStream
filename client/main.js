const width = 1872;
const height = 1404;

const rawCanvas = new OffscreenCanvas(width, height); // Define width and height as needed
let portrait = false;
// Assuming rawCanvas is an OffscreenCanvas that's already been defined
const ctx = rawCanvas.getContext('2d');
const visibleCanvas = document.getElementById("canvas");
const canvasPresent = document.getElementById("canvasPresent");
const iFrame = document.getElementById("content");
const messageDiv = document.getElementById('message');


// Initialize the worker
const streamWorker = new Worker('worker_stream_processing.js');
const eventWorker = new Worker('worker_event_processing.js');
//let imageData = ctx.createImageData(width, height); // width and height of your canvas
window.onload = function() {
	// Function to get the value of a query parameter by name
	function getQueryParam(name) {
		const urlParams = new URLSearchParams(window.location.search);
		return urlParams.get(name);
	}

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
});
