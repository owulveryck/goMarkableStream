let height;
let width;
let eventURL;
let portrait;
let draw;
let latestX;
let latestY;
let maxXValue;
let maxYValue;
let authToken = null;

// Throttling variables for laser pointer updates
let pendingUpdate = false;
let lastSentX = -1;
let lastSentY = -1;
const MIN_DELTA = 2; // minimum pixel change to trigger update

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			eventURL = event.data.eventURL;
			portrait = event.data.portrait;
            maxXValue = event.data.maxXValue;
            maxYValue = event.data.maxYValue;
			authToken = event.data.authToken || null;
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
	// EventSource doesn't support custom headers, so pass token as query param
	let url = eventURL;
	if (authToken) {
		const separator = url.includes('?') ? '&' : '?';
		url = `${url}${separator}token=${encodeURIComponent(authToken)}`;
	}
	const eventSource = new EventSource(url);
	draw = true;
	eventSource.onmessage = (event) => {
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
			// Device axes are rotated 90° from screen coordinates:
			// Code 0 = device Y-axis = horizontal movement on device
			// Code 1 = device X-axis = vertical movement on device
			if (portrait) {
				if (message.Code === 0) { // device Y (horizontal) → screen X (NOT inverted)
					latestX = scaleValue(message.Value, maxYValue, width);
				} else if (message.Code === 1) { // device X (vertical) → screen Y (NOT inverted)
					latestY = scaleValue(message.Value, maxXValue, height);
				}
			} else {
				if (message.Code === 0) { // device Y → screen Y (inverted)
					latestY = height - scaleValue(message.Value, maxYValue, height);
				} else if (message.Code === 1) { // device X → screen X
					latestX = scaleValue(message.Value, maxXValue, width);
				}
			}
			if (draw) {
				// Skip if position hasn't changed meaningfully
				const dx = Math.abs(latestX - lastSentX);
				const dy = Math.abs(latestY - lastSentY);
				if (dx < MIN_DELTA && dy < MIN_DELTA) return;

				if (!pendingUpdate) {
					pendingUpdate = true;
					// Align with display refresh (~60fps = 16ms)
					setTimeout(() => {
						postMessage({ type: 'update', X: latestX, Y: latestY });
						lastSentX = latestX;
						lastSentY = latestY;
						pendingUpdate = false;
					}, 16);
				}
			}
		}
	}

	eventSource.onerror = () => {
		postMessage({
			type: 'error',
			message: "EventSource error",
		});
		console.error('EventSource error occurred.');
	};

	eventSource.onclose = () => {
		postMessage({
			type: 'error',
			message: 'Connection closed'
		});
		console.log('EventSource connection closed.');
	};
}

// Function to scale the incoming value to the canvas size
function scaleValue(value, maxValue, canvasSize) {
	return (value / maxValue) * canvasSize;
}
