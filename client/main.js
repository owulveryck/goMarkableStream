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

const ctxCanvasPresent = canvasPresent.getContext('2d');

// Variables to store the latest positions
let latestX = canvasPresent.width / 2;
let latestY = canvasPresent.height / 2;
let ws;
// Constants for the maximum values from the WebSocket messages
const MAX_X_VALUE = 15725;
const MAX_Y_VALUE = 20966;

// Function to draw the laser pointer
function drawLaser(x, y) {
	ctxCanvasPresent.clearRect(0, 0, canvasPresent.width, canvasPresent.height); // Clear the canvasPresent
	ctxCanvasPresent.beginPath();
	ctxCanvasPresent.arc(x, y, 10, 0, 2 * Math.PI, false); // Draw a circle for the laser pointer
	ctxCanvasPresent.fillStyle = 'red';
	ctxCanvasPresent.fill();
}

// Function to clear the laser pointer
function clearLaser() {
	ctxCanvasPresent.clearRect(0, 0, canvasPresent.width, canvasPresent.height); // Clear the canvasPresent
}

// Function to establish a WebSocket connection
function connectWebSocket() {
	// Determine the WebSocket protocol based on the current window protocol
	const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	const wsURL = `${wsProtocol}//${window.location.host}/events`;
	let draw = true;

	ws = new WebSocket(wsURL);

	ws.onmessage = (event) => {
		const message = JSON.parse(event.data);
		//if (message.Type === 0) {
		// Code 0: Clear the laser pointer
		//	clearLaser();
		//} else if (message.Type === 3) {
		if (message.Type === 3) {
			if (message.Code === 24) {
				draw = false;	
				clearLaser();
			} else if (message.Code === 25) {
				draw = true;	

			}
		}
		if (message.Type === 3) {
			// Code 3: Update and draw laser pointer
			if (message.Code === 1) { // Horizontal position
				latestX = scaleValue(message.Value, MAX_X_VALUE, canvasPresent.width);
			} else if (message.Code === 0) { // Vertical position
				latestY = canvasPresent.height - scaleValue(message.Value, MAX_Y_VALUE, canvasPresent.height);
			}
			if (draw) {
				drawLaser(latestX, latestY);
			}
		}
	};

	ws.onerror = () => {
		console.error('WebSocket error occurred. Attempting to reconnect...');
		//setTimeout(connectWebSocket, 3000); // Reconnect after 3 seconds
	};

	ws.onclose = () => {
		console.log('WebSocket connection closed. Attempting to reconnect...');
		//setTimeout(connectWebSocket, 3000); // Reconnect after 3 seconds
	};
}
// Function to scale the incoming value to the canvas size
function scaleValue(value, maxValue, canvasSize) {
	return (value / maxValue) * canvasSize;
}

// Initial WebSocket connection
connectWebSocket();
