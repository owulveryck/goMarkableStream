let height;
let width;
let wsURL;
let portrait;
let draw; 
let latestX;
let latestY;
// Constants for the maximum values from the WebSocket messages
const MAX_X_VALUE = 15725;
const MAX_Y_VALUE = 20966;

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			wsURL = event.data.wsURL;
			portrait = event.data.portrait;
			initiateEventsListener();
			break;
		case 'portrait':
			portrait = event.data.portrait;
			// Handle the error, maybe show a user-friendly message or take some corrective action
			break;
		case 'terminate':
			console.log("terminating worker");
			close();
			break;
	}
};


async function initiateEventsListener() {
	console.log(portrait);
	const RETRY_DELAY_MS = 3000; // Delay before retrying the connection (in milliseconds)
	ws = new WebSocket(wsURL);
	draw = true;
	ws.onmessage = (event) => {
		const message = JSON.parse(event.data);
		if (message.Type === 3) {
			if (message.Code === 24) {
				draw = false;	
				postMessage({ type: 'clear' });
				//						clearLaser();
			} else if (message.Code === 25) {
				draw = true;	

			}
		}
		if (message.Type === 3) {
			// Code 3: Update and draw laser pointer
			if (portrait) {
				if (message.Code === 1) { // Horizontal position
					latestX = scaleValue(message.Value, MAX_X_VALUE, width);
				} else if (message.Code === 0) { // Vertical position
					latestY = height - scaleValue(message.Value, MAX_Y_VALUE, height);
				}
			} else {
				// wrong
				if (message.Code === 1) { // Horizontal position
					latestY = scaleValue(message.Value, MAX_X_VALUE, height);
				} else if (message.Code === 0) { // Vertical position
					latestX = scaleValue(message.Value, MAX_Y_VALUE, width);
				}
			}
			if (draw) {
				postMessage({ type: 'update', X: latestX, Y: latestY });
			}
		}
	}

	ws.onerror = () => {
		postMessage({
			type: 'error',
			message: "websocket error",
		});
		console.error('WebSocket error occurred. Attempting to reconnect...');
		//setTimeout(connectWebSocket, 3000); // Reconnect after 3 seconds
	};

	ws.onclose = () => {
		postMessage({
			type: 'error',
			message: 'closed connection'
		});
		console.log('WebSocket connection closed. Attempting to reconnect...');
		//setTimeout(connectWebSocket, 3000); // Reconnect after 3 seconds
	};
}

// Function to scale the incoming value to the canvas size
function scaleValue(value, maxValue, canvasSize) {
	return (value / maxValue) * canvasSize;
}

