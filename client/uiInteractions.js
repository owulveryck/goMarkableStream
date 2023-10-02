let rotate = true;
let withColor = true;
let recordingWithSound = false;

document.getElementById('rotate').addEventListener('click', function() {
    rotate = !rotate;
    this.classList.toggle('toggled');
    resizeAndCopy();
});

document.getElementById('colors').addEventListener('click', function() {
    withColor = !withColor;
    this.classList.toggle('toggled');
    resizeAndCopy();
});

const sidebar = document.querySelector('.sidebar');
sidebar.addEventListener('mouseover', function() {
    sidebar.classList.add('active');
});
sidebar.addEventListener('mouseout', function() {
    sidebar.classList.remove('active');
});
