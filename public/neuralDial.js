async function startSpeakingAudioStream(callback, sampleRate) {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
        const inputAudioCtx = new window.AudioContext();
        const inputSource = inputAudioCtx.createMediaStreamSource(stream);
        await inputAudioCtx.audioWorklet.addModule("/public/audioProcessor.js");
        const inputProcessor = new AudioWorkletNode(inputAudioCtx, "pcm-processor");
        inputSource.connect(inputProcessor);
        inputProcessor.connect(inputAudioCtx.destination);

        const doDownSample = inputAudioCtx.sampleRate > sampleRate

        inputProcessor.port.onmessage = (e) => {
            // if (!isSpeakingInt16(e.data, 0.02)) return;
            if (doDownSample) {
                const downSampled = downSampleInt16Buffer(e.data, inputAudioCtx.sampleRate, sampleRate)
                callback(downSampled);
            } else {
                callback(e.data);
            }
        }
    } catch (err) {
        console.error("Audio stream error:", err);
    }
}

async function setupAudioPlayerStream(sampleRate) {
    const outputAudioCtx = new window.AudioContext({ sampleRate });
    await outputAudioCtx.audioWorklet.addModule("/public/audioPlayer.js");
    const outputProcessor = new AudioWorkletNode(outputAudioCtx, "pcm-player");
    outputProcessor.connect(outputAudioCtx.destination);
    return outputProcessor.port;
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
