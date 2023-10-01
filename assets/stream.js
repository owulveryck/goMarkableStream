let rotate = true;  // start with a false boolean
let withColor = true;  // start with a false boolean
document.getElementById('rotate').addEventListener('click', function() {
	rotate  = !rotate;  // toggle the boolean
	
    // Toggle the 'toggled' class on the button
    this.classList.toggle('toggled');

    resizeAndCopy();
});

document.getElementById('colors').addEventListener('click', function() {
	withColor  = !withColor;  // toggle the boolean
	
    // Toggle the 'toggled' class on the button
    this.classList.toggle('toggled');

    resizeAndCopy();
});

const sidebar = document.querySelector('.sidebar');

sidebar.addEventListener('mouseover', function() {
	sidebar.classList.add('active');
});

sidebar.addEventListener('mouseout', function() {
	sidebar.classList.remove('active');
});


// Use the fixed-size canvas context to draw on the canvas
var fixedCanvas = document.getElementById("fixedCanvas");
var fixedContext = fixedCanvas.getContext("2d");
var resizableCanvas = document.getElementById("canvas");
var resizableContext = resizableCanvas.getContext("2d");
// Set the fill color
resizableContext.fillStyle = '#666666'; 

// Fill the canvas with the specified color
resizableContext.fillRect(0, 0, resizableCanvas.width, resizableCanvas.height);
waiting("waiting for reMarkable");

function copyCanvasContent() {
	if (rotate) {
		// clear the canvas
		resizableContext.clearRect(0, 0, resizableCanvas.width, resizableCanvas.height);
		resizableContext.save(); // Save the current state
		// Calculate the destination coordinates for drawing the rotated image
		var destX = (resizableCanvas.width - fixedCanvas.height) / 2;
		var destY = (resizableCanvas.height - fixedCanvas.width) / 2;
		var destWidth = fixedCanvas.height;
		var destHeight = fixedCanvas.width;

		// Move the rotation point to the center of the rectangle
		resizableContext.translate(resizableCanvas.width / 2, resizableCanvas.height / 2);

		// Rotate the canvas
		resizableContext.rotate(Math.PI / 180 * 270); // Rotate 45 degrees

		resizableContext.translate(-resizableCanvas.width / 2, -resizableCanvas.height / 2);

		// Draw the image on the second canvas
		//resizableContext.drawImage(fixedCanvas, -fixedCanvas.width / 2, -fixedCanvas.height / 2);
		//					resizableContext.drawImage(fixedCanvas, 0,0, resizableCanvas.width, resizableCanvas.height);
		resizableContext.drawImage(fixedCanvas, 0, 0, fixedCanvas.width, fixedCanvas.height, destX, destY, destWidth, destHeight);

		resizableContext.restore(); // Restore the state
		resizeCanvas();
		return;

	}
	resizableContext.drawImage(fixedCanvas, 0, 0, resizableCanvas.width, resizableCanvas.height);
	// Draw the image from the first canvas onto the second canvas
}

// JavaScript code for working with the canvas element
function resizeCanvas() {
	var canvas = document.getElementById("canvas");
	var container = document.getElementById("container");

	if (rotate) {
		var aspectRatio = 1404 / 1872;
	} else {
		var aspectRatio = 1872 / 1404;
	}

	var containerWidth = container.offsetWidth;
	var containerHeight = container.offsetHeight;

	var containerAspectRatio = containerWidth / containerHeight;

	if (containerAspectRatio > aspectRatio) {
		canvas.style.width = containerHeight * aspectRatio + "px";
		canvas.style.height = containerHeight + "px";
	} else {
		canvas.style.width = containerWidth + "px";
		canvas.style.height = containerWidth / aspectRatio + "px";
	}
}

function resizeAndCopy() {
	resizeCanvas();
	copyCanvasContent();
}

// Resize the canvas whenever the window is resized
window.addEventListener("resize", resizeAndCopy);


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
screenshotButton.addEventListener("click", function() {
	var screenshotDataUrl = fixedCanvas.toDataURL("image/png");
	downloadScreenshot(screenshotDataUrl);
});

function downloadScreenshot(dataUrl) {
	var link = document.getElementById("screenshot");
	//var link = document.createElement("a");
	link.href = dataUrl;
	link.download = "reMarkable.png";
	link.click();
}

// JavaScript file (stream.js)

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

let mediaRecorder;
let recordedChunks = [];
function startRecording() {
	console.log("recording in progress");
	let stream = fixedCanvas.captureStream(25); // 25 fps

	mediaRecorder = new MediaRecorder(stream, {
		mimeType: 'video/webm;codecs=vp9'
	});

	mediaRecorder.ondataavailable = function(event) {
		if (event.data.size > 0) {
			recordedChunks.push(event.data);
		}
	};

	mediaRecorder.onstop = function() {
		download();
	};

	mediaRecorder.start();
}

function stopRecording() {
	mediaRecorder.stop();
}

function download() {
	let blob = new Blob(recordedChunks, {
		type: 'video/webm'
	});

	let url = URL.createObjectURL(blob);
	let a = document.createElement('a');
	a.style.display = 'none';
	a.href = url;
	a.download = 'goMarkableStreamRecording.webm';
	document.body.appendChild(a);
	a.click();
	setTimeout(() => {
		document.body.removeChild(a);
		window.URL.revokeObjectURL(url);
	}, 100);
}

document.getElementById('startStopButton').addEventListener('click', function() {
	let icon = document.getElementById('icon');
	let label = document.getElementById('label');


	if (label.textContent === 'Record') {
		label.textContent = 'Stop';
		icon.classList.add('recording');
		startRecording();
	} else {
		label.textContent = 'Record';
		icon.classList.remove('recording');
		stopRecording();
	}
});
initiateStream();


