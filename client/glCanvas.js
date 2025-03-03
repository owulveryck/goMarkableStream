// WebGL initialization
// Use -10,-10 as the default laser coordinate (off-screen) to hide the pointer initially
let laserX = -10; 
let laserY = -10;
const gl = canvas.getContext('webgl', { 
    antialias: true,
    preserveDrawingBuffer: true,  // Important for proper rendering
    alpha: true                   // Enable transparency
});


if (!gl) {
	alert('WebGL not supported');
}

// Vertex shader program
const vsSource = `
attribute vec4 aVertexPosition;
attribute vec2 aTextureCoord;
uniform mat4 uRotationMatrix;
uniform float uScaleFactor;
varying highp vec2 vTextureCoord;

void main(void) {
    // Apply scaling and rotation transformations
    gl_Position = uRotationMatrix * vec4(aVertexPosition.xy * uScaleFactor, aVertexPosition.zw);
    
    // Pass texture coordinates to fragment shader
    vTextureCoord = aTextureCoord;
}
`;

// Fragment shader program
const fsSource = `
precision highp float;

varying highp vec2 vTextureCoord;
uniform sampler2D uSampler;
uniform float uLaserX;
uniform float uLaserY;
uniform bool uDarkMode;

// Constants for laser pointer visualization
const float LASER_RADIUS = 6.0;
const float LASER_EDGE_SOFTNESS = 2.0;
const vec3 LASER_COLOR = vec3(1.0, 0.0, 0.0);

// Constants for image processing
const float CONTRAST = 1.15;  // Slight contrast boost
const float BRIGHTNESS = 0.05;  // Slight brightness boost
const float SHARPNESS = 0.5;  // Sharpness level

// Get texture color without any sharpening - better for handwriting
vec4 getBaseTexture(sampler2D sampler, vec2 texCoord) {
    return texture2D(sampler, texCoord);
}

void main(void) {
    // Get base texture color directly - no sharpening for clearer handwriting
    vec4 texColor = getBaseTexture(uSampler, vTextureCoord);
    
    // Apply very mild contrast adjustments - avoid distortion
    vec3 adjusted = (texColor.rgb - 0.5) * 1.05 + 0.5;
    texColor.rgb = clamp(adjusted, 0.0, 1.0);
    
    // Calculate laser pointer effect
    float dx = gl_FragCoord.x - uLaserX;
    float dy = gl_FragCoord.y - uLaserY;
    float distance = sqrt(dx * dx + dy * dy);
    
    if (uDarkMode) {
        // Invert colors in dark mode, but preserve alpha
        texColor.rgb = 1.0 - texColor.rgb;
    }
    
    // Simple laser pointer - more reliable rendering
    if (distance < 8.0 && uLaserX > 0.0 && uLaserY > 0.0) {
        // Create solid circle with slight fade at edge
        float fade = 1.0 - smoothstep(6.0, 8.0, distance);
        gl_FragColor = vec4(1.0, 0.0, 0.0, fade); // Red with fade at edge
    } else {
        gl_FragColor = texColor;
    }
}
`;

function makeRotationZMatrix(angleInDegrees) {
	var angleInRadians = angleInDegrees * Math.PI / 180;
	var s = Math.sin(angleInRadians);
	var c = Math.cos(angleInRadians);

	return [
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1
	];
}

// Initialize a shader program
function initShaderProgram(gl, vsSource, fsSource) {
	const vertexShader = loadShader(gl, gl.VERTEX_SHADER, vsSource);
	const fragmentShader = loadShader(gl, gl.FRAGMENT_SHADER, fsSource);

	const shaderProgram = gl.createProgram();
	gl.attachShader(shaderProgram, vertexShader);
	gl.attachShader(shaderProgram, fragmentShader);
	gl.linkProgram(shaderProgram);

	if (!gl.getProgramParameter(shaderProgram, gl.LINK_STATUS)) {
		alert('Unable to initialize the shader program: ' + gl.getProgramInfoLog(shaderProgram));
		return null;
	}

	return shaderProgram;
}

// Creates a shader of the given type, uploads the source and compiles it.
	function loadShader(gl, type, source) {
		const shader = gl.createShader(type);
		gl.shaderSource(shader, source);
		gl.compileShader(shader);

		if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
			alert('An error occurred compiling the shaders: ' + gl.getShaderInfoLog(shader));
			gl.deleteShader(shader);
			return null;
		}

		return shader;
	}

