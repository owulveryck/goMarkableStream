function downloadScreenshot(dataUrl) {
	// Use 'toDataURL' to capture the current canvas content
	// Create an 'a' element for downloading
	var link = document.getElementById("screenshot");

	link.download = 'goMarkableScreenshot.png';
	link.href = dataURL;
	link.click();
}

// Message duration constants for consistent UX
const MessageDuration = {
	QUICK: 2000,      // Simple confirmations (toggle states, mode changes)
	NORMAL: 3500,     // Standard messages (status updates)
	IMPORTANT: 5000   // Messages requiring attention (errors, warnings)
};

// Track the current message timeout to prevent race conditions
let messageTimeoutId = null;

// Function to show a message with auto-hide after specified duration
function showMessage(message, duration = MessageDuration.NORMAL) {
	const messageDiv = document.getElementById('message');
	messageDiv.textContent = message;
	messageDiv.classList.add('visible');

	// Clear any pending timeout
	if (messageTimeoutId) {
		clearTimeout(messageTimeoutId);
	}

	// Auto-hide after specified duration
	messageTimeoutId = setTimeout(() => {
		messageDiv.classList.remove('visible');
		messageTimeoutId = null;
	}, duration);
}

// Show persistent status message (e.g., for Funnel mode)
function showStatusMessage(message) {
	const messageDiv = document.getElementById('message');

	// Clear any pending timeout from transient messages
	if (messageTimeoutId) {
		clearTimeout(messageTimeoutId);
		messageTimeoutId = null;
	}

	messageDiv.textContent = message;
	messageDiv.classList.add('info');
	messageDiv.classList.add('visible');
}

// Hide persistent status message
function hideStatusMessage() {
	const messageDiv = document.getElementById('message');
	messageDiv.classList.remove('visible');
	messageDiv.classList.remove('info');
}

// Wait/loading message display
function waiting(message) {
	const messageDiv = document.getElementById('message');
	messageDiv.innerHTML = `${message} <span class="loading-dots"></span>`;
	messageDiv.classList.add('visible');
}

// Connection status management
const ConnectionState = {
	CONNECTING: 'connecting',
	CONNECTED: 'connected',
	PAUSED: 'paused',
	RATE_LIMITED: 'rate-limited',
	RECONNECTING: 'reconnecting',
	DISCONNECTED: 'disconnected'
};

let statusHideTimeout = null;

// Timeout for collapsing status indicator to dot-only
let statusCollapseTimeout = null;

// Update the connection status indicator
function updateConnectionStatus(state) {
	const indicator = document.getElementById('statusIndicator');
	const statusLabel = indicator.querySelector('.status-label');

	// Clear any pending hide timeout
	if (statusHideTimeout) {
		clearTimeout(statusHideTimeout);
		statusHideTimeout = null;
	}

	// Clear collapse timeout
	if (statusCollapseTimeout) {
		clearTimeout(statusCollapseTimeout);
		statusCollapseTimeout = null;
	}

	// Remove all state classes
	indicator.classList.remove('connected', 'paused', 'rate-limited', 'reconnecting', 'disconnected', 'hidden', 'collapsed');

	switch (state) {
		case ConnectionState.CONNECTING:
			// Gray dot (default)
			if (statusLabel) statusLabel.textContent = 'Connecting...';
			break;
		case ConnectionState.CONNECTED:
			indicator.classList.add('connected');
			// Pulsing green dot indicates receiving - no label needed
			break;
		case ConnectionState.PAUSED:
			indicator.classList.add('paused');
			if (statusLabel) statusLabel.textContent = 'Stream paused';
			break;
		case ConnectionState.RATE_LIMITED:
			indicator.classList.add('rate-limited');
			if (statusLabel) statusLabel.textContent = 'Rate limited';
			break;
		case ConnectionState.RECONNECTING:
			indicator.classList.add('reconnecting');
			if (statusLabel) statusLabel.textContent = 'Reconnecting...';
			break;
		case ConnectionState.DISCONNECTED:
			indicator.classList.add('disconnected');
			if (statusLabel) statusLabel.textContent = 'Disconnected';
			break;
	}
}

// Show connection error with optional retry button
function showConnectionError(message, retryable = true, onRetry = null) {
	// Remove any existing error dialog
	const existingError = document.getElementById('connectionError');
	if (existingError) {
		existingError.remove();
	}

	const errorDiv = document.createElement('div');
	errorDiv.id = 'connectionError';
	errorDiv.className = 'connection-error';

	let buttonsHtml = '';
	if (retryable && onRetry) {
		buttonsHtml = `
			<button id="retryConnectionBtn">Retry</button>
			<button class="dismiss-btn" id="dismissErrorBtn">Dismiss</button>
		`;
	} else {
		buttonsHtml = '<button id="dismissErrorBtn">OK</button>';
	}

	errorDiv.innerHTML = `
		<h3>Connection Error</h3>
		<p>${message}</p>
		${buttonsHtml}
	`;

	document.body.appendChild(errorDiv);

	// Set up button handlers
	const dismissBtn = document.getElementById('dismissErrorBtn');
	dismissBtn.addEventListener('click', () => {
		errorDiv.remove();
	});

	if (retryable && onRetry) {
		const retryBtn = document.getElementById('retryConnectionBtn');
		retryBtn.addEventListener('click', () => {
			errorDiv.remove();
			onRetry();
		});
	}
}

// Hide any visible connection error
function hideConnectionError() {
	const existingError = document.getElementById('connectionError');
	if (existingError) {
		existingError.remove();
	}
}

// Onboarding hint management
const ONBOARDING_KEY = 'goMarkableStream_onboardingComplete';

function showOnboardingHint() {
	// Check if user has already seen the hint
	if (localStorage.getItem(ONBOARDING_KEY)) {
		return;
	}

	const hint = document.getElementById('onboardingHint');
	if (!hint) return;

	// Show hint after a short delay
	setTimeout(() => {
		hint.classList.add('visible');
	}, 1000);

	// Dismiss on click
	hint.addEventListener('click', dismissOnboardingHint);

	// Dismiss when sidebar is hovered
	const sidebar = document.querySelector('.sidebar');
	if (sidebar) {
		sidebar.addEventListener('mouseenter', dismissOnboardingHint, { once: true });
	}
}

function dismissOnboardingHint() {
	const hint = document.getElementById('onboardingHint');
	if (hint) {
		hint.classList.remove('visible');
		localStorage.setItem(ONBOARDING_KEY, 'true');
	}
}

// Show/hide reconnection banner
function showReconnectBanner(attempt, maxAttempts) {
	const banner = document.getElementById('reconnectBanner');
	if (!banner) return;

	const text = banner.querySelector('.reconnect-text');
	if (text) {
		text.textContent = `Reconnecting (attempt ${attempt}/${maxAttempts})...`;
	}
	banner.classList.add('visible');
}

function hideReconnectBanner() {
	const banner = document.getElementById('reconnectBanner');
	if (banner) {
		banner.classList.remove('visible');
	}
}


