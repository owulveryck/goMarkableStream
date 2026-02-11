// Reconnection state
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;
let reconnectTimeout = null;
let isReconnecting = false;

// Calculate reconnection delay with exponential backoff (1s, 2s, 4s, 8s, 16s)
function getReconnectDelay() {
	return Math.pow(2, reconnectAttempts) * 1000;
}

// Attempt to reconnect the stream
function attemptStreamReconnect() {
	if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
		updateConnectionStatus(ConnectionState.DISCONNECTED);
		hideReconnectBanner();
		showConnectionError(
			'Unable to connect to the stream after multiple attempts. Please check your connection and try again.',
			true,
			() => {
				reconnectAttempts = 0;
				attemptStreamReconnect();
			}
		);
		isReconnecting = false;
		return;
	}

	isReconnecting = true;
	reconnectAttempts++;
	updateConnectionStatus(ConnectionState.RECONNECTING);
	showReconnectBanner(reconnectAttempts, MAX_RECONNECT_ATTEMPTS);

	const delay = getReconnectDelay();
	console.log(`Reconnecting in ${delay/1000}s (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);

	reconnectTimeout = setTimeout(() => {
		// Just replace the worker - don't send terminate
		// The old worker will be garbage collected
		streamWorker = new Worker('worker_stream_processing.js');
		// Reset isReconnecting so next error can trigger another attempt
		isReconnecting = false;
		initStreamWorker();
	}, delay);
}

// Reset reconnection state on successful frame
function resetReconnectState() {
	if (isReconnecting || reconnectAttempts > 0) {
		console.log('Stream reconnected successfully');
		hideConnectionError();
		hideReconnectBanner();
	}
	reconnectAttempts = 0;
	isReconnecting = false;
	if (reconnectTimeout) {
		clearTimeout(reconnectTimeout);
		reconnectTimeout = null;
	}
}

// Stream message handler as a named function for reuse
function handleStreamMessage(event) {
	// To hide the message (e.g., when you start drawing in WebGL again)
	messageDiv.style.display = 'none';

	const data = event.data;

	switch (data.type) {
		case 'update':
			// Reset reconnection state on successful frame
			resetReconnectState();
			updateConnectionStatus(ConnectionState.CONNECTED);

			// Handle the update
			const frameData = event.data.data;
			updateTexture(frameData, portrait, 1);
			break;
		case 'error':
			console.error('Error from worker:', event.data.message);

			// Check if this is a rate limit (429) error
			if (data.code === 'RATE_LIMITED') {
				updateConnectionStatus(ConnectionState.RATE_LIMITED);
				return;
			}

			// Check if this is a retryable error
			const isRetryable = data.retryable !== false;
			const severity = data.severity || 'error';

			if (severity === 'error' && isRetryable) {
				// Attempt automatic reconnection
				if (!isReconnecting) {
					attemptStreamReconnect();
				}
			} else if (severity === 'error') {
				// Non-retryable error
				updateConnectionStatus(ConnectionState.DISCONNECTED);
				showConnectionError(event.data.message, false);
			} else {
				// Warning or info level - just show message
				waiting(event.data.message);
			}
			break;
	}
}

// Initialize stream worker - reusable function for restart after Funnel toggle
function initStreamWorker() {
	// Attach handlers FIRST to catch any errors during init
	streamWorker.onmessage = handleStreamMessage;
	streamWorker.onerror = function(error) {
		console.error('Worker error:', error);
		if (!isReconnecting) {
			attemptStreamReconnect();
		}
	};
	streamWorker.postMessage({
		type: 'init',
		width: screenWidth,
		height: screenHeight,
		rate: rate,
		authToken: typeof getAuthToken === 'function' ? getAuthToken() : null,
	});
}

// Initialize on load
initStreamWorker();


// Determine the WebSocket protocol based on the current window protocol
const eventURL = `/events`;
// Send the OffscreenCanvas to the worker for initialization
eventWorker.postMessage({
	type: 'init',
	width: screenWidth,
	height: screenHeight,
	portrait: portrait,
	eventURL: eventURL,
    maxXValue: MaxXValue,
    maxYValue: MaxYValue,
	deviceModel: DeviceModel,
	authToken: typeof getAuthToken === 'function' ? getAuthToken() : null,
});
gestureWorker.postMessage({
	type: 'init',
	authToken: typeof getAuthToken === 'function' ? getAuthToken() : null,
});

gestureWorker.onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'gesture':

			switch (event.data.value) {
				case 'left':
					document.getElementById('content').contentWindow.postMessage( JSON.stringify({ method: 'left' }), '*' );
					break;
				case 'right':
					document.getElementById('content').contentWindow.postMessage( JSON.stringify({ method: 'right' }), '*' );
					break;
				case 'topleft-to-bottomright':
					document.getElementById('content').contentWindow.postMessage( JSON.stringify({ method: 'right' }), '*' );
					break;
				case 'topright-to-bottomleft':
					document.getElementById('content').contentWindow.postMessage( JSON.stringify({ method: 'left' }), '*' );
					break;
				case 'bottomright-to-topleft':
					iFrame.style.zIndex = 1;
					break;
				case 'bottomleft-to-topright':
					iFrame.style.zIndex = 4;
					break;
				default:
					// Code to execute if none of the above cases match
			}
			break;
		case 'error':
			console.error('Error from worker:', event.data.message);
			break;
	}

}

let messageTimeout;

function clearLaser() {
	// Function to call when no message is received for 300 ms
	updateLaserPosition(-10,-10);
}
// Event worker connection state
let eventWorkerConnected = false;

// Listen for updates from the worker
eventWorker.onmessage = (event) => {
	// Reset the timer every time a message is received
	clearTimeout(messageTimeout);
	messageTimeout = setTimeout(clearLaser, 300);

	// To hide the message (e.g., when you start drawing in WebGL again)
	messageDiv.style.display = 'none';

	const data = event.data;

	switch (data.type) {
		case 'connected':
			// SSE connection established
			eventWorkerConnected = true;
			console.log('Event worker connected');
			break;
		case 'clear':
			updateLaserPosition(-10,-10);
			//clearLaser();
			break;
		case 'update':
			// Mark as connected on first update if not already
			if (!eventWorkerConnected) {
				eventWorkerConnected = true;
			}
			// Handle the update
			const X = event.data.X;
			const Y = event.data.Y;
			updateLaserPosition(X,Y);
			break;
		case 'error':
			console.error('Error from event worker:', event.data.message);
			eventWorkerConnected = false;
			// Don't show UI errors for event worker - stream worker handles primary status
			// Just log to console
			break;
	}
};
