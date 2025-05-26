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

function decodeAndScheduleWav(base64Audio) {
    if (!base64Audio) {
        console.warn("Empty audio input");
        return;
    }

    // const arrayBuffer = base64ToArrayBuffer();
    const arrayBuffer = base64ToArrayBuffer(base64Audio);

    audioContext.decodeAudioData(arrayBuffer, (audioBuffer) => {
        const source = audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(audioContext.destination);

        const now = audioContext.currentTime;
        if (lastPlaybackTime < now) {
            lastPlaybackTime = now;
        }

        source.start(lastPlaybackTime);
        lastPlaybackTime += audioBuffer.duration;
    }, (error) => {
        console.error("Failed to decode audio:", error);
    });
}

function base64ToArrayBuffer(base64) {
    // If the string has a data URI prefix, strip it
    base64 = base64.includes(",") ? base64.split(",")[1] : base64;
    const binary = atob(base64);
    const len = binary.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
}

function pcmToWav(pcmData, sampleRate = 44100, numChannels = 1, bitsPerSample = 16) {
    const bytesPerSample = bitsPerSample / 8;
    const blockAlign = numChannels * bytesPerSample;
    const byteRate = sampleRate * blockAlign;
    const dataSize = pcmData.length * bytesPerSample;
    const buffer = new ArrayBuffer(44 + dataSize);
    const view = new DataView(buffer);

    function writeString(view, offset, str) {
        for (let i = 0; i < str.length; i++) {
            view.setUint8(offset + i, str.charCodeAt(i));
        }
    }

    // RIFF header
    writeString(view, 0, 'RIFF');
    view.setUint32(4, 36 + dataSize, true);
    writeString(view, 8, 'WAVE');

    // fmt subchunk
    writeString(view, 12, 'fmt ');
    view.setUint32(16, 16, true); // Subchunk1Size (PCM)
    view.setUint16(20, 1, true); // Audio format (1 = PCM)
    view.setUint16(22, numChannels, true);
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, byteRate, true);
    view.setUint16(32, blockAlign, true);
    view.setUint16(34, bitsPerSample, true);

    // data subchunk
    writeString(view, 36, 'data');
    view.setUint32(40, dataSize, true);

    // PCM data
    let offset = 44;
    for (let i = 0; i < pcmData.length; i++) {
        view.setInt16(offset, pcmData[i], true); // little-endian
        offset += 2;
    }

    return buffer;
}

function writeString(view, offset, string) {
    for (let i = 0; i < string.length; i++) {
        view.setUint8(offset + i, string.charCodeAt(i));
    }
}
