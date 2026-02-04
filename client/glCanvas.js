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
	// Wait for DOM to be ready, then show styled error
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', () => {
			showConnectionError('WebGL is not supported by your browser. Please use a modern browser with WebGL support.', false);
		});
	} else {
		showConnectionError('WebGL is not supported by your browser. Please use a modern browser with WebGL support.', false);
	}
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

const float LASER_RADIUS = 6.0;
const float LASER_EDGE_SOFTNESS = 2.0;
const vec3 LASER_COLOR = vec3(1.0, 0.0, 0.0);

void main(void) {
    vec4 texColor = texture2D(uSampler, vTextureCoord);

    // Calculate laser pointer effect
    float dx = gl_FragCoord.x - uLaserX;
    float dy = gl_FragCoord.y - uLaserY;
    float distance = sqrt(dx * dx + dy * dy);

    // Simple laser pointer
    if (distance < 8.0 && uLaserX > 0.0 && uLaserY > 0.0) {
        float fade = 1.0 - smoothstep(6.0, 8.0, distance);
        gl_FragColor = vec4(1.0, 0.0, 0.0, fade);
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
		console.error('Unable to initialize shader program:', gl.getProgramInfoLog(shaderProgram));
		showConnectionError('Unable to initialize graphics. Your browser may not support the required WebGL features.', false);
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
			console.error('Shader compilation error:', gl.getShaderInfoLog(shader));
			showConnectionError('Unable to compile graphics shaders. Your browser may not support the required WebGL features.', false);
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
    if (TextureFlipped) {
        // Paper Pro style or RM2 firmware 3.24+ (portrait orientation)
        return [
			1.0, 0.0,
			0.0, 0.0,
			1.0, 1.0,
			0.0, 1.0,
		];
    } else {
        // Legacy RM2 style (pre-3.24, landscape orientation)
        return [
			1.0, 1.0,
			0.0, 1.0,
			1.0, 0.0,
			0.0, 0.0,
		];
    }
}

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

// Draw the scene
function drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture) {
	// Handle canvas resize for proper rendering
	if (resizeGLCanvas(gl.canvas)) {
		gl.viewport(0, 0, gl.canvas.width, gl.canvas.height);
	}
	
	// Set background color
	gl.clearColor(0.98, 0.98, 0.98, 0.25);
	
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
	const rotationMatrix = shouldRotate ? makeRotationZMatrix(90) : makeRotationZMatrix(0);
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

// Redraw scene with current texture (for rotation changes without new frame data)
function redrawScene(shouldRotate, scaleFactor) {
    const uRotationMatrixLocation = gl.getUniformLocation(shaderProgram, 'uRotationMatrix');
    const rotationMatrix = shouldRotate ? makeRotationZMatrix(90) : makeRotationZMatrix(0);
    gl.uniformMatrix4fv(uRotationMatrixLocation, false, rotationMatrix);

    const uScaleFactorLocation = gl.getUniformLocation(shaderProgram, 'uScaleFactor');
    gl.uniform1f(uScaleFactorLocation, scaleFactor);

    drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
}

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
    // Check if laser is disabled
    if (!laserEnabled) {
        laserX = -10;
        laserY = -10;
        return;
    }

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

// Clear laser pointer (hide it off-screen)
function clearLaser() {
    laserX = -10;
    laserY = -10;
    drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
}

