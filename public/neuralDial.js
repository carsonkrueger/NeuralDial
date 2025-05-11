async function startSpeakingAudioStream(callback, sampleRate) {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
        const audioCtx = new AudioContext();
        const source = audioCtx.createMediaStreamSource(stream);
        await audioCtx.audioWorklet.addModule("/public/audioProcessor.js");
        const audioWorkletNode = new AudioWorkletNode(audioCtx, "pcm-processor");
        source.connect(audioWorkletNode);
        audioWorkletNode.connect(audioCtx.destination);

        const doDownSample = audioCtx.sampleRate > sampleRate
        // const doDownSample = false;

        audioWorkletNode.port.onmessage = (e) => {
            if (!isSpeakingInt16(e.data, 0.02)) return;
            if (doDownSample) {
                const downSampled = downSampleInt16Buffer(e.data, audioCtx.sampleRate, sampleRate)
                callback(downSampled);
            } else {
                callback(e.data);
            }
        }
    } catch (err) {
        console.error("Audio stream error:", err);
    }
}

// Function to detect sound above a threshold
function isSpeakingFloat32(floatData, threshold) {
    let sumSquares = 0;
    for (let i = 0; i < floatData.length; i++) {
        sumSquares += floatData[i] * floatData[i];
    }
    const rms = Math.sqrt(sumSquares / floatData.length);
    return rms > threshold;
}

function isSpeakingInt16(int16Data, threshold) {
    let sumSquares = 0;
    for (let i = 0; i < int16Data.length; i++) {
        const sample = int16Data[i] / 32768; // normalize to [-1, 1]
        sumSquares += sample * sample;
    }
    const rms = Math.sqrt(sumSquares / int16Data.length);
    return rms > threshold;
}

function convertUnsigned8ToFloat32(array) {
    const targetArray = new Float32Array(array.byteLength / 2);

    // A DataView is used to read our 16-bit little-endian samples
    // out of the Uint8Array buffer
    const sourceDataView = new DataView(array.buffer);

    // Loop through, get values, and divide by 32,768
    for (let i = 0; i < targetArray.length; i++) {
        targetArray[i] = sourceDataView.getInt16(i * 2, true) / Math.pow(2, 16 - 1);
    }
    return targetArray;
}

function convertFloat32ToUnsigned8(array) {
    const buffer = new ArrayBuffer(array.length * 2);
    const view = new DataView(buffer);

    for (let i = 0; i < array.length; i++) {
        const value = array[i] * 32768;
        view.setInt16(i * 2, value, true); // true for little-endian
    }

    return new Uint8Array(buffer);
}

function downSampleInt16Buffer(buffer, inputSampleRate, outputSampleRate) {
    if (outputSampleRate >= inputSampleRate) {
        throw new Error("Output sample rate must be lower than input sample rate.");
    }

    const sampleRateRatio = inputSampleRate / outputSampleRate;
    const newLength = Math.round(buffer.length / sampleRateRatio);
    const result = new Int16Array(newLength);

    let offset = 0;
    for (let i = 0; i < result.length; i++) {
        const nextOffset = Math.round((i + 1) * sampleRateRatio);
        // Average samples in the range
        let sum = 0, count = 0;
        for (let j = offset; j < nextOffset && j < buffer.length; j++) {
            sum += buffer[j];
            count++;
        }
        result[i] = sum / count;
        offset = nextOffset;
    }
    return result;
}
