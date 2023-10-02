let rotate = true;
let withColor = true;
let recordingWithSound = false;

document.getElementById('rotate').addEventListener('click', function() {
	rotate = !rotate;
	this.classList.toggle('active');
	resizeVisibleCanvas();
});

document.getElementById('colors').addEventListener('click', function() {
	withColor = !withColor;
	this.classList.toggle('active');
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
