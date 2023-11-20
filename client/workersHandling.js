// Send the OffscreenCanvas to the worker for initialization
streamWorker.postMessage({ 
	type: 'init', 
	width: width, 
	height: height 
});


// Listen for updates from the worker
streamWorker.onmessage = (event) => {
	// To hide the message (e.g., when you start drawing in WebGL again)
	messageDiv.style.display = 'none';

	const data = event.data;

	switch (data.type) {
		case 'update':
			// Handle the update
			const data = event.data.data;
			updateTexture(data, portrait, 1);
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
	portrait: portrait,
	wsURL: wsURL
});

// Listen for updates from the worker
eventWorker.onmessage = (event) => {
	// To hide the message (e.g., when you start drawing in WebGL again)
	messageDiv.style.display = 'none';

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
