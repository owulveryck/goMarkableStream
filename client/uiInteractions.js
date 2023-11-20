let recordingWithSound = false;

document.getElementById('rotate').addEventListener('click', function () {
	portrait = !portrait;
	this.classList.toggle('toggled');
	//visibleCanvas.style.transform = "portrait(270deg)";
	//visibleCanvas.style.transformOrigin = "center center";
	eventWorker.postMessage({ type: 'portrait', portrait: portrait });

	resizeVisibleCanvas();
});
document.getElementById('pointerButton').addEventListener('click', function () {
	if (isWebSocketConnected(ws)) {
		stopWebSocket();
	} else {
		connectWebSocket();
	}
});


document.getElementById('colors').addEventListener('click', function () {
	withColor = !withColor;
	this.classList.toggle('toggled');
	streamWorker.postMessage({ type: 'withColorChanged', withColor: withColor });
});

const sidebar = document.querySelector('.sidebar');
sidebar.addEventListener('mouseover', function () {
	sidebar.classList.add('active');
});
sidebar.addEventListener('mouseout', function () {
	sidebar.classList.remove('active');
});

// Resize the canvas whenever the window is resized
window.addEventListener("resize", resizeVisibleCanvas);
resizeVisibleCanvas();

document.getElementById('switchOrderButton').addEventListener('click', function () {
	// Swap z-index values
	if (iFrame.style.zIndex == 1) {
		iFrame.style.zIndex = 4;
		return;
	}
	iFrame.style.zIndex = 1;
});


