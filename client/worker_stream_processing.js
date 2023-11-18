let withColor=true;
let height;
let width;

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			initiateStream();
			break;
		case 'withColorChanged':
			withColor = event.data.withColor;
			// Handle the error, maybe show a user-friendly message or take some corrective action
			break;
		case 'terminate':
			console.log("terminating worker");
			close();
			break;

	}
};


async function initiateStream() {
	const RETRY_DELAY_MS = 3000; // Delay before retrying the connection (in milliseconds)

	try {

		// Create a new ReadableStream instance from a fetch request
		const response = await fetch('/stream');
		const stream = response.body;

		// Create a reader for the ReadableStream
		const reader = stream.getReader();
		// Create an ImageData object with the byte array length
		const pixelDataSize = width * height * 4;
		const imageData = new Uint8ClampedArray(pixelDataSize);



		var offset = 0;
		var count = 0;
		var lastSum = 0;


		// Define a function to process the chunks of data as they arrive
		const processData = async ({ done, value }) => {
			try {
				if (done) {
					postMessage({
						type: 'error',
						message: "end of transmission"
					});
					return;
				}

				// Process the received data chunk
				// Assuming each pixel is represented by 4 bytes (RGBA)
				var uint8Array = new Uint8Array(value);

				for (let i = 0; i < uint8Array.length; i++) {
					// if no count, then it is a count
					if (count === 0) {
						count = uint8Array[i];
						continue;
					}
					// if we have a count, it is a value...
						const value = uint8Array[i];
					for (let c=0;c<count;c++) {
						offset += 4;
						if (withColor) {
							switch (value) {
								case 30: // red
									imageData[offset+3] = 0;
									break;
								case 10: // red
									imageData[offset] = 255;
									imageData[offset+1] = 0;
									imageData[offset+2] = 0;
									imageData[offset+3] = 255;
									break;
								case 18: // blue
									imageData[offset] = 0;
									imageData[offset+1] = 0;
									imageData[offset+2] = 255;
									imageData[offset+3] = 255;
									break;
								case 20: // green
									imageData[offset] = 125;
									imageData[offset+1] = 184;
									imageData[offset+2] = 86;
									imageData[offset+3] = 255;
									break;
								case 24: // yellow
									imageData[offset] = 255;
									imageData[offset+1] = 253;
									imageData[offset+2] = 84;
									imageData[offset+3] = 255;
									break;
								default:
									imageData[offset] = value * 10;
									imageData[offset+1] = value * 10;
									imageData[offset+2] = value * 10;
									imageData[offset+3] = 255;
									break;
							}
						} else {
							if (value === 30) {
								imageData[offset+3] = 0;
							} else {
								imageData[offset] = value * 10;
								imageData[offset+1] = value * 10;
								imageData[offset+2] = value * 10;
								imageData[offset+3] = 255;
							}
						}
					}
					// value is treated, wait for a count
					count = 0;
					if (offset >= height*width*4) {
						offset = 0;
						// Later, check if the sum has changed
						//const currentSum = simpleSum(imageData);
						//if (currentSum !== lastSum) {
							// The sum has changed, execute your desired action

							// Instead of calling copyCanvasContent(), send the OffscreenCanvas to the main thread
							postMessage({ type: 'update', data: imageData });
						//}
						//lastSum = currentSum;
					}

				}

				// Read the next chunk
				const nextChunk = await reader.read();
				processData(nextChunk);
			} catch (error) {
				console.log(error)
				postMessage({
					type: 'error',
					message: error.message
				});
			}

		};

		// Start reading the initial chunk of data
		const initialChunk = await reader.read();
		processData(initialChunk);
	} catch (error) {
		console.error('Error:', error);
		// Handle the error and determine if a reconnection should be attempted
		// For example, you can check the error message or status code to decide

		// Retry the connection after the delay
		postMessage({
			type: 'error',
			message: error.message
		});
	}
}

function simpleSum(data) {
	return data.reduce((acc, val) => acc + val, 0);
}
