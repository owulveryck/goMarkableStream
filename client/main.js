const width = 1872;
const height = 1404;

const rawCanvas = new OffscreenCanvas(width, height); // Define width and height as needed
const visibleCanvas = document.getElementById("canvas");
const canvasPresent = document.getElementById("canvasPresent");

// Initialize the worker
const worker = new Worker('worker_stream_processing.js');

// Send the OffscreenCanvas to the worker for initialization
worker.postMessage({ 
	type: 'init', 
	width: width, 
	height: height 
});

// Listen for updates from the worker
worker.onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'update':
			// Handle the update
			const data = event.data.data;
			// Assuming rawCanvas is an OffscreenCanvas that's already been defined
			const ctx = rawCanvas.getContext('2d');

			// Create an ImageData object with the provided Uint8ClampedArray
			const imageData = new ImageData(data, width, height);

			// Draw the ImageData onto the OffscreenCanvas
			ctx.putImageData(imageData, 0, 0);
			renderCanvas(rawCanvas,visibleCanvas);
			//resizeVisibleCanvas();
			break;
		case 'error':
			console.error('Error from worker:', event.data.message);
			waiting(event.data.message)
			// Handle the error, maybe show a user-friendly message or take some corrective action
			break;
			// ... handle other message types as needed
	}
};

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
// connectWebSocket();

