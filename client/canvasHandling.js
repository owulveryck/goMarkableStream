// JavaScript code for working with the canvas element
function resizeVisibleCanvas() {
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
		visibleCanvas.style.width = containerHeight * aspectRatio + "px";
		visibleCanvas.style.height = containerHeight + "px";
	} else {
		visibleCanvas.style.width = containerWidth + "px";
		visibleCanvas.style.height = containerWidth / aspectRatio + "px";
	}
	renderCanvas(rawCanvas,visibleCanvas);
}
function waiting(message) {
	var ctx = visibleCanvas.getContext("2d");
	ctx.fillStyle = '#666666'; 
	ctx.fillRect(0, 0, visibleCanvas.width, visibleCanvas.height);

	var fontSize = 48;
	var fontFamily = "Arial";
	var textColor = "red";

	// Calculate the text dimensions
	ctx.font = fontSize + "px " + fontFamily;
	var textWidth = ctx.measureText(message).width;
	var textHeight = fontSize;

	// Calculate the center position
	var centerX = canvas.width / 2;
	var centerY = canvas.height / 2;

	// Set the fill style and align the text in the center
	ctx.fillStyle = textColor;
	ctx.textAlign = "center";
	ctx.textBaseline = "middle";

	// Draw the text at the center
	ctx.fillText(message, centerX, centerY);
}

function renderCanvas(srcCanvas, dstCanvas) {
	let ctxSrc = srcCanvas.getContext('2d');
	let ctxDst = dstCanvas.getContext('2d');

	let w = srcCanvas.width;
	let h = srcCanvas.height;

	// Clear the destination canvas
	ctxDst.clearRect(0, 0, w, h);
	ctxDst.imageSmoothingEnabled = true;


	if (rotate) {
		// Swap width and height for dstCanvas to accommodate rotated content
		dstCanvas.width = h;
		dstCanvas.height = w;
		ctxDst.translate(0,w);  // Move the drawing origin to the right side of dstCanvas
		ctxDst.rotate(-Math.PI / 2); // Rotate by 90 degrees


		// Since the source canvas is now rotated, width and height are swapped
		ctxDst.drawImage(srcCanvas, 0, 0);
	} else {
		dstCanvas.width = w;
		dstCanvas.height = h;
		ctxDst.drawImage(srcCanvas, 0, 0);
	}

	// Reset transformations for future calls
	ctxDst.setTransform(1, 0, 0, 1, 0, 0);
}

