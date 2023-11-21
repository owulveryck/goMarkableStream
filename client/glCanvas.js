// WebGL initialization
//const gl = visibleCanvas.getContext('webgl');
//const gl = canvas.getContext('webgl', { antialias: true, preserveDrawingBuffer: true  });
let laserX = 0; // Initialize with default values
let laserY = 0;
const gl = canvas.getContext('webgl', { antialias: true });


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
	gl_Position = uRotationMatrix * vec4(aVertexPosition.xy * uScaleFactor, aVertexPosition.zw);
	vTextureCoord = aTextureCoord;
}
`;

// Fragment shader program
const fsSource = `
precision mediump float; // Add this line for precision specification

varying highp vec2 vTextureCoord;
uniform sampler2D uSampler;
uniform float uLaserX;
uniform float uLaserY;

void main(void) {
	
	// Check if the current fragment is at the laser position
    if(abs(gl_FragCoord.x - uLaserX) < 5.0 && abs(gl_FragCoord.y - uLaserY) < 5.0) {
        gl_FragColor = vec4(1.0, 0.0, 0.0, 1.0); // Red color for the laser pointer
    } else {
        gl_FragColor = texture2D(uSampler, vTextureCoord);
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

const textureCoordinates = [
//	1.0,  0.0,  // Bottom right
//	0.0,  0.0,  // Bottom left
//	1.0,  1.0,  // Top right
//	0.0,  1.0,  // Top left
	1.0, 1.0,
	0.0, 1.0,
	1.0, 0.0,
	0.0, 0.0,
];

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
let imageData = new ImageData(width, height);
gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, imageData);

// Draw the scene
function drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture) {
	if (resizeGLCanvas(gl.canvas)) {
		gl.viewport(0, 0, gl.canvas.width, gl.canvas.height);
	}
	gl.clearColor(0.5, 0.5, 0.5, 0.25);  // Gray with 75% transparency
	gl.clearDepth(1.0);                 // Clear everything
	gl.enable(gl.DEPTH_TEST);           // Enable depth testing
	gl.depthFunc(gl.LEQUAL);            // Near things obscure far things

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
	gl.bindTexture(gl.TEXTURE_2D, texture);
	gl.texSubImage2D(gl.TEXTURE_2D, 0, 0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, newRawData);

	// Set rotation
	const uRotationMatrixLocation = gl.getUniformLocation(shaderProgram, 'uRotationMatrix');
	const rotationMatrix = shouldRotate ? makeRotationZMatrix(270) : makeRotationZMatrix(0);
	gl.uniformMatrix4fv(uRotationMatrixLocation, false, rotationMatrix);

	// Set scaling
	const uScaleFactorLocation = gl.getUniformLocation(shaderProgram, 'uScaleFactor');
	gl.uniform1f(uScaleFactorLocation, scaleFactor);

	drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
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

function updateLaserPosition(x, y) {
    laserX = x;
    laserY = y;
    drawScene(gl, programInfo, positionBuffer, textureCoordBuffer, texture);
}
