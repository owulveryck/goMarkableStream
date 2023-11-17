const width = 1872;
const height = 1404;

const rawCanvas = new OffscreenCanvas(width, height); // Define width and height as needed
// Assuming rawCanvas is an OffscreenCanvas that's already been defined
const ctx = rawCanvas.getContext('2d');
const visibleCanvas = document.getElementById("canvas");
const canvasPresent = document.getElementById("canvasPresent");
const iFrame = document.getElementById("content");

// Initialize the worker
const streamWorker = new Worker('worker_stream_processing.js');
const eventWorker = new Worker('worker_event_processing.js');

// Send the OffscreenCanvas to the worker for initialization
streamWorker.postMessage({ 
	type: 'init', 
	width: width, 
	height: height 
});

let imageData = ctx.createImageData(width, height); // width and height of your canvas

// Listen for updates from the worker
streamWorker.onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'update':
			// Handle the update
			const data = event.data.data;
			updateTexture(data);

			//imageData.data.set(data);

			// Draw the ImageData onto the OffscreenCanvas
			//ctx.putImageData(imageData, 0, 0);
			//renderCanvas(rawCanvas,visibleCanvas);

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


// Determine the WebSocket protocol based on the current window protocol
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsURL = `${wsProtocol}//${window.location.host}/events`;
// Send the OffscreenCanvas to the worker for initialization
eventWorker.postMessage({ 
	type: 'init', 
	width: width, 
	height: height, 
	rotate: true,
	wsURL: wsURL
});

// Listen for updates from the worker
eventWorker.onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'clear':
			clearLaser();
			break;
		case 'update':
			// Handle the update
			const X = event.data.X;
			const Y = event.data.Y;
			drawLaser(X,Y);

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

