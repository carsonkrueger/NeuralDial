WsType = {
    AGENT_SPEAK: "agent_speak",
    AGENT_TRANSCRIBE: "agent_transcribe",
    USER_SPEAK: "user_speak"
}


class NeuralDial {
    started = false;

    startWebsocket() {
        if (this.started) return;
        this.started = true;

        let speakws = new WebSocket("/speak/ws");
        speakws.binaryType = "arraybuffer";
        const sampleRate = 16000;
        let audioPlayer = null;

        speakws.onopen = async () => {
            console.log("WebSocket connected");
            await this.startSpeakingAudioStream(sendData, sampleRate);
            audioPlayer = await this.setupAudioPlayerStream(sampleRate);
            console.log("WebSocket setup");
        };

        speakws.onmessage = (e) => {
            const jsonStr = new TextDecoder().decode(e.data);
            const msg = JSON.parse(jsonStr);
            switch (msg.type) {
                case WsType.AGENT_SPEAK:
                    console.log("AGENT");
                    audioPlayer.port.postMessage(base64ToInt16Array(msg.data));
                    break;
                case WsType.AGENT_TRANSCRIBE:
                    const bytes = new Uint8Array(base64ToUint8Array(msg.data));
                    const decoder = new TextDecoder();
                    console.log("AGENT TRANSCRIBE:", decoder.decode(bytes));
                    break;
                case WsType.USER_SPEAK:
                    console.log("USER");
                    audioPlayer.port.postMessage('clear');
                    break;
                default:
                    console.log("Received unknown message", msg);
                    break;
            }
        };

        function sendData(data) {
            speakws.send(data);
        }
    }

    async startSpeakingAudioStream(callback, sampleRate) {
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
                    const downSampled = this.downSampleInt16Buffer(e.data, inputAudioCtx.sampleRate, sampleRate)
                    callback(downSampled);
                } else {
                    callback(e.data);
                }
            }
        } catch (err) {
            console.error("Audio stream error:", err);
        }
    }

    async setupAudioPlayerStream(sampleRate) {
        const outputAudioCtx = new window.AudioContext({ sampleRate });
        await outputAudioCtx.audioWorklet.addModule("/public/audioPlayer.js");
        const outputProcessor = new AudioWorkletNode(outputAudioCtx, "pcm-player");
        outputProcessor.connect(outputAudioCtx.destination);
        return outputProcessor;
    }

    isSpeakingInt16(int16Data, threshold) {
        let sumSquares = 0;
        for (let i = 0; i < int16Data.length; i++) {
            const sample = int16Data[i] / 32768; // normalize to [-1, 1]
            sumSquares += sample * sample;
        }
        const rms = Math.sqrt(sumSquares / int16Data.length);
        return rms > threshold;
    }

    downSampleInt16Buffer(buffer, inputSampleRate, outputSampleRate) {
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
}


function base64ToUint8Array(base64) {
    const binaryStr = atob(base64);
    const len = binaryStr.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binaryStr.charCodeAt(i);
    }
    return bytes;
}

function base64ToInt16Array(base64) {
    const binaryStr = atob(base64);
    const len = binaryStr.length / 2; // 2 bytes per int16
    const buffer = new ArrayBuffer(binaryStr.length);
    const uint8View = new Uint8Array(buffer);

    for (let i = 0; i < binaryStr.length; i++) {
        uint8View[i] = binaryStr.charCodeAt(i);
    }

    // Create Int16Array over the same buffer, using little-endian interpretation
    // If your Go data is in little-endian (typical), this will align correctly.
    return new Int16Array(buffer);
}
