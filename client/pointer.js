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
			if (rotate) {
				if (message.Code === 1) { // Horizontal position
					latestX = scaleValue(message.Value, MAX_X_VALUE, canvasPresent.width);
				} else if (message.Code === 0) { // Vertical position
					latestY = canvasPresent.height - scaleValue(message.Value, MAX_Y_VALUE, canvasPresent.height);
				}
			} else {
				if (message.Code === 1) { // Horizontal position
					latestX = canvasPresent.width - scaleValue(message.Value, MAX_X_VALUE, canvasPresent.width);
				} else if (message.Code === 0) { // Vertical position
					latestY = scaleValue(message.Value, MAX_Y_VALUE, canvasPresent.height);
				}
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
function stopWebSocket() {
    if (ws) {
        ws.close();
    }
	clearLaser();
}

function isWebSocketConnected(ws) {
    return ws && ws.readyState === WebSocket.OPEN;
}
connectWebSocket();
