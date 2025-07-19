class LinearPCMPlayer extends AudioWorkletProcessor {
    constructor() {
        super();
        this.bufferQueue = [];
        this.readOffset = 0;
        this.currentBuffer = null;

        this.port.onmessage = (e) => {
            switch (e.data) {
                case 'clear':
                    console.log('Clearing buffer');
                    this.clear();
                    break;
                default:
                    console.log(e.data.length)
                    const pcmData = new Int16Array(e.data);
                    this.bufferQueue.push(pcmData);
                    break;
            }
        };
    }

    process(_inputs, outputs, _parameters) {
        const output = outputs[0][0]; // mono output

        for (let i = 0; i < output.length; i++) {
            if (!this.currentBuffer || this.readOffset >= this.currentBuffer.length) {
                if (this.bufferQueue.length > 0) {
                    this.currentBuffer = this.bufferQueue.shift();
                    this.readOffset = 0;
                } else {
                    // If no data, output silence
                    output[i] = 0;
                    continue;
                }
            }

            const sample = this.currentBuffer[this.readOffset++];
            // Convert Int16 [-32768, 32767] to Float32 [-1.0, 1.0]
            output[i] = sample < 0 ? sample / 32768 : sample / 32767;
        }

        return true; // keep processor alive
    }

    clear() {
        this.bufferQueue = [];
        this.currentBuffer = null;
        this.readOffset = 0;
    }
}

registerProcessor("pcm-player", LinearPCMPlayer);
