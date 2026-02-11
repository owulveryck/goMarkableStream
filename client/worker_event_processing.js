let height;
let width;
let eventURL;
let portrait;
let draw;
let latestX;
let latestY;
let maxXValue;
let maxYValue;
let deviceModel = "Remarkable2";  // default
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
			deviceModel = event.data.deviceModel || "Remarkable2";
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
			// Device-specific coordinate transformations
			// RM2 (landscape native): Code 0=Y-axis, Code 1=X-axis
			// RMPP (portrait native): Code 0=X-axis, Code 1=Y-axis
			if (deviceModel === "RemarkablePaperPro") {
				// RMPP transformations
				if (portrait) {
					// this is landscape
					if (message.Code === 0) {
						latestY = scaleValue(message.Value, maxXValue, height);
					} else if (message.Code === 1) {
						latestX = width - scaleValue(message.Value, maxYValue, width);
					}
				} else {
					// this is portrait
					if (message.Code === 0) {
						latestX = scaleValue(message.Value, maxXValue, width);
					} else if (message.Code === 1) {
						latestY = scaleValue(message.Value, maxYValue, height);
					}
				}
			} else {
				// RM2 transformations (keep existing working logic)
				if (portrait) {
					if (message.Code === 0) {
						latestX = scaleValue(message.Value, maxYValue, width);
					} else if (message.Code === 1) {
						latestY = scaleValue(message.Value, maxXValue, height);
					}
				} else {
					if (message.Code === 0) {
						latestY = height - scaleValue(message.Value, maxYValue, height);
					} else if (message.Code === 1) {
						latestX = scaleValue(message.Value, maxXValue, width);
					}
				}
			}

			if (draw) {
				// Existing throttling logic remains unchanged
				const dx = Math.abs(latestX - lastSentX);
				const dy = Math.abs(latestY - lastSentY);
				if (dx < MIN_DELTA && dy < MIN_DELTA) return;

				if (!pendingUpdate) {
					pendingUpdate = true;
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
