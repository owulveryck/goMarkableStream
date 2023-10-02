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

function renderCanvas(sourceCanvas, destCanvas) {
	var ctx = destCanvas.getContext("2d");
	if (rotate) {
		// clear the canvas
		ctx.clearRect(0, 0, destCanvas.width, destCanvas.height);
		ctx.save(); // Save the current state
		// Calculate the destination coordinates for drawing the rotated image
		var destX = (destCanvas.width - sourceCanvas.height) / 2;
		var destY = (destCanvas.height - sourceCanvas.width) / 2;
		var destWidth = sourceCanvas.height;
		var destHeight = sourceCanvas.width;

		// Move the rotation point to the center of the rectangle
		ctx.translate(destCanvas.width / 2, destCanvas.height / 2);

		// Rotate the canvas
		ctx.rotate(Math.PI / 180 * 270); // Rotate 45 degrees

		ctx.translate(-destCanvas.width / 2, -destCanvas.height / 2);

		// Draw the image on the second canvas
		ctx.drawImage(sourceCanvas, 0, 0, sourceCanvas.width, sourceCanvas.height, destX, destY, destWidth, destHeight);

		ctx.restore(); // Restore the state
		//resizeCanvas(dstCanvas);
		return;

	}
	ctx.drawImage(sourceCanvas, 0, 0, destCanvas.width, destCanvas.height);
}

