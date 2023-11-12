let mediaRecorder;
let recordedChunks = [];

async function startRecording() {
	const tempCanvas = createTempCanvas(); // Create the temporary canvas

	console.log("recording in progress");
	let videoStream = tempCanvas.captureStream(5); // 25 fps

	if (recordingWithSound) {
		// Capture audio stream from the user's microphone
		let audioStream;
		try {
			audioStream = await navigator.mediaDevices.getUserMedia({ audio: true });
		} catch (err) {
			console.error("Error capturing audio:", err);
			return;
		}

		// Combine video and audio streams
		let combinedStream = new MediaStream([...videoStream.getTracks(), ...audioStream.getTracks()]);

		mediaRecorder = new MediaRecorder(combinedStream, {
			mimeType: 'video/webm;codecs=vp9'
		});
	} else {
		mediaRecorder = new MediaRecorder(videoStream, {
			mimeType: 'video/webm;codecs=vp9'
		});
	}

	mediaRecorder.ondataavailable = function(event) {
		if (event.data.size > 0) {
			recordedChunks.push(event.data);
		}
	};

	mediaRecorder.onstop = function() {
		download();
	};

	mediaRecorder.start();
}

function stopRecording() {
	mediaRecorder.stop();
	removeTempCanvas(); // Remove the temporary canvas after recording

	// Stop updating tempCanvas
	if (animationFrameId) {
		cancelAnimationFrame(animationFrameId);
	}
}

function download() {
	let blob = new Blob(recordedChunks, {
		type: 'video/webm'
	});

	let url = URL.createObjectURL(blob);
	let a = document.createElement('a');
	a.style.display = 'none';
	a.href = url;
	a.download = 'goMarkableStreamRecording.webm';
	document.body.appendChild(a);
	a.click();
	setTimeout(() => {
		document.body.removeChild(a);
		window.URL.revokeObjectURL(url);
	}, 100);
}

document.getElementById('startStopButtonWithSound').addEventListener('click', function() {
	let icon = document.getElementById('icon2');
	let label = document.getElementById('label2');


	if (label.textContent === 'Record with audio') {
		label.textContent = 'Stop';
		icon.classList.add('recording');
		recordingWithSound = true;
		startRecording();
	} else {
		label.textContent = 'Record with audio';
		icon.classList.remove('recording');
		recordingWithSound = false;
		stopRecording();
	}
});
document.getElementById('startStopButton').addEventListener('click', function() {
	let icon = document.getElementById('icon');
	let label = document.getElementById('label');


	if (label.textContent === 'Record') {
		label.textContent = 'Stop';
		icon.classList.add('recording');
		startRecording();
	} else {
		label.textContent = 'Record';
		icon.classList.remove('recording');
		stopRecording();
	}
});
// JavaScript file (stream.js)
function createTempCanvas() {
	const tempCanvas = document.createElement('canvas');
	tempCanvas.width = width;
	tempCanvas.height = height;
	tempCanvas.id = 'tempCanvas'; // Assign an ID for easy reference

	// Hide the tempCanvas
	tempCanvas.style.display = 'none';

	// Start updating tempCanvas
	updateTempCanvas(tempCanvas);

	// Append tempCanvas to the body (or any other container)
	document.body.appendChild(tempCanvas);

	return tempCanvas;
}
function removeTempCanvas() {
	const tempCanvas = document.getElementById('tempCanvas');
	if (tempCanvas) {
		tempCanvas.remove();
	}
}
let animationFrameId;
function updateTempCanvas(tempCanvas) {
	renderCanvas(rawCanvas,tempCanvas);
	// Continue updating tempCanvas
	animationFrameId = requestAnimationFrame(() => updateTempCanvas(tempCanvas));
}