const shaderProgram = initShaderProgram(gl, vsSource, fsSource);

// Collect all the info needed to use the shader program.
	// Look up locations of attributes and uniforms used by our shader
const programInfo = {
	program: shaderProgram,
	attribLocations: {
		vertexPosition: gl.getAttribLocation(shaderProgram, 'aVertexPosition'),
		textureCoord: gl.getAttribLocation(shaderProgram, 'aTextureCoord'),
	},
	uniformLocations: {
		uSampler: gl.getUniformLocation(shaderProgram, 'uSampler'),
        uLaserX: gl.getUniformLocation(shaderProgram, 'uLaserX'),
        uLaserY: gl.getUniformLocation(shaderProgram, 'uLaserY'),
        uDarkMode: gl.getUniformLocation(shaderProgram, 'uDarkMode'),
	},
};

// Create a buffer for the square's positions.
	const positionBuffer = gl.createBuffer();

// Select the positionBuffer as the one to apply buffer operations to from here out.
	gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);

// Now create an array of positions for the square.
	const positions = [
		1.0, 1.0,
		-1.0, 1.0,
		1.0, -1.0,
		-1.0, -1.0,
	];

// Pass the list of positions into WebGL to build the shape.
	gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(positions), gl.STATIC_DRAW);

// Set up texture coordinates for the rectangle
const textureCoordBuffer = gl.createBuffer();
gl.bindBuffer(gl.ARRAY_BUFFER, textureCoordBuffer);

const textureCoordinates = getTextureCoordinates();

function getTextureCoordinates() {
    if (DeviceModel === "Remarkable2") {
        return [
			1.0, 1.0,
			0.0, 1.0,
			1.0, 0.0,
			0.0, 0.0,
		];
    } else {
        return [
			1.0, 0.0,
			0.0, 0.0,
			1.0, 1.0,
			0.0, 1.0,
		];
    };
};

gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(textureCoordinates), gl.STATIC_DRAW);

// Create a texture.
	const texture = gl.createTexture();
gl.bindTexture(gl.TEXTURE_2D, texture);


// Set the parameters so we can render any size image.
gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
// To apply a smoothing algorithm, you'll likely want to adjust the texture filtering parameters in your WebGL setup. 
// For smoothing, typically gl.LINEAR is used for both gl.TEXTURE_MIN_FILTER and gl.TEXTURE_MAG_FILTER
gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
// gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
// gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);

// Upload the image into the texture.
let imageData = new ImageData(screenWidth, screenHeight);
gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, imageData);

// Variable to track dark mode state, default is false (light mode)
let isDarkMode = false;

// Draw the scene
function drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture) {
	// Handle canvas resize for proper rendering
	if (resizeGLCanvas(gl.canvas)) {
		gl.viewport(0, 0, gl.canvas.width, gl.canvas.height);
	}
	
	// Adjust background color based on dark mode
	const bgColor = isDarkMode 
		? [0.12, 0.12, 0.13, 0.25]  // Darker, more neutral dark mode bg
		: [0.98, 0.98, 0.98, 0.25]; // Nearly white light mode bg
	gl.clearColor(bgColor[0], bgColor[1], bgColor[2], bgColor[3]);
	
	// Enable alpha blending for transparency
	gl.enable(gl.BLEND);
	gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
	
	// Setup depth buffer
	gl.clearDepth(1.0);
	gl.enable(gl.DEPTH_TEST);
	gl.depthFunc(gl.LEQUAL);

	// Clear the canvas before we start drawing on it.
	gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);

	// Tell WebGL to use our program when drawing
	gl.useProgram(programInfo.program);

	// Set the shader attributes
	gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
	gl.vertexAttribPointer(programInfo.attribLocations.vertexPosition, 2, gl.FLOAT, false, 0, 0);
	gl.enableVertexAttribArray(programInfo.attribLocations.vertexPosition);

	gl.bindBuffer(gl.ARRAY_BUFFER, textureCoordBuffer);
	gl.vertexAttribPointer(programInfo.attribLocations.textureCoord, 2, gl.FLOAT, false, 0, 0);
	gl.enableVertexAttribArray(programInfo.attribLocations.textureCoord);

	// Tell WebGL we want to affect texture unit 0
	gl.activeTexture(gl.TEXTURE0);

	// Bind the texture to texture unit 0
	gl.bindTexture(gl.TEXTURE_2D, texture);

	// Tell the shader we bound the texture to texture unit 0
	gl.uniform1i(programInfo.uniformLocations.uSampler, 0);

	// Set the laser coordinates
    gl.uniform1f(programInfo.uniformLocations.uLaserX, laserX);
    gl.uniform1f(programInfo.uniformLocations.uLaserY, laserY);
    
    // Set the dark mode flag
    gl.uniform1i(programInfo.uniformLocations.uDarkMode, isDarkMode ? 1 : 0);

	gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
}

drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);

// Update texture
function updateTexture(newRawData, shouldRotate, scaleFactor) {
	if (useBGRA) {
        convertBGRAtoRGBA(newRawData);
    };

	gl.bindTexture(gl.TEXTURE_2D, texture);
	gl.texSubImage2D(gl.TEXTURE_2D, 0, 0, 0, screenWidth, screenHeight, gl.RGBA, gl.UNSIGNED_BYTE, newRawData);

	// Set rotation
	const uRotationMatrixLocation = gl.getUniformLocation(shaderProgram, 'uRotationMatrix');
	const rotationMatrix = shouldRotate ? makeRotationZMatrix(270) : makeRotationZMatrix(0);
	gl.uniformMatrix4fv(uRotationMatrixLocation, false, rotationMatrix);

	// Set scaling
	const uScaleFactorLocation = gl.getUniformLocation(shaderProgram, 'uScaleFactor');
	gl.uniform1f(uScaleFactorLocation, scaleFactor);

	drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
}

function convertBGRAtoRGBA(data) {
    for (let i = 0; i < data.length; i += 4) {
        const b = data[i];     // Blue
        data[i] = data[i + 2]; // Swap Red and Blue
        data[i + 2] = b;
    }
}

// Call `updateTexture` with new data whenever you need to update the image

// Let's create a function that resizes the canvas element. 
// This function will adjust the canvas's width and height attributes based on its display size, which can be set using CSS or directly in JavaScript.
function resizeGLCanvas(canvas) {
	const displayWidth = canvas.clientWidth;
	const displayHeight = canvas.clientHeight;

	// Check if the canvas size is different from its display size
	if (canvas.width !== displayWidth || canvas.height !== displayHeight) {
		// Make the canvas the same size as its display size
		canvas.width = displayWidth;
		canvas.height = displayHeight;
		return true; // indicates that the size was changed
	}

	return false; // indicates no change in size
}

// Direct laser pointer position - no animation for more reliability
function updateLaserPosition(x, y) {
    // If x and y are valid positive values
    if (x > 0 && y > 0) {
        // Position is now directly proportional to canvas size
        laserX = x * (gl.canvas.width / screenWidth);
        laserY = gl.canvas.height - (y * (gl.canvas.height / screenHeight));
    } else {
        // Hide the pointer by moving it off-screen
        laserX = -10;
        laserY = -10;
    }
    
    // Redraw immediately
    drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
}

// Function to update dark mode state with transition effect
let darkModeTransition = 0; // 0 = light mode, 1 = dark mode
let transitionActive = false;

function setDarkMode(darkModeEnabled) {
    isDarkMode = darkModeEnabled;
    
    // If not already transitioning, start a smooth transition
    if (!transitionActive) {
        transitionActive = true;
        const startTime = performance.now();
        const duration = 300; // transition duration in ms
        
        function animateDarkModeTransition(timestamp) {
            const elapsed = timestamp - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // Update transition value (0 to 1 for light to dark)
            darkModeTransition = darkModeEnabled ? progress : 1 - progress;
            
            // Render with current transition value
            drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
            
            // Continue animation if not complete
            if (progress < 1) {
                requestAnimationFrame(animateDarkModeTransition);
            } else {
                transitionActive = false;
            }
        }
        
        requestAnimationFrame(animateDarkModeTransition);
    } else {
        // Just update the scene if already transitioning
        drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
    }
}
