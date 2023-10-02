var fixedCanvas = document.getElementById("fixedCanvas");
var fixedContext = fixedCanvas.getContext("2d");
var resizableCanvas = document.getElementById("canvas");
var resizableContext = resizableCanvas.getContext("2d");
resizableContext.fillStyle = '#666666'; 
resizableContext.fillRect(0, 0, resizableCanvas.width, resizableCanvas.height);

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
