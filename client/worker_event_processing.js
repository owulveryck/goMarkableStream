let height;
let width;
let eventURL;
let portrait;
let draw; 
let latestX;
let latestY;
let maxXValue;
let maxYValue;

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
	const eventSource = new EventSource(eventURL);
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
			if (portrait) {
				if (message.Code === 1) { // Horizontal position
					latestX = scaleValue(message.Value, maxXValue, width);
				} else if (message.Code === 0) { // Vertical position
					latestY = height - scaleValue(message.Value, maxYValue, height);
				}
			} else {
				// wrong
				if (message.Code === 1) { // Horizontal position
					latestY = scaleValue(message.Value, maxYValue, height);
				} else if (message.Code === 0) { // Vertical position
					latestX = scaleValue(message.Value, maxXValue, width);
				}
			}
			if (draw) {
				postMessage({ type: 'update', X: latestX, Y: latestY });
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