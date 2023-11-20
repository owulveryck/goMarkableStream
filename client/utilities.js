
function downloadScreenshot(dataUrl) {
	var link = document.getElementById("screenshot");
	//var link = document.createElement("a");
	link.href = dataUrl;
	link.download = "reMarkable.png";
	link.click();
}


