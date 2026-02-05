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
			if (statusLabel) statusLabel.textContent = 'Connected';
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

// Enhanced error message builder
function buildErrorMessage(error) {
	let userMessage = '';
	let technicalDetails = '';
	let troubleshootingSteps = [];

	// Determine error type and build appropriate message
	if (error.code === 1006) {
		userMessage = 'Connection was unexpectedly closed';
		technicalDetails = 'WebSocket closed abnormally (code 1006)';
		troubleshootingSteps = [
			'Check your network connection',
			'Ensure the reMarkable device is connected',
			'Verify no firewall is blocking the connection'
		];
	} else if (error.code === 1000) {
		userMessage = 'Connection closed normally';
		technicalDetails = 'WebSocket closed (code 1000)';
		troubleshootingSteps = ['The connection was intentionally closed'];
	} else if (error.message && error.message.includes('network')) {
		userMessage = 'Network connection issue';
		technicalDetails = error.message;
		troubleshootingSteps = [
			'Check your internet connection',
			'Try refreshing the page',
			'Disable VPN if enabled'
		];
	} else {
		userMessage = error.message || 'An unexpected error occurred';
		technicalDetails = error.toString();
		troubleshootingSteps = [
			'Try refreshing the page',
			'Check browser console for more details'
		];
	}

	return { userMessage, technicalDetails, troubleshootingSteps };
}

// Show connection error with enhanced messaging
function showConnectionError(error, retryable = true, onRetry = null) {
	// Remove any existing error dialog
	const existingError = document.getElementById('connectionError');
	if (existingError) {
		existingError.remove();
	}

	const { userMessage, technicalDetails, troubleshootingSteps } =
		typeof error === 'string' ?
			{ userMessage: error, technicalDetails: '', troubleshootingSteps: [] } :
			buildErrorMessage(error);

	const errorDiv = document.createElement('div');
	errorDiv.id = 'connectionError';
	errorDiv.className = 'connection-error visible';

	let troubleshootingHtml = '';
	if (troubleshootingSteps.length > 0) {
		troubleshootingHtml = `
			<div class="error-troubleshooting">
				<h4>Try these steps:</h4>
				<ul>
					${troubleshootingSteps.map(step => `<li>${step}</li>`).join('')}
				</ul>
			</div>
		`;
	}

	let technicalHtml = '';
	if (technicalDetails) {
		technicalHtml = `
			<details class="error-technical">
				<summary>Technical details</summary>
				<code>${technicalDetails}</code>
			</details>
		`;
	}

	let buttonsHtml = '';
	if (retryable && onRetry) {
		buttonsHtml = `
			<div class="error-buttons">
				<button id="retryConnectionBtn" class="error-primary-btn">Retry Connection</button>
				<button id="refreshPageBtn" class="error-secondary-btn">Refresh Page</button>
				<button class="dismiss-btn" id="dismissErrorBtn">Dismiss</button>
			</div>
		`;
	} else {
		buttonsHtml = `
			<div class="error-buttons">
				<button id="refreshPageBtn" class="error-primary-btn">Refresh Page</button>
				<button id="dismissErrorBtn" class="dismiss-btn">Dismiss</button>
			</div>
		`;
	}

	errorDiv.innerHTML = `
		<div class="error-icon">
			<svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor">
				<path d="M12,2L1,21H23M12,6L19.53,19H4.47M11,10V14H13V10M11,16V18H13V16"/>
			</svg>
		</div>
		<h3>Connection Error</h3>
		<p class="error-message">${userMessage}</p>
		${troubleshootingHtml}
		${technicalHtml}
		${buttonsHtml}
	`;

	document.body.appendChild(errorDiv);

	// Set up button handlers
	const dismissBtn = document.getElementById('dismissErrorBtn');
	if (dismissBtn) {
		dismissBtn.addEventListener('click', () => {
			errorDiv.classList.remove('visible');
			setTimeout(() => errorDiv.remove(), 300);
		});
	}

	const refreshBtn = document.getElementById('refreshPageBtn');
	if (refreshBtn) {
		refreshBtn.addEventListener('click', () => {
			window.location.reload();
		});
	}

	if (retryable && onRetry) {
		const retryBtn = document.getElementById('retryConnectionBtn');
		if (retryBtn) {
			retryBtn.addEventListener('click', () => {
				errorDiv.classList.remove('visible');
				setTimeout(() => errorDiv.remove(), 300);
				onRetry();
			});
		}
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


