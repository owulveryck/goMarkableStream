function downloadScreenshot(dataUrl) {
	// Use 'toDataURL' to capture the current canvas content
	// Create an 'a' element for downloading
	var link = document.getElementById("screenshot");

	link.download = 'goMarkableScreenshot.png';
	link.href = dataURL;
	link.click();
}


