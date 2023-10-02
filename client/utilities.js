function copyCanvasContent(fixedCanvas) {
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
		return;

	}
	resizableContext.drawImage(fixedCanvas, 0, 0, resizableCanvas.width, resizableCanvas.height);
	// Draw the image from the first canvas onto the second canvas
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


