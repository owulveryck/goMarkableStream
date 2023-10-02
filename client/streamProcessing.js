function unpackValues(packedValue) {
	// Extract the upper 4 bits as the first value
	const value1 = (packedValue >> 4) & 0x0F;

	// Extract the lower 4 bits as the second value
	const value2 = packedValue & 0x0F;

	return [value1+1, value2];
}
function waiting(message) {
	var fontSize = 48;
	var fontFamily = "Arial";
	var textColor = "red";

	// Calculate the text dimensions
	resizableContext.font = fontSize + "px " + fontFamily;
	var textWidth = resizableContext.measureText(message).width;
	var textHeight = fontSize;

	// Calculate the center position
	var centerX = canvas.width / 2;
	var centerY = canvas.height / 2;

	// Set the fill style and align the text in the center
	resizableContext.fillStyle = textColor;
	resizableContext.textAlign = "center";
	resizableContext.textBaseline = "middle";

	// Draw the text at the center
	resizableContext.fillText(message, centerX, centerY);
}


async function initiateStream() {
	const RETRY_DELAY_MS = 3000; // Delay before retrying the connection (in milliseconds)

	try {

		// Create a new ReadableStream instance from a fetch request
		const response = await fetch('/stream');
		const stream = response.body;

		// Create a reader for the ReadableStream
		const reader = stream.getReader();
		// Create an ImageData object with the byte array length
		var imageData = fixedContext.createImageData(fixedCanvas.width, fixedCanvas.height);


		var offset = 0;
		var count = 0;
		var value = 0;


		// Define a function to process the chunks of data as they arrive
		const processData = async ({ done, value }) => {
			try {
				if (done) {
					console.log('Stream has ended');
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
								case 10: // red
									imageData.data[offset] = 255;
									imageData.data[offset+1] = 0;
									imageData.data[offset+2] = 0;
									imageData.data[offset+3] = 255;
									break;
								case 18: // blue
									imageData.data[offset] = 0;
									imageData.data[offset+1] = 0;
									imageData.data[offset+2] = 255;
									imageData.data[offset+3] = 255;
									break;
								case 20: // green
									imageData.data[offset] = 125;
									imageData.data[offset+1] = 184;
									imageData.data[offset+2] = 86;
									imageData.data[offset+3] = 255;
									break;
								case 24: // yellow
									imageData.data[offset] = 255;
									imageData.data[offset+1] = 253;
									imageData.data[offset+2] = 84;
									imageData.data[offset+3] = 255;
									break;
								default:
									imageData.data[offset] = value * 10;
									imageData.data[offset+1] = value * 10;
									imageData.data[offset+2] = value * 10;
									imageData.data[offset+3] = 255;
									break;
							}
						} else {
							imageData.data[offset] = value * 10;
							imageData.data[offset+1] = value * 10;
							imageData.data[offset+2] = value * 10;
							imageData.data[offset+3] = 255;
						}
					}
					// value is treated, wait for a count
					count = 0;
					if (offset >= fixedCanvas.height*fixedCanvas.width*4) {

						offset = 0;
						// Display the ImageData on the canvas
						fixedContext.putImageData(imageData, 0, 0);

						copyCanvasContent();
					}

				}

				// Read the next chunk
				const nextChunk = await reader.read();
				processData(nextChunk);
			} catch (error) {
				console.error('Error:', error);
				// Handle the error and determine if a reconnection should be attempted
				// For example, you can check the error message or status code to decide

				// Retry the connection after the delay
				waiting("reMarkable disconnected, please refresh");
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
		waiting("reMarkable disconnected, please refresh");
	}
}

