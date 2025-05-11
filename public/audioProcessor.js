// linear-pcm-processor.js (must be in its own file!)
class LinearPCMProcessor extends AudioWorkletProcessor {
    // The size of the buffer. Must be a multiple of 128 (the number of frames in the
    // input channel). An audio block is posted to the main thread every time the
    // buffer is full, which means a large buffer will emit less frequently (higher
    // latency), but more efficiently (fewer I/O interruptions between the worker and
    // the main thread)
    static BUFFER_SIZE = 16;

    constructor() {
        super();
        this.buffer = new Int16Array(LinearPCMProcessor.BUFFER_SIZE);
        this.offset = 0;
    }

    /**
     * Converts input data from Float32Array to Int16Array, and stores it to
     * to the buffer. When the buffer is full, its content is posted to the main
     * thread, and the buffer is emptied
     */
    process(inputList, outputList, _parameters) {
        // Assumes the input is mono (1 channel). If there are more channels, they
        // are ignored
        const input = inputList[0][0]; // first channel of first input
        const outputLeft = outputList[0][0]; // first channel of first input
        const outputRight = outputList[0][1]; // first channel of first input

        for (let i = 0; i < input.length; i++) {
            const sample = Math.max(-1, Math.min(1, input[i]));
            const normalized = sample < 0 ? sample * 0x8000 : sample * 0x7fff;
            // console.log(normalized)
            this.buffer[i + this.offset] = normalized
            // This will echo audio back out - stereo
            outputLeft[i] = input[i]
            outputRight[i] = input[i]
        }
        this.offset += input.length;

        // Once the buffer is filled entirely, flush the buffer
        if (this.offset >= this.buffer.length - 1) {
            this.flush();
        }
        return true;
    }

    /**
     * Sends the buffer's content to the main thread via postMessage(), and reset
     * the offset to 0
     */
    flush() {
        this.offset = 0;
        this.port.postMessage(this.buffer.slice());
    }
}
registerProcessor("pcm-processor", LinearPCMProcessor);
