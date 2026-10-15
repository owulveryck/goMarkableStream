let height;
let width;
let rate;
let useRLE;
let bytesPerPixel = 4;
let withColor = true;
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
const FRAME_TYPE_DELTA_ZSTD = 0x04;  // Zstd-compressed delta frame

// Import fzstd for zstd decompression (vendored locally for offline use)
importScripts('/lib/fzstd.min.js');

onmessage = (event) => {
	const data = event.data;

	switch (data.type) {
		case 'init':
			height = event.data.height;
			width = event.data.width;
			rate = event.data.rate;
			useRLE = event.data.useRLE;
			bytesPerPixel = event.data.bytesPerPixel || 4;
			withColor = event.data.withColor;
			authToken = event.data.authToken || null;
			initiateStream();
			break;
		case 'withColorChanged':
			withColor = event.data.withColor;
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

		let offset = 0;
		let count = 0;

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

				if (useRLE) {
					({ offset, count } = decodeRLE(imageData, uint8Array, offset, count, withColor, pixelDataSize));
				} else if (bytesPerPixel === 2) {
					offset = decodeRGB565(imageData, uint8Array, offset, pixelDataSize);
				} else {
					await processDeltaData(uint8Array, imageData, pixelDataSize);
				}

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
		} else if (frameType === FRAME_TYPE_DELTA_ZSTD) {
			let runs;
			try {
				runs = fzstd.decompress(payload);
			} catch (err) {
				console.error('Zstd delta decompression failed:', err);
				continue;
			}
			handleDeltaFrame(runs, imageData, pixelDataSize);
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

// Accumulation buffer for RGB565 input (2 bytes per pixel).
// Filled across chunks; decoded to RGBA when a full frame is received.
let rgb565Buf = null;
let rgb565Pos = 0;

function decodeRGB565(imageData, chunkData, offset, pixelDataSize) {
    const frameSizeRGB565 = pixelDataSize / 2; // 2 bytes per pixel in, 4 bytes per pixel out

    if (!rgb565Buf) {
        rgb565Buf = new Uint8Array(frameSizeRGB565);
    }

    let start = 0;
    while (start < chunkData.length) {
        const remaining = frameSizeRGB565 - rgb565Pos;
        const toCopy = Math.min(chunkData.length - start, remaining);
        rgb565Buf.set(chunkData.subarray(start, start + toCopy), rgb565Pos);
        rgb565Pos += toCopy;
        start += toCopy;

        if (rgb565Pos >= frameSizeRGB565) {
            // Decode complete RGB565 frame to RGBA
            for (let i = 0; i < frameSizeRGB565; i += 2) {
                const lo = rgb565Buf[i];
                const hi = rgb565Buf[i + 1];
                const pixIdx = (i / 2) * 4;

                // RGB565 little-endian: lo = GGGBBBBB, hi = RRRRRGGG
                const r5 = (hi >> 3) & 0x1F;
                const g6 = ((hi & 0x07) << 3) | ((lo >> 5) & 0x07);
                const b5 = lo & 0x1F;

                // Expand to 8-bit per channel
                imageData[pixIdx]     = (r5 << 3) | (r5 >> 2);
                imageData[pixIdx + 1] = (g6 << 2) | (g6 >> 4);
                imageData[pixIdx + 2] = (b5 << 3) | (b5 >> 2);
                imageData[pixIdx + 3] = 255;
            }
            postMessage({ type: 'update', data: imageData });
            rgb565Pos = 0;
        }
    }

    return offset; // offset not used for RGB565 (rgb565Pos tracks state instead)
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

		// Copy changed pixels to previousFrame (bulk native memcpy)
		previousFrame.set(payload.subarray(dataStart, dataStart + dataLen), frameOffset);

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
			// This byte represents how many times the next value will be repeated
			count = chunkData[i];
			continue;
		}

		const value = chunkData[i];
		for (let c = 0; c < count; c++) {
			if (offset >= pixelDataSize) {
				break;
			}
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

			offset += 4;
			if (offset >= pixelDataSize) {
				postMessage({ type: 'update', data: imageData });
				offset = 0;
			}
		}

		// Reset count after processing this run
		count = 0;
	}

	return { offset, count };
}

