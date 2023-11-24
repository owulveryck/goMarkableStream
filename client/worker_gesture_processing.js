let wsURL;
// Constants for the maximum values from the WebSocket messages
const SWIPE_DISTANCE = 200;

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			wsURL = event.data.wsURL;
			fetchStream();
			break;
		case 'terminate':
			console.log("terminating worker");
			close();
			break;
	}
};

async function fetchStream() {
	const response = await fetch('/gestures');

	const reader = response.body.getReader();
	const decoder = new TextDecoder('utf-8');
	let buffer = '';

	while (true) {
		const { value, done } = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, { stream: true });

		while (buffer.includes('\n')) {
			const index = buffer.indexOf('\n');
			const jsonStr = buffer.slice(0, index);
			buffer = buffer.slice(index + 1);

			try {
				const json = JSON.parse(jsonStr);
				let swipe = checkSwipeDirection(json);
				if (swipe != 'none') {
					postMessage({ type: 'gesture', value: swipe}) ;
				}
			} catch (e) {
				console.error('Error parsing JSON:', e);
			}
		}
	}
}


function checkSwipeDirection(json) {
	if (json.left > 200 && json.right < 75 && json.up < 100 && json.down < 100) {
		return 'left';
	} else if (json.right > 200 && json.left < 75 && json.up < 100 && json.down < 100) {
		return 'right';
	} else if (json.up > 200 && json.right < 100 && json.left < 100 && json.down < 75) {
		return 'up';
	} else if (json.down > 200 && json.right < 100 && json.up < 75 && json.left < 100) {
		return 'down';
	} else {
		return 'none';
	}
}

