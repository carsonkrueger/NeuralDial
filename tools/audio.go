package tools

import "time"

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

// Duration of PCM audio in milliseconds
func PCMDuration(pcmLength int, sampleRate int, channels int, bitsPerSample int) time.Duration {
	bytesPerSecond := float64(sampleRate * channels * (bitsPerSample / 8))
	seconds := float64(pcmLength) / bytesPerSecond
	return time.Duration(seconds*1000) * time.Millisecond
}

func BasicWAVHeader() []byte {
	return []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x00, 0x00, 0x00, 0x00, // Placeholder for file size
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6d, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Chunk size (16)
		0x01, 0x00, // Audio format (1 for PCM)
		0x01, 0x00, // Number of channels (1)
		0x80, 0x3e, 0x00, 0x00, // Sample rate (16000)
		0x00, 0x7d, 0x00, 0x00, // Byte rate (16000 * 2)
		0x02, 0x00, // Block align (2)
		0x10, 0x00, // Bits per sample (16)
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Placeholder for data size
	}
}
