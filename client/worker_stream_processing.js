let height;
let width;
let rate;
let useRLE;
let bytesPerPixel = 4;
let withColor = true;
let authToken = null;

// Delta decoding state
let previousFrame = null;
let activeImageData = null;
let activeImageData32 = null;
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

// Endian-safe RGBA -> Uint32 packer for single-store 32-bit pixel writes
function packRGBA32(r, g, b, a) {
	const tmp = new Uint8Array([r, g, b, a]);
	return new Uint32Array(tmp.buffer)[0];
}

// Precomputed 256-entry 32-bit RGBA lookup tables for 1-byte Color8 (RM1 default)
// and legacy RLE modes. In normal viewing mode, white paper (code 30) is fully opaque
// [255, 255, 255, 255] (matching RM2 3.24+ and Paper Pro), ensuring crisp contrast
// and preventing texture-filtering washout on thin glyph stems, ruled notebook lines,
// and anti-aliased strokes. In presentation overlay mode (?present=...), paper uses
// [255, 255, 255, 0] so slides beneath the canvas remain visible.
const color8LUT32 = new Uint32Array(256);
const gray8LUT32 = new Uint32Array(256);
let isOverlayMode = false;

function initColor8LUTs(overlay) {
	isOverlayMode = !!overlay;
	const paper = isOverlayMode ? packRGBA32(255, 255, 255, 0) : packRGBA32(255, 255, 255, 255);
	for (let v = 0; v < 256; v++) {
		if (v === 30) {
			color8LUT32[v] = paper;
			gray8LUT32[v] = paper;
			continue;
		}
		const g = Math.min(255, v * 10);
		const gray = packRGBA32(g, g, g, 255);
		gray8LUT32[v] = gray;
		switch (v) {
			case 6:
			case 8:
				color8LUT32[v] = packRGBA32(255, 0, 0, 255); // Red
				break;
			case 12:
				color8LUT32[v] = packRGBA32(0, 0, 255, 255); // Blue
				break;
			case 20:
				color8LUT32[v] = packRGBA32(125, 184, 86, 255); // Green
				break;
			case 24:
				color8LUT32[v] = packRGBA32(255, 253, 84, 255); // Yellow
				break;
			default:
				color8LUT32[v] = gray;
				break;
		}
	}
}
initColor8LUTs(false);

// Lazily initialized 65,536-entry 32-bit RGBA LUT for optional 16-bit RGB565 mode
let rgb565LUT32 = null;
let rgb565IsOverlay = null;
function getRGB565LUT32(overlay) {
	const ov = overlay !== undefined ? !!overlay : isOverlayMode;
	if (rgb565LUT32 && rgb565IsOverlay === ov) return rgb565LUT32;
	rgb565IsOverlay = ov;
	rgb565LUT32 = new Uint32Array(65536);
	const paper = ov ? packRGBA32(255, 255, 255, 0) : packRGBA32(255, 255, 255, 255);
	for (let px = 0; px < 65536; px++) {
		const lo = px & 0xFF;
		const hi = (px >> 8) & 0xFF;
		const r8 = (hi & 0xF8) | (hi >> 5);
		const g6 = ((hi & 0x07) << 3) | (lo >> 5);
		const g8 = (g6 << 2) | (g6 >> 4);
		const b5 = lo & 0x1F;
		const b8 = (b5 << 3) | (b5 >> 2);
		if (r8 + g8 + b8 > 737) {
			rgb565LUT32[px] = paper;
		} else {
			rgb565LUT32[px] = packRGBA32(r8, g8, b8, 255);
		}
	}
	return rgb565LUT32;
}

function renderColor8Full(src, dst32) {
	const lut = withColor ? color8LUT32 : gray8LUT32;
	const n = dst32.length;
	for (let i = 0; i < n; i++) {
		dst32[i] = lut[src[i]];
	}
}

function renderRGB565Full(src, dst32) {
	const lut = getRGB565LUT32();
	const n = dst32.length;
	for (let i = 0; i < n; i++) {
		const b = i << 1;
		dst32[i] = lut[src[b] | (src[b + 1] << 8)];
	}
}

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
			initColor8LUTs(event.data.isOverlay || false);
			initiateStream();
			break;
		case 'withColorChanged':
			withColor = event.data.withColor;
			if (!useRLE && bytesPerPixel === 1 && previousFrame && activeImageData && activeImageData32) {
				renderColor8Full(previousFrame, activeImageData32);
				postMessage({ type: 'update', data: activeImageData });
			}
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
		const numPixels = width * height;
		const pixelDataSize = numPixels * 4;
		const wireFrameSize = numPixels * bytesPerPixel;
		const imageData = new Uint8ClampedArray(pixelDataSize);
		const imageData32 = new Uint32Array(imageData.buffer);

		activeImageData = imageData;
		activeImageData32 = imageData32;

		// Initialize previous frame buffer in wire format (1 B/px Color8, 2 B/px RGB565, or 4 B/px BGRA)
		previousFrame = new Uint8Array(wireFrameSize);
		pendingBuffer = new Uint8Array(0);

		let offset = 0;
		let count = 0;

		const processData = async ({ done, value }) => {
			try {
				if (done) {
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
					({ offset, count } = decodeRLE(imageData, imageData32, uint8Array, offset, count, withColor, numPixels));
				} else {
					await processDeltaData(uint8Array, imageData, imageData32, wireFrameSize);
				}

				const nextChunk = await reader.read();
				processData(nextChunk);
			} catch (error) {
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
async function processDeltaData(chunkData, imageData, imageData32, wireFrameSize) {
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
			return;
		}

		// Extract payload (make a copy since we'll modify pendingBuffer)
		const payload = pendingBuffer.slice(4, 4 + payloadLen);

		// Remove processed bytes from buffer before async operations
		pendingBuffer = pendingBuffer.slice(4 + payloadLen);

		if (frameType === FRAME_TYPE_FULL) {
			await handleFullFrame(payload, imageData, imageData32, wireFrameSize, 'none');
		} else if (frameType === FRAME_TYPE_FULL_COMPRESSED) {
			await handleFullFrame(payload, imageData, imageData32, wireFrameSize, 'gzip');
		} else if (frameType === FRAME_TYPE_FULL_ZSTD) {
			await handleFullFrame(payload, imageData, imageData32, wireFrameSize, 'zstd');
		} else if (frameType === FRAME_TYPE_DELTA) {
			handleDeltaFrame(payload, imageData, imageData32, wireFrameSize);
		} else if (frameType === FRAME_TYPE_DELTA_ZSTD) {
			let runs;
			try {
				runs = fzstd.decompress(payload);
			} catch (err) {
				console.error('Zstd delta decompression failed:', err);
				continue;
			}
			handleDeltaFrame(runs, imageData, imageData32, wireFrameSize);
		}
	}
}

