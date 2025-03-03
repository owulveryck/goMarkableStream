function downloadScreenshot(dataUrl) {
	// Use 'toDataURL' to capture the current canvas content
	// Create an 'a' element for downloading
	var link = document.getElementById("screenshot");

	link.download = 'goMarkableScreenshot.png';
	link.href = dataURL;
	link.click();
}

// Function to show a message with auto-hide after specified duration
function showMessage(message, duration = 3000) {
	const messageDiv = document.getElementById('message');
	messageDiv.textContent = message;
	messageDiv.classList.add('visible');
	
	// Auto-hide after specified duration
	setTimeout(() => {
		messageDiv.classList.remove('visible');
	}, duration);
}

// Wait/loading message display
function waiting(message) {
	const messageDiv = document.getElementById('message');
	messageDiv.innerHTML = `${message} <span class="loading-dots"></span>`;
	messageDiv.classList.add('visible');
}

// Function to fetch app version from the server
async function fetchVersion() {
	try {
		const response = await fetch('/version');
		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}
		const version = await response.text();
		return version;
	} catch (error) {
		console.error('Error fetching version:', error);
		return 'unknown';
	}
}


