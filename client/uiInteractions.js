// UI interactions module

// Check user preference on page load
document.addEventListener('DOMContentLoaded', function() {
    // Ensure colors button starts toggled since colors are on by default
    const colorsButton = document.getElementById('colors');
    if (!colorsButton.classList.contains('toggled')) {
        colorsButton.classList.add('toggled');
    }
});

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

