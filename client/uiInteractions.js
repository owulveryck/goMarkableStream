// UI interactions module

// Function to toggle dark mode
function toggleDarkMode() {
    document.body.classList.toggle('dark-mode');
    
    // Save user preference to localStorage
    const isDarkMode = document.body.classList.contains('dark-mode');
    localStorage.setItem('darkMode', isDarkMode ? 'enabled' : 'disabled');
    
    // Update the canvas to invert colors in dark mode
    if (typeof setDarkMode === 'function') {
        setDarkMode(isDarkMode);
    }
}

// Check user preference on page load
document.addEventListener('DOMContentLoaded', function() {
    // Check for saved theme preference
    const savedTheme = localStorage.getItem('darkMode');
    const checkbox = document.getElementById('checkbox');
    
    // If user previously enabled dark mode
    if (savedTheme === 'enabled') {
        document.body.classList.add('dark-mode');
        checkbox.checked = true;
        
        // Apply dark mode to canvas
        if (typeof setDarkMode === 'function') {
            setDarkMode(true);
        }
    }
    
    // Ensure colors button starts toggled since colors are on by default
    const colorsButton = document.getElementById('colors');
    if (!colorsButton.classList.contains('toggled')) {
        colorsButton.classList.add('toggled');
    }
});

// Event listeners for dark mode toggle
document.getElementById('checkbox').addEventListener('change', toggleDarkMode);

// Rotate button functionality
document.getElementById('rotate').addEventListener('click', function () {
    portrait = !portrait;
    this.classList.toggle('toggled');
    eventWorker.postMessage({ type: 'portrait', portrait: portrait });
    resizeVisibleCanvas();
});

// Colors button functionality
document.getElementById('colors').addEventListener('click', function () {
    withColor = !withColor;
    this.classList.toggle('toggled');
    streamWorker.postMessage({ type: 'withColorChanged', withColor: withColor });
});

// Sidebar hover effect
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

// Mask drawing button functionality
document.getElementById('switchOrderButton').addEventListener('click', function () {
    // Swap z-index values
    if (iFrame.style.zIndex == 1) {
        iFrame.style.zIndex = 4;
        return;
    }
    iFrame.style.zIndex = 1;
});


