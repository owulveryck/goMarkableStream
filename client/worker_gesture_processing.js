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
	if (json.left > 400 && json.right < 100 && json.up < 100 && json.down < 100) {
		return 'left';
	} else if (json.right > 400 && json.left < 100 && json.up < 100 && json.down < 100) {
		return 'right';
	} else if (json.up > 400 && json.right < 100 && json.left < 100 && json.down < 100) {
		return 'up';
	} else if (json.down > 400 && json.right < 100 && json.up < 100 && json.left < 100) {
		return 'down';
	} else if (json.right > 600 && json.down > 600 && json.up < 50 && json.left < 50 ) {
		return 'topright-to-bottomleft'
	} else if (json.left > 600 && json.down > 600 && json.up < 50 && json.right < 50 ) {
		return 'topleft-to-bottomright'
	} else if (json.left > 600 && json.up > 600 && json.down < 50 && json.right < 50 ) {
		return 'bottomleft-to-topright'
	} else if (json.right > 600 && json.up > 600 && json.down < 50 && json.left < 50 ) {
		return 'bottomright-to-topleft'
	} else {
		return 'none';
	}
}
