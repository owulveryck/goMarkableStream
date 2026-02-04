let withColor = true;
let height;
let width;
let rate;
let useRLE;
let useDelta;

// Delta decoding state
let previousFrame = null;
let pendingBuffer = new Uint8Array(0);

// Frame type constants (must match server)
const FRAME_TYPE_FULL = 0x00;
const FRAME_TYPE_DELTA = 0x01;

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			withColor = event.data.withColor;
			rate = event.data.rate;
			useRLE = event.data.useRLE;
			useDelta = event.data.useDelta;
			initiateStream();
			break;
		case 'withColorChanged':
			withColor = event.data.withColor;
			break;
		case 'terminate':
			console.log("terminating worker");
			close();
			break;
	}
};

async function initiateStream() {
	try {
		const response = await fetch('/stream?rate=' + rate);
		const stream = response.body;
		const reader = stream.getReader();
		const pixelDataSize = width * height * 4;
		const imageData = new Uint8ClampedArray(pixelDataSize);

		// Initialize previous frame buffer for delta decoding
		if (useDelta) {
			previousFrame = new Uint8Array(pixelDataSize);
		}

		let offset = 0;
		let count = 0;

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

				if (useDelta) {
					processDeltaData(uint8Array, imageData, pixelDataSize);
				} else if (useRLE) {
					({ offset, count } = decodeRLE(imageData, uint8Array, offset, count, withColor, pixelDataSize));
				} else {
					offset = decodeRaw(imageData, uint8Array, offset, pixelDataSize);
				}

				const nextChunk = await reader.read();
				processData(nextChunk);
			} catch (error) {
				console.log(error);
				postMessage({
					type: 'error',
					message: error.message
				});
			}
		};

		const initialChunk = await reader.read();
		processData(initialChunk);
	} catch (error) {
		console.error('Error:', error);
		postMessage({
			type: 'error',
			message: error.message
		});
	}
}

// Process delta-encoded data with frame header parsing
function processDeltaData(chunkData, imageData, pixelDataSize) {
	// Append new data to pending buffer
	const newBuffer = new Uint8Array(pendingBuffer.length + chunkData.length);
	newBuffer.set(pendingBuffer);
	newBuffer.set(chunkData, pendingBuffer.length);
	pendingBuffer = newBuffer;

	// Process complete frames from buffer
	while (pendingBuffer.length >= 4) {
		// Read 4-byte header
		const frameType = pendingBuffer[0];
		const payloadLen = pendingBuffer[1] | (pendingBuffer[2] << 8) | (pendingBuffer[3] << 16);

		// Check if we have the complete frame
		if (pendingBuffer.length < 4 + payloadLen) {
			// Wait for more data
			return;
		}

		// Extract payload
		const payload = pendingBuffer.subarray(4, 4 + payloadLen);

		if (frameType === FRAME_TYPE_FULL) {
			handleFullFrame(payload, imageData, pixelDataSize);
		} else if (frameType === FRAME_TYPE_DELTA) {
			handleDeltaFrame(payload, imageData, pixelDataSize);
		}

		// Remove processed bytes from buffer
		pendingBuffer = pendingBuffer.slice(4 + payloadLen);
	}
}

// Handle full frame: copy to previousFrame and render
function handleFullFrame(payload, imageData, pixelDataSize) {
	if (payload.length !== pixelDataSize) {
		console.error('Full frame size mismatch:', payload.length, 'expected:', pixelDataSize);
		return;
	}

	// Store as previous frame
	previousFrame.set(payload);

	// Copy to imageData for rendering
	imageData.set(payload);

	// Send frame update
	postMessage({ type: 'update', data: imageData });
}

// Handle delta frame: apply runs to previousFrame and render
function handleDeltaFrame(payload, imageData, pixelDataSize) {
	let pos = 0;
	let frameOffset = 0; // Current position in the frame

	while (pos < payload.length) {
		// Read length byte
		const lengthByte = payload[pos];
		let runLength;
		let relativeOffset;
		let dataStart;

		if ((lengthByte & 0x80) === 0) {
			// Short run: [1 byte length] [2 bytes offset LE] [data]
			if (pos + 3 > payload.length) break;

			runLength = lengthByte;
			relativeOffset = payload[pos + 1] | (payload[pos + 2] << 8);
			dataStart = pos + 3;
			pos = dataStart + runLength * 4;
		} else {
			// Long run: [1 byte 0x80|len_high] [1 byte len_low] [3 bytes offset LE] [data]
			if (pos + 5 > payload.length) break;

			runLength = ((lengthByte & 0x7F) << 8) | payload[pos + 1];
			relativeOffset = payload[pos + 2] | (payload[pos + 3] << 8) | (payload[pos + 4] << 16);
			dataStart = pos + 5;
			pos = dataStart + runLength * 4;
		}

		// Validate we have enough data
		if (pos > payload.length) {
			console.error('Delta frame truncated');
			break;
		}

		// Apply the run: advance by offset, then copy pixels
		frameOffset += relativeOffset;
		const dataLen = runLength * 4;

		if (frameOffset + dataLen > pixelDataSize) {
			console.error('Delta run exceeds frame bounds');
			break;
		}

		// Copy changed pixels to previousFrame
		for (let i = 0; i < dataLen; i++) {
			previousFrame[frameOffset + i] = payload[dataStart + i];
		}

		frameOffset += dataLen;
	}

	// Copy previousFrame to imageData for rendering
	imageData.set(previousFrame);

	// Send frame update
	postMessage({ type: 'update', data: imageData });
}

function decodeRLE(imageData, chunkData, offset, count, withColor, pixelDataSize) {
	for (let i = 0; i < chunkData.length; i++) {
		if (count === 0) {
			count = chunkData[i];
			continue;
		}

		const value = chunkData[i];
		for (let c = 0; c < count; c++) {
			offset += 4;
			if (withColor) {
				switch (value) {
					case 30:
						imageData[offset + 3] = 0;
						break;
					case 6:
					case 8:
						imageData[offset] = 255;
						imageData[offset + 1] = 0;
						imageData[offset + 2] = 0;
						imageData[offset + 3] = 255;
						break;
					case 12:
						imageData[offset] = 0;
						imageData[offset + 1] = 0;
						imageData[offset + 2] = 255;
						imageData[offset + 3] = 255;
						break;
					case 20:
						imageData[offset] = 125;
						imageData[offset + 1] = 184;
						imageData[offset + 2] = 86;
						imageData[offset + 3] = 255;
						break;
					case 24:
						imageData[offset] = 255;
						imageData[offset + 1] = 253;
						imageData[offset + 2] = 84;
						imageData[offset + 3] = 255;
						break;
					default:
						imageData[offset] = value * 10;
						imageData[offset + 1] = value * 10;
						imageData[offset + 2] = value * 10;
						imageData[offset + 3] = 255;
						break;
				}
			} else {
				if (value === 30) {
					imageData[offset + 3] = 0;
				} else {
					imageData[offset] = value * 10;
					imageData[offset + 1] = value * 10;
					imageData[offset + 2] = value * 10;
					imageData[offset + 3] = 255;
				}
			}

			if (offset >= pixelDataSize) {
				break;
			}
		}

		count = 0;

		if (offset >= pixelDataSize) {
			postMessage({ type: 'update', data: imageData });
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

		if (offset >= pixelDataSize) {
			postMessage({ type: 'update', data: imageData });
			offset = 0;
		}
	}

	return offset;
}

function simpleSum(data) {
	return data.reduce((acc, val) => acc + val, 0);
}
