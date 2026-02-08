// JWT Authentication Module
const AUTH_TOKEN_KEY = 'goMarkableStream_jwt';
const AUTH_EXPIRY_KEY = 'goMarkableStream_jwt_expiry';

// Get stored auth token
function getAuthToken() {
    const token = localStorage.getItem(AUTH_TOKEN_KEY);
    const expiry = localStorage.getItem(AUTH_EXPIRY_KEY);

    if (!token || !expiry) {
        return null;
    }

    // Check if token has expired
    if (Date.now() > parseInt(expiry, 10)) {
        clearAuthToken();
        return null;
    }

    return token;
}

// Store auth token
function setAuthToken(token, expiresIn) {
    localStorage.setItem(AUTH_TOKEN_KEY, token);
    // Store expiry time with a small buffer (5 minutes before actual expiry)
    const expiryTime = Date.now() + (expiresIn * 1000) - (5 * 60 * 1000);
    localStorage.setItem(AUTH_EXPIRY_KEY, expiryTime.toString());
}

// Clear stored auth token
function clearAuthToken() {
    localStorage.removeItem(AUTH_TOKEN_KEY);
    localStorage.removeItem(AUTH_EXPIRY_KEY);
}

// Check if user is authenticated
function isAuthenticated() {
    return getAuthToken() !== null;
}

// Login with username and password
async function login(username, password) {
    const response = await fetch('/login', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || 'Login failed');
    }

    const data = await response.json();
    setAuthToken(data.token, data.expiresIn);
    return data;
}

// Logout
function logout() {
    clearAuthToken();
    window.location.reload();
}

// Show login modal
function showLoginModal() {
    const modal = document.getElementById('loginModal');
    if (modal) {
        modal.style.display = 'flex';
        const usernameInput = document.getElementById('loginUsername');
        if (usernameInput) {
            usernameInput.focus();
        }
    }
}

// Hide login modal
function hideLoginModal() {
    const modal = document.getElementById('loginModal');
    if (modal) {
        modal.style.display = 'none';
    }
    // Clear form
    const form = document.getElementById('loginForm');
    if (form) {
        form.reset();
    }
    const errorEl = document.getElementById('loginError');
    if (errorEl) {
        errorEl.style.display = 'none';
    }
}

// Handle login form submission
async function handleLoginSubmit(event) {
    event.preventDefault();

    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    const errorEl = document.getElementById('loginError');
    const submitBtn = document.getElementById('loginSubmit');

    // Disable submit button during login
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.textContent = 'Logging in...';
    }

    try {
        await login(username, password);
        hideLoginModal();
        // Reload page to reinitialize with token
        window.location.reload();
    } catch (error) {
        if (errorEl) {
            errorEl.textContent = error.message;
            errorEl.style.display = 'block';
        }
    } finally {
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Login';
        }
    }
}

// Initialize auth UI
function initAuthUI() {
    const form = document.getElementById('loginForm');
    if (form) {
        form.addEventListener('submit', handleLoginSubmit);
    }

    // Close modal when clicking outside
    const modal = document.getElementById('loginModal');
    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                // Don't close if not authenticated - they need to log in
                if (isAuthenticated()) {
                    hideLoginModal();
                }
            }
        });
    }

    // Close modal on Escape key (only if authenticated)
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && isAuthenticated()) {
            hideLoginModal();
        }
    });
}

// Add auth header to fetch requests
function authFetch(url, options = {}) {
    const token = getAuthToken();
    if (token) {
        options.headers = options.headers || {};
        options.headers['Authorization'] = `Bearer ${token}`;
    }
    return fetch(url, options);
}

// Get URL with token parameter (for SSE endpoints)
function getAuthenticatedURL(url) {
    const token = getAuthToken();
    if (!token) {
        return url;
    }
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}token=${encodeURIComponent(token)}`;
}
