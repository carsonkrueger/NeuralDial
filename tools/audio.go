package tools

func Int16ToWAV(data []int16, sampleRate int) []byte {
	const numChannels = 1
	const bitsPerSample = 16
	const bytesPerSample = bitsPerSample / 8

	dataSize := len(data) * bytesPerSample
	fileSize := 36 + dataSize
	byteRate := sampleRate * numChannels * bytesPerSample
	blockAlign := numChannels * bytesPerSample

	buffer := make([]byte, 44+dataSize)

	// Header
	copy(buffer[0:], []byte("RIFF"))
	putLE32(buffer[4:], uint32(fileSize))
	copy(buffer[8:], []byte("WAVE"))
	copy(buffer[12:], []byte("fmt "))
	putLE32(buffer[16:], 16) // Subchunk1Size
	putLE16(buffer[20:], 1)  // PCM format
	putLE16(buffer[22:], numChannels)
	putLE32(buffer[24:], uint32(sampleRate))
	putLE32(buffer[28:], uint32(byteRate))
	putLE16(buffer[32:], uint16(blockAlign))
	putLE16(buffer[34:], bitsPerSample)
	copy(buffer[36:], []byte("data"))
	putLE32(buffer[40:], uint32(dataSize))

	// PCM data
	for i, sample := range data {
		putLE16(buffer[44+i*2:], uint16(sample))
	}

	return buffer
}

func putLE16(buf []byte, val uint16) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
}

func putLE32(buf []byte, val uint32) {
	buf[0] = byte(val)
	buf[1] = byte(val >> 8)
	buf[2] = byte(val >> 16)
	buf[3] = byte(val >> 24)
}
