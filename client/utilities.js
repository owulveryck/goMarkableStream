screenshotButton.addEventListener("click", function() {
	const tempCanvas = createTempCanvas(); // Create the temporary canvas
    var screenshotDataUrl = tempCanvas.toDataURL("image/png");
    downloadScreenshot(screenshotDataUrl);
	removeTempCanvas();
});

function downloadScreenshot(dataUrl) {
	var link = document.getElementById("screenshot");
	//var link = document.createElement("a");
	link.href = dataUrl;
	link.download = "reMarkable.png";
	link.click();
}


