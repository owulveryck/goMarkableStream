const ctxCanvasPresent = canvasPresent.getContext('2d');

// Variables to store the latest positions
let latestX = canvasPresent.width / 2;
let latestY = canvasPresent.height / 2;
let ws;
// Constants for the maximum values from the WebSocket messages
const MAX_X_VALUE = 15725;
const MAX_Y_VALUE = 20966;

// Function to draw the laser pointer
function drawLaser(x, y) {
	ctxCanvasPresent.clearRect(0, 0, canvasPresent.width, canvasPresent.height); // Clear the canvasPresent
	ctxCanvasPresent.beginPath();
	ctxCanvasPresent.arc(x, y, 10, 0, 2 * Math.PI, false); // Draw a circle for the laser pointer
	ctxCanvasPresent.fillStyle = 'red';
	ctxCanvasPresent.fill();
}

// Function to clear the laser pointer
function clearLaser() {
	ctxCanvasPresent.clearRect(0, 0, canvasPresent.width, canvasPresent.height); // Clear the canvasPresent
}

