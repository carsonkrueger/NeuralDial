package tools

// converts raw int16 audio data to WAV format
func Int16ToWAV(data []int16, sampleRate int, sampleSize int) []byte {
	// Calculate data size (in bytes)
	dataSize := len(data) * 2 // int16 = 2 bytes

	// Total file size = 44 (header) + data size
	fileSize := 36 + dataSize

	// Create buffer for the WAV file
	buffer := make([]byte, 44+dataSize)

	// RIFF header
	copy(buffer[0:4], []byte("RIFF"))
	buffer[4] = byte(fileSize)
	buffer[5] = byte(fileSize >> 8)
	buffer[6] = byte(fileSize >> 16)
	buffer[7] = byte(fileSize >> 24)
	copy(buffer[8:12], []byte("WAVE"))

	// Format chunk
	copy(buffer[12:16], []byte("fmt "))
	buffer[16] = 16 // Size of format chunk (16 bytes)
	buffer[17] = 0
	buffer[18] = 0
	buffer[19] = 0
	buffer[20] = 1 // Audio format (1 = PCM)
	buffer[21] = 0
	buffer[22] = 1 // Number of channels (1 = mono)
	buffer[23] = 0

	// Sample rate
	buffer[24] = byte(sampleRate)
	buffer[25] = byte(sampleRate >> 8)
	buffer[26] = byte(sampleRate >> 16)
	buffer[27] = byte(sampleRate >> 24)

	// Byte rate = SampleRate * NumChannels * BitsPerSample/8
	byteRate := sampleRate * 1 * sampleSize / 8
	buffer[28] = byte(byteRate)
	buffer[29] = byte(byteRate >> 8)
	buffer[30] = byte(byteRate >> 16)
	buffer[31] = byte(byteRate >> 24)

	// Block align = NumChannels * BitsPerSample/8
	blockAlign := 1 * sampleSize / 8
	buffer[32] = byte(blockAlign)
	buffer[33] = byte(blockAlign >> 8)

	// Bits per sample
	buffer[34] = byte(sampleSize)
	buffer[35] = byte(sampleSize >> 8)

	// Data chunk
	copy(buffer[36:40], []byte("data"))
	buffer[40] = byte(dataSize)
	buffer[41] = byte(dataSize >> 8)
	buffer[42] = byte(dataSize >> 16)
	buffer[43] = byte(dataSize >> 24)

	// Copy audio data
	for i, sample := range data {
		buffer[44+i*2] = byte(sample)
		buffer[44+i*2+1] = byte(sample >> 8)
	}

	return buffer
}
