let rotate = true;
let withColor = true;
let recordingWithSound = false;

document.getElementById('rotate').addEventListener('click', function() {
	rotate = !rotate;
	this.classList.toggle('toggled');
	resizeVisibleCanvas();
});
document.getElementById('pointerButton').addEventListener('click', function() {
	if (isWebSocketConnected(ws)) {
		stopWebSocket();
	} else {
		connectWebSocket();
	}
});


document.getElementById('colors').addEventListener('click', function() {
	withColor = !withColor;
	this.classList.toggle('toggled');
	worker.postMessage({ type: 'withColorChanged', withColor: withColor });
});

const sidebar = document.querySelector('.sidebar');
sidebar.addEventListener('mouseover', function() {
	sidebar.classList.add('active');
});
sidebar.addEventListener('mouseout', function() {
	sidebar.classList.remove('active');
});

// Resize the canvas whenever the window is resized
window.addEventListener("resize", resizeVisibleCanvas);
resizeVisibleCanvas();

document.getElementById('switchOrderButton').addEventListener('click', function() {
	// Swap z-index values
	var temp = iFrame.style.zIndex;
	iFrame.style.zIndex = visibleCanvas.style.zIndex;
	visibleCanvas.style.zIndex = temp;
});


