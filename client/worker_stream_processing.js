let withColor=true;
let height;
let width;
let rate;
let useRLE;

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			withColor = event.data.withColor;
			rate = event.data.rate;
			useRLE = event.data.useRLE;
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
		const response = await fetch('/stream?rate='+rate);
		const stream = response.body;

		// Create a reader for the ReadableStream
		const reader = stream.getReader();
		// Create an ImageData object with the byte array length
		const pixelDataSize = width * height * 4;
		const imageData = new Uint8ClampedArray(pixelDataSize);

		var offset = 0;
		var count = 0;

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

                const uint8Array = new Uint8Array(value);

				// Process the received data chunk and render if needed.
                if (useRLE) {
                    ({ offset, count } = decodeRLE(imageData, uint8Array, offset, count, withColor, pixelDataSize));
                } else {
                    offset = decodeRaw(imageData, uint8Array, offset, pixelDataSize);
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


function decodeRLE(imageData, chunkData, offset, count, withColor, pixelDataSize) {
	for (let i = 0; i < chunkData.length; i++) {
		if (count === 0) {
			// This byte represents how many times the next value will be repeated
			count = chunkData[i];
			continue;
		}

		const value = chunkData[i];
		for (let c = 0; c < count; c++) {
			offset += 4;
			if (withColor) {
				switch (value) {
					case 30: // Transparent
						imageData[offset+3] = 0;
						break;
					case 6: // Red
						imageData[offset] = 255;
						imageData[offset+1] = 0;
						imageData[offset+2] = 0;
						imageData[offset+3] = 255;
						break;
					case 8: // Red
						imageData[offset] = 255;
						imageData[offset+1] = 0;
						imageData[offset+2] = 0;
						imageData[offset+3] = 255;
						break;
					case 12: // Blue
						imageData[offset] = 0;
						imageData[offset+1] = 0;
						imageData[offset+2] = 255;
						imageData[offset+3] = 255;
						break;
					case 20: // Green
						imageData[offset] = 125;
						imageData[offset+1] = 184;
						imageData[offset+2] = 86;
						imageData[offset+3] = 255;
						break;
					case 24: // Yellow
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

			if (offset >= pixelDataSize) {
				break;
			}
		}

		// Reset count after processing this run
		count = 0;

		if (offset >= pixelDataSize) {
            // Send the frame
            postMessage({ type: 'update', data: imageData });

            // Reset for next frame
            offset = 0;
		}
	}

	return { offset, count };
}

function decodeRaw(imageData, chunkData, offset, pixelDataSize) {
    let start = 0;
    while (start < chunkData.length) {
        const bytesLeftInFrame = pixelDataSize - offset;
        const bytesToCopy = Math.min(chunkData.length - start, bytesLeftInFrame);
        imageData.set(chunkData.subarray(start, start + bytesToCopy), offset);

        offset += bytesToCopy;
        start += bytesToCopy;

        // If we've completed a full frame
        if (offset >= pixelDataSize) {
            // Send the frame
            postMessage({ type: 'update', data: imageData });

            // Reset for next frame
            offset = 0;
        }
    }

    return offset;
}

function simpleSum(data) {
	return data.reduce((acc, val) => acc + val, 0);
}
