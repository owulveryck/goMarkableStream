// Minimal QR Code Generator - SVG output
// Based on QR Code specification, supports alphanumeric URLs

(function() {
    'use strict';

    // QR Code constants
    const MODE_BYTE = 4;
    const EC_LEVEL_L = 1; // Low error correction (7%)

    // Version capacities for byte mode with EC level L
    const VERSION_CAPACITY = [
        0, 17, 32, 53, 78, 106, 134, 154, 192, 230, 271,
        311, 362, 412, 450, 504, 560, 624, 688, 750, 816,
        882, 948, 1016, 1080, 1150, 1220, 1290, 1370, 1450, 1530,
        1610, 1690, 1770, 1850, 1940, 2030, 2120, 2210, 2310, 2410
    ];

    // Error correction codewords per block
    const EC_CODEWORDS = [
        0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18,
        20, 24, 26, 30, 22, 24, 28, 30, 28, 28,
        28, 28, 30, 30, 26, 28, 30, 30, 30, 30,
        30, 30, 30, 30, 30, 30, 30, 30, 30, 30
    ];

    // Number of blocks for EC level L
    const NUM_BLOCKS = [
        0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4,
        4, 4, 4, 4, 6, 6, 6, 6, 7, 8,
        8, 9, 9, 10, 12, 12, 12, 13, 14, 15,
        16, 17, 18, 19, 19, 20, 21, 22, 24, 25
    ];

    // Galois field tables for Reed-Solomon
    const GF_EXP = new Uint8Array(512);
    const GF_LOG = new Uint8Array(256);

    // Initialize Galois field tables
    (function initGF() {
        let x = 1;
        for (let i = 0; i < 255; i++) {
            GF_EXP[i] = x;
            GF_LOG[x] = i;
            x <<= 1;
            if (x & 0x100) x ^= 0x11d;
        }
        for (let i = 255; i < 512; i++) {
            GF_EXP[i] = GF_EXP[i - 255];
        }
    })();

    function gfMul(a, b) {
        if (a === 0 || b === 0) return 0;
        return GF_EXP[GF_LOG[a] + GF_LOG[b]];
    }

    function rsGenPoly(nsym) {
        let g = new Uint8Array(nsym + 1);
        g[0] = 1;
        for (let i = 0; i < nsym; i++) {
            for (let j = nsym; j > 0; j--) {
                g[j] = gfMul(g[j], GF_EXP[i]) ^ g[j - 1];
            }
            g[0] = gfMul(g[0], GF_EXP[i]);
        }
        return g;
    }

    function rsEncode(data, nsym) {
        const gen = rsGenPoly(nsym);
        const res = new Uint8Array(data.length + nsym);
        res.set(data);
        for (let i = 0; i < data.length; i++) {
            const coef = res[i];
            if (coef !== 0) {
                for (let j = 0; j <= nsym; j++) {
                    res[i + j] ^= gfMul(gen[j], coef);
                }
            }
        }
        return res.slice(data.length);
    }

    function getVersion(len) {
        for (let v = 1; v <= 40; v++) {
            if (VERSION_CAPACITY[v] >= len) return v;
        }
        throw new Error('Data too long');
    }

    function getSize(version) {
        return 17 + version * 4;
    }

    function encodeData(text, version) {
        const bytes = new TextEncoder().encode(text);
        const totalCodewords = VERSION_CAPACITY[version] + EC_CODEWORDS[version] * NUM_BLOCKS[version];
        const dataCodewords = VERSION_CAPACITY[version];

        // Build bit stream
        let bits = '';
        bits += '0100'; // Byte mode indicator
        const ccLen = version < 10 ? 8 : 16;
        bits += bytes.length.toString(2).padStart(ccLen, '0');
        for (const b of bytes) {
            bits += b.toString(2).padStart(8, '0');
        }
        bits += '0000'; // Terminator

        // Pad to byte boundary
        while (bits.length % 8 !== 0) bits += '0';

        // Convert to bytes
        const data = [];
        for (let i = 0; i < bits.length; i += 8) {
            data.push(parseInt(bits.slice(i, i + 8), 2));
        }

        // Pad to fill data capacity
        let pad = 0xec;
        while (data.length < dataCodewords) {
            data.push(pad);
            pad = pad === 0xec ? 0x11 : 0xec;
        }

        // Add error correction
        const numBlocks = NUM_BLOCKS[version];
        const ecCodewords = EC_CODEWORDS[version];
        const blockSize = Math.floor(dataCodewords / numBlocks);
        const largeBlocks = dataCodewords % numBlocks;

        const blocks = [];
        const ecBlocks = [];
        let offset = 0;

        for (let i = 0; i < numBlocks; i++) {
            const size = blockSize + (i >= numBlocks - largeBlocks ? 1 : 0);
            const block = new Uint8Array(data.slice(offset, offset + size));
            blocks.push(block);
            ecBlocks.push(rsEncode(block, ecCodewords));
            offset += size;
        }

        // Interleave blocks
        const result = [];
        const maxBlockLen = blockSize + 1;
        for (let i = 0; i < maxBlockLen; i++) {
            for (const block of blocks) {
                if (i < block.length) result.push(block[i]);
            }
        }
        for (let i = 0; i < ecCodewords; i++) {
            for (const block of ecBlocks) {
                result.push(block[i]);
            }
        }

        return new Uint8Array(result);
    }

    function createMatrix(version) {
        const size = getSize(version);
        const matrix = [];
        const reserved = [];
        for (let i = 0; i < size; i++) {
            matrix.push(new Array(size).fill(null));
            reserved.push(new Array(size).fill(false));
        }
        return { matrix, reserved, size };
    }

    function addFinderPattern(matrix, reserved, row, col) {
        for (let r = -1; r <= 7; r++) {
            for (let c = -1; c <= 7; c++) {
                const y = row + r, x = col + c;
                if (y < 0 || y >= matrix.length || x < 0 || x >= matrix.length) continue;
                let val;
                if (r === -1 || r === 7 || c === -1 || c === 7) {
                    val = 0; // Separator
                } else if (r === 0 || r === 6 || c === 0 || c === 6) {
                    val = 1;
                } else if (r >= 2 && r <= 4 && c >= 2 && c <= 4) {
                    val = 1;
                } else {
                    val = 0;
                }
                matrix[y][x] = val;
                reserved[y][x] = true;
            }
        }
    }

    function addAlignmentPattern(matrix, reserved, row, col) {
        for (let r = -2; r <= 2; r++) {
            for (let c = -2; c <= 2; c++) {
                const y = row + r, x = col + c;
                if (reserved[y][x]) return;
            }
        }
        for (let r = -2; r <= 2; r++) {
            for (let c = -2; c <= 2; c++) {
                const y = row + r, x = col + c;
                let val;
                if (r === -2 || r === 2 || c === -2 || c === 2 || (r === 0 && c === 0)) {
                    val = 1;
                } else {
                    val = 0;
                }
                matrix[y][x] = val;
                reserved[y][x] = true;
            }
        }
    }

    const ALIGNMENT_POSITIONS = [
        [], [], [6, 18], [6, 22], [6, 26], [6, 30], [6, 34],
        [6, 22, 38], [6, 24, 42], [6, 26, 46], [6, 28, 50],
        [6, 30, 54], [6, 32, 58], [6, 34, 62], [6, 26, 46, 66],
        [6, 26, 48, 70], [6, 26, 50, 74], [6, 30, 54, 78],
        [6, 30, 56, 82], [6, 30, 58, 86], [6, 34, 62, 90],
        [6, 28, 50, 72, 94], [6, 26, 50, 74, 98], [6, 30, 54, 78, 102],
        [6, 28, 54, 80, 106], [6, 32, 58, 84, 110], [6, 30, 58, 86, 114],
        [6, 34, 62, 90, 118], [6, 26, 50, 74, 98, 122], [6, 30, 54, 78, 102, 126],
        [6, 26, 52, 78, 104, 130], [6, 30, 56, 82, 108, 134],
        [6, 34, 60, 86, 112, 138], [6, 30, 58, 86, 114, 142],
        [6, 34, 62, 90, 118, 146], [6, 30, 54, 78, 102, 126, 150],
        [6, 24, 50, 76, 102, 128, 154], [6, 28, 54, 80, 106, 132, 158],
        [6, 32, 58, 84, 110, 136, 162], [6, 26, 54, 82, 110, 138, 166],
        [6, 30, 58, 86, 114, 142, 170]
    ];

    function addTimingPatterns(matrix, reserved, size) {
        for (let i = 8; i < size - 8; i++) {
            const val = (i + 1) % 2;
            if (!reserved[6][i]) {
                matrix[6][i] = val;
                reserved[6][i] = true;
            }
            if (!reserved[i][6]) {
                matrix[i][6] = val;
                reserved[i][6] = true;
            }
        }
    }

    function reserveFormatArea(matrix, reserved, size) {
        // Around top-left finder
        for (let i = 0; i < 9; i++) {
            reserved[8][i] = true;
            reserved[i][8] = true;
        }
        // Around top-right finder
        for (let i = 0; i < 8; i++) {
            reserved[8][size - 1 - i] = true;
        }
        // Around bottom-left finder
        for (let i = 0; i < 7; i++) {
            reserved[size - 1 - i][8] = true;
        }
        // Dark module
        matrix[size - 8][8] = 1;
        reserved[size - 8][8] = true;
    }

    function reserveVersionArea(reserved, size, version) {
        if (version < 7) return;
        for (let i = 0; i < 6; i++) {
            for (let j = 0; j < 3; j++) {
                reserved[i][size - 11 + j] = true;
                reserved[size - 11 + j][i] = true;
            }
        }
    }

    function placeData(matrix, reserved, data, size) {
        let bitIndex = 0;
        let up = true;
        for (let col = size - 1; col >= 0; col -= 2) {
            if (col === 6) col = 5; // Skip timing pattern column
            for (let row = up ? size - 1 : 0; up ? row >= 0 : row < size; row += up ? -1 : 1) {
                for (let c = 0; c < 2; c++) {
                    const x = col - c;
                    if (reserved[row][x]) continue;
                    if (bitIndex < data.length * 8) {
                        const byte = data[Math.floor(bitIndex / 8)];
                        const bit = (byte >> (7 - (bitIndex % 8))) & 1;
                        matrix[row][x] = bit;
                        bitIndex++;
                    } else {
                        matrix[row][x] = 0;
                    }
                }
            }
            up = !up;
        }
    }

    function applyMask(matrix, reserved, size, maskNum) {
        const maskFn = [
            (r, c) => (r + c) % 2 === 0,
            (r, c) => r % 2 === 0,
            (r, c) => c % 3 === 0,
            (r, c) => (r + c) % 3 === 0,
            (r, c) => (Math.floor(r / 2) + Math.floor(c / 3)) % 2 === 0,
            (r, c) => ((r * c) % 2) + ((r * c) % 3) === 0,
            (r, c) => (((r * c) % 2) + ((r * c) % 3)) % 2 === 0,
            (r, c) => (((r + c) % 2) + ((r * c) % 3)) % 2 === 0
        ][maskNum];

        for (let r = 0; r < size; r++) {
            for (let c = 0; c < size; c++) {
                if (!reserved[r][c] && maskFn(r, c)) {
                    matrix[r][c] ^= 1;
                }
            }
        }
    }

    function addFormatInfo(matrix, size, maskNum) {
        const FORMAT_BITS = [
            0x77c4, 0x72f3, 0x7daa, 0x789d, 0x662f, 0x6318, 0x6c41, 0x6976
        ];
        const bits = FORMAT_BITS[maskNum];

        // Place format info
        const positions = [
            [[8, 0], [8, 1], [8, 2], [8, 3], [8, 4], [8, 5], [8, 7], [8, 8],
             [7, 8], [5, 8], [4, 8], [3, 8], [2, 8], [1, 8], [0, 8]],
            [[size - 1, 8], [size - 2, 8], [size - 3, 8], [size - 4, 8],
             [size - 5, 8], [size - 6, 8], [size - 7, 8],
             [8, size - 8], [8, size - 7], [8, size - 6], [8, size - 5],
             [8, size - 4], [8, size - 3], [8, size - 2], [8, size - 1]]
        ];

        for (const posSet of positions) {
            for (let i = 0; i < 15; i++) {
                const [r, c] = posSet[i];
                matrix[r][c] = (bits >> (14 - i)) & 1;
            }
        }
    }

    function addVersionInfo(matrix, size, version) {
        if (version < 7) return;
        const VERSION_BITS = [
            0, 0, 0, 0, 0, 0, 0,
            0x07c94, 0x085bc, 0x09a99, 0x0a4d3, 0x0bbf6, 0x0c762, 0x0d847, 0x0e60d,
            0x0f928, 0x10b78, 0x1145d, 0x12a17, 0x13532, 0x149a6, 0x15683, 0x168c9,
            0x177ec, 0x18ec4, 0x191e1, 0x1afab, 0x1b08e, 0x1cc1a, 0x1d33f, 0x1ed75,
            0x1f250, 0x209d5, 0x216f0, 0x228ba, 0x2379f, 0x24b0b, 0x2542e, 0x26a64,
            0x27541, 0x28c69
        ];
        const bits = VERSION_BITS[version];

        for (let i = 0; i < 18; i++) {
            const bit = (bits >> i) & 1;
            const r = Math.floor(i / 3);
            const c = i % 3;
            matrix[r][size - 11 + c] = bit;
            matrix[size - 11 + c][r] = bit;
        }
    }

    function calculatePenalty(matrix, size) {
        let penalty = 0;

        // Rule 1: Consecutive modules in row/column
        for (let r = 0; r < size; r++) {
            let count = 1;
            for (let c = 1; c < size; c++) {
                if (matrix[r][c] === matrix[r][c - 1]) {
                    count++;
                } else {
                    if (count >= 5) penalty += count - 2;
                    count = 1;
                }
            }
            if (count >= 5) penalty += count - 2;
        }
        for (let c = 0; c < size; c++) {
            let count = 1;
            for (let r = 1; r < size; r++) {
                if (matrix[r][c] === matrix[r - 1][c]) {
                    count++;
                } else {
                    if (count >= 5) penalty += count - 2;
                    count = 1;
                }
            }
            if (count >= 5) penalty += count - 2;
        }

        // Rule 2: 2x2 blocks
        for (let r = 0; r < size - 1; r++) {
            for (let c = 0; c < size - 1; c++) {
                const val = matrix[r][c];
                if (val === matrix[r][c + 1] && val === matrix[r + 1][c] && val === matrix[r + 1][c + 1]) {
                    penalty += 3;
                }
            }
        }

        // Rule 3: Finder-like patterns
        const pattern1 = [1, 0, 1, 1, 1, 0, 1, 0, 0, 0, 0];
        const pattern2 = [0, 0, 0, 0, 1, 0, 1, 1, 1, 0, 1];
        for (let r = 0; r < size; r++) {
            for (let c = 0; c <= size - 11; c++) {
                let match1 = true, match2 = true;
                for (let i = 0; i < 11; i++) {
                    if (matrix[r][c + i] !== pattern1[i]) match1 = false;
                    if (matrix[r][c + i] !== pattern2[i]) match2 = false;
                }
                if (match1 || match2) penalty += 40;
            }
        }
        for (let c = 0; c < size; c++) {
            for (let r = 0; r <= size - 11; r++) {
                let match1 = true, match2 = true;
                for (let i = 0; i < 11; i++) {
                    if (matrix[r + i][c] !== pattern1[i]) match1 = false;
                    if (matrix[r + i][c] !== pattern2[i]) match2 = false;
                }
                if (match1 || match2) penalty += 40;
            }
        }

        // Rule 4: Dark/light balance
        let dark = 0;
        for (let r = 0; r < size; r++) {
            for (let c = 0; c < size; c++) {
                if (matrix[r][c]) dark++;
            }
        }
        const ratio = Math.abs((dark * 100) / (size * size) - 50);
        penalty += Math.floor(ratio / 5) * 10;

        return penalty;
    }

    function generateQRMatrix(text) {
        const version = getVersion(new TextEncoder().encode(text).length);
        const size = getSize(version);
        const data = encodeData(text, version);

        const { matrix, reserved } = createMatrix(version);

        // Add finder patterns
        addFinderPattern(matrix, reserved, 0, 0);
        addFinderPattern(matrix, reserved, 0, size - 7);
        addFinderPattern(matrix, reserved, size - 7, 0);

        // Add alignment patterns
        const alignPos = ALIGNMENT_POSITIONS[version];
        for (const r of alignPos) {
            for (const c of alignPos) {
                addAlignmentPattern(matrix, reserved, r, c);
            }
        }

        // Add timing patterns
        addTimingPatterns(matrix, reserved, size);

        // Reserve format and version areas
        reserveFormatArea(matrix, reserved, size);
        reserveVersionArea(reserved, size, version);

        // Place data
        placeData(matrix, reserved, data, size);

        // Find best mask
        let bestMask = 0;
        let bestPenalty = Infinity;
        for (let mask = 0; mask < 8; mask++) {
            const testMatrix = matrix.map(row => [...row]);
            applyMask(testMatrix, reserved, size, mask);
            addFormatInfo(testMatrix, size, mask);
            const penalty = calculatePenalty(testMatrix, size);
            if (penalty < bestPenalty) {
                bestPenalty = penalty;
                bestMask = mask;
            }
        }

        // Apply best mask
        applyMask(matrix, reserved, size, bestMask);
        addFormatInfo(matrix, size, bestMask);
        addVersionInfo(matrix, size, version);

        return { matrix, size };
    }

    function matrixToSVG(matrix, size, pixelSize) {
        const svgSize = size * pixelSize;
        const margin = pixelSize * 2;
        const totalSize = svgSize + margin * 2;

        let svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${totalSize} ${totalSize}" width="${totalSize}" height="${totalSize}">`;
        svg += `<rect width="${totalSize}" height="${totalSize}" fill="white"/>`;

        for (let r = 0; r < size; r++) {
            for (let c = 0; c < size; c++) {
                if (matrix[r][c]) {
                    svg += `<rect x="${margin + c * pixelSize}" y="${margin + r * pixelSize}" width="${pixelSize}" height="${pixelSize}" fill="black"/>`;
                }
            }
        }

        svg += '</svg>';
        return svg;
    }

    // Expose the function globally
    window.generateQRCode = function(text, displaySize) {
        const { matrix, size } = generateQRMatrix(text);
        const pixelSize = Math.max(1, Math.floor(displaySize / (size + 4)));
        return matrixToSVG(matrix, size, pixelSize);
    };
})();
