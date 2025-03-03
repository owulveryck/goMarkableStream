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
    
    // Show confirmation message
    showMessage(`Display ${portrait ? 'portrait' : 'landscape'} mode activated`, 2000);
});

// Colors button functionality
document.getElementById('colors').addEventListener('click', function () {
    withColor = !withColor;
    this.classList.toggle('toggled');
    streamWorker.postMessage({ type: 'withColorChanged', withColor: withColor });
    
    // Show confirmation message
    showMessage(`${withColor ? 'Color' : 'Grayscale'} mode enabled`, 2000);
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
    const isLayerSwitched = iFrame.style.zIndex != 1;
    
    if (isLayerSwitched) {
        iFrame.style.zIndex = 1;
        this.classList.remove('toggled');
        showMessage('Drawing layer on top', 2000);
    } else {
        iFrame.style.zIndex = 4;
        this.classList.add('toggled');
        showMessage('Content layer on top', 2000);
    }
});

// Contrast slider functionality
document.getElementById('contrastSlider').addEventListener('input', function() {
    // Get the slider value (between 1.0 and 3.0)
    const contrastLevel = this.value;
    
    // Update renderer if function exists
    if (typeof setContrast === 'function') {
        setContrast(contrastLevel);
    }
    
    // Show feedback when user stops moving the slider
    clearTimeout(this.timeout);
    this.timeout = setTimeout(() => {
        showMessage(`Contrast: ${parseFloat(contrastLevel).toFixed(1)}`, 1000);
    }, 500);
});

// Load saved contrast value on initialization
document.addEventListener('DOMContentLoaded', function() {
    // Check for saved contrast preference
    const savedContrast = localStorage.getItem('contrastLevel');
    const contrastSlider = document.getElementById('contrastSlider');
    
    if (savedContrast) {
        // Set the slider to the saved value
        contrastSlider.value = savedContrast;
        
        // Update the contrast setting
        if (typeof setContrast === 'function') {
            setContrast(savedContrast);
        }
    }
});