// Handle full frame: decompress if needed, copy to previousFrame and render
async function handleFullFrame(payload, imageData, imageData32, wireFrameSize, compressionType) {
	let frameData = payload;

	if (compressionType === 'zstd') {
		try {
			frameData = fzstd.decompress(payload);
		} catch (err) {
			console.error('Zstd decompression failed:', err);
			return;
		}
	} else if (compressionType === 'gzip') {
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

		const totalLen = chunks.reduce((acc, c) => acc + c.length, 0);
		frameData = new Uint8Array(totalLen);
		let offset = 0;
		for (const chunk of chunks) {
			frameData.set(chunk, offset);
			offset += chunk.length;
		}
	}

	if (frameData.length !== wireFrameSize) {
		console.error('Full frame size mismatch:', frameData.length, 'expected:', wireFrameSize);
		return;
	}

	// Store wire-format previous frame
	previousFrame.set(frameData);

	// Convert or copy into RGBA imageData
	if (bytesPerPixel === 1) {
		renderColor8Full(frameData, imageData32);
	} else if (bytesPerPixel === 2) {
		renderRGB565Full(frameData, imageData32);
	} else {
		imageData.set(frameData);
	}

	postMessage({ type: 'update', data: imageData });
}

// Handle delta frame: apply runs to previousFrame and render ONLY changed spans for 1/2 bpp
function handleDeltaFrame(payload, imageData, imageData32, wireFrameSize) {
	let pos = 0;
	let frameOffset = 0; // Current byte position in previousFrame

	const colorLUT = bytesPerPixel === 1 ? (withColor ? color8LUT32 : gray8LUT32) : null;
	const rgbLUT = bytesPerPixel === 2 ? getRGB565LUT32() : null;

	while (pos < payload.length) {
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
		} else {
			// Long run: [1 byte 0x80|len_high] [1 byte len_low] [3 bytes offset LE] [data]
			if (pos + 5 > payload.length) break;

			runLength = ((lengthByte & 0x7F) << 8) | payload[pos + 1];
			relativeOffset = payload[pos + 2] | (payload[pos + 3] << 8) | (payload[pos + 4] << 16);
			dataStart = pos + 5;
		}

		const dataLen = runLength * bytesPerPixel;
		pos = dataStart + dataLen;

		if (pos > payload.length) {
			console.error('Delta frame truncated');
			break;
		}

		frameOffset += relativeOffset;
		if (frameOffset + dataLen > wireFrameSize) {
			console.error('Delta run exceeds frame bounds');
			break;
		}

		// Update wire-format previousFrame
		previousFrame.set(payload.subarray(dataStart, pos), frameOffset);

		// For Color8 (1 B/px) and RGB565 (2 B/px), expand ONLY the changed run
		// directly into imageData32 (O(changed pixels) instead of O(screen pixels)).
		if (bytesPerPixel === 1) {
			for (let i = 0; i < runLength; i++) {
				imageData32[frameOffset + i] = colorLUT[payload[dataStart + i]];
			}
		} else if (bytesPerPixel === 2) {
			const pixBase = frameOffset >> 1;
			for (let i = 0; i < runLength; i++) {
				const src = dataStart + (i << 1);
				imageData32[pixBase + i] = rgbLUT[payload[src] | (payload[src + 1] << 8)];
			}
		}

		frameOffset += dataLen;
	}

	if (bytesPerPixel === 4) {
		imageData.set(previousFrame);
	}

	postMessage({ type: 'update', data: imageData });
}

function decodeRLE(imageData, imageData32, chunkData, pixOffset, count, withColor, numPixels) {
	const lut = withColor ? color8LUT32 : gray8LUT32;

	for (let i = 0; i < chunkData.length; i++) {
		if (count === 0) {
			count = chunkData[i];
			continue;
		}

		const rgba32 = lut[chunkData[i]];
		for (let c = 0; c < count; c++) {
			if (pixOffset >= numPixels) {
				break;
			}
			imageData32[pixOffset++] = rgba32;
			if (pixOffset >= numPixels) {
				postMessage({ type: 'update', data: imageData });
				pixOffset = 0;
			}
		}

		count = 0;
	}

	return { offset: pixOffset, count };
}
