let height;
let width;
let rate;
let authToken = null;

// Delta decoding state
let previousFrame = null;
let pendingBuffer = new Uint8Array(0);

// AbortController to cancel the stream fetch on terminate
let abortController = null;

// Frame type constants (must match server)
const FRAME_TYPE_FULL = 0x00;  // Deprecated: uncompressed full frame
const FRAME_TYPE_DELTA = 0x01;
const FRAME_TYPE_FULL_COMPRESSED = 0x02;  // Gzip-compressed full frame (legacy)
const FRAME_TYPE_FULL_ZSTD = 0x03;  // Zstd-compressed full frame

// Import fzstd for zstd decompression (vendored locally for offline use)
importScripts('/lib/fzstd.min.js');

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			rate = event.data.rate;
			authToken = event.data.authToken || null;
			initiateStream();
			break;
		case 'terminate':
			console.log("terminating worker");
			if (abortController) {
				abortController.abort();
			}
			close();
			break;
	}
};

async function initiateStream() {
	try {
		// Create AbortController to allow canceling the fetch
		abortController = new AbortController();

		// Build fetch options
		const fetchOptions = { signal: abortController.signal };
		if (authToken) {
			fetchOptions.headers = {
				'Authorization': `Bearer ${authToken}`
			};
		}

		const response = await fetch('/stream?rate=' + rate, fetchOptions);

		// Handle rate limiting (429)
		if (response.status === 429) {
			postMessage({
				type: 'error',
				severity: 'warning',
				code: 'RATE_LIMITED',
				message: 'Rate limited - too many clients',
				retryable: true
			});
			return;
		}

		// Handle other non-OK responses
		if (!response.ok) {
			postMessage({
				type: 'error',
				severity: 'error',
				code: 'HTTP_ERROR',
				message: `Server error: ${response.status}`,
				retryable: true
			});
			return;
		}

		const stream = response.body;
		const reader = stream.getReader();
		const pixelDataSize = width * height * 4;
		const imageData = new Uint8ClampedArray(pixelDataSize);

		// Initialize previous frame buffer for delta decoding
		previousFrame = new Uint8Array(pixelDataSize);

		const processData = async ({ done, value }) => {
			try {
				if (done) {
					// Only report error if not intentionally aborted
					if (!abortController.signal.aborted) {
						postMessage({
							type: 'error',
							severity: 'error',
							code: 'STREAM_ENDED',
							message: "Stream ended unexpectedly",
							retryable: true
						});
					}
					return;
				}

				const uint8Array = new Uint8Array(value);

				await processDeltaData(uint8Array, imageData, pixelDataSize);

				const nextChunk = await reader.read();
				processData(nextChunk);
			} catch (error) {
				// Don't report errors if intentionally aborted
				if (abortController && abortController.signal.aborted) {
					return;
				}
				console.log(error);
				postMessage({
					type: 'error',
					severity: 'error',
					code: 'STREAM_ERROR',
					message: error.message,
					retryable: true
				});
			}
		};

		const initialChunk = await reader.read();
		processData(initialChunk);
	} catch (error) {
		// Don't report errors if intentionally aborted
		if (abortController && abortController.signal.aborted) {
			return;
		}
		console.error('Error:', error);
		postMessage({
			type: 'error',
			severity: 'error',
			code: 'CONNECTION_ERROR',
			message: error.message,
			retryable: true
		});
	}
}

// Process delta-encoded data with frame header parsing
async function processDeltaData(chunkData, imageData, pixelDataSize) {
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

		// Extract payload (make a copy since we'll modify pendingBuffer)
		const payload = pendingBuffer.slice(4, 4 + payloadLen);

		// Remove processed bytes from buffer before async operations
		pendingBuffer = pendingBuffer.slice(4 + payloadLen);

		if (frameType === FRAME_TYPE_FULL) {
			await handleFullFrame(payload, imageData, pixelDataSize, 'none');
		} else if (frameType === FRAME_TYPE_FULL_COMPRESSED) {
			await handleFullFrame(payload, imageData, pixelDataSize, 'gzip');
		} else if (frameType === FRAME_TYPE_FULL_ZSTD) {
			await handleFullFrame(payload, imageData, pixelDataSize, 'zstd');
		} else if (frameType === FRAME_TYPE_DELTA) {
			handleDeltaFrame(payload, imageData, pixelDataSize);
		}
	}
}

// Handle full frame: decompress if needed, copy to previousFrame and render
async function handleFullFrame(payload, imageData, pixelDataSize, compressionType) {
	let frameData = payload;

	if (compressionType === 'zstd') {
		// Decompress using fzstd library
		try {
			frameData = fzstd.decompress(payload);
		} catch (err) {
			console.error('Zstd decompression failed:', err);
			return;
		}
	} else if (compressionType === 'gzip') {
		// Decompress using DecompressionStream API (legacy support)
		const ds = new DecompressionStream('gzip');
		const writer = ds.writable.getWriter();
		writer.write(payload);
		writer.close();

		const reader = ds.readable.getReader();
		const chunks = [];
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			chunks.push(value);
		}

		// Concatenate chunks
		const totalLen = chunks.reduce((acc, c) => acc + c.length, 0);
		frameData = new Uint8Array(totalLen);
		let offset = 0;
		for (const chunk of chunks) {
			frameData.set(chunk, offset);
			offset += chunk.length;
		}
	}

	if (frameData.length !== pixelDataSize) {
		console.error('Full frame size mismatch:', frameData.length, 'expected:', pixelDataSize);
		return;
	}

	// Store as previous frame
	previousFrame.set(frameData);

	// Copy to imageData for rendering
	imageData.set(frameData);

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

