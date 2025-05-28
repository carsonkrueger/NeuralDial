package tools

import "unsafe"

func BytesToInt16Slice(b []byte) []int16 {
	if len(b)%2 != 0 {
		panic("byte slice length must be even to convert to []int16")
	}
	return unsafe.Slice((*int16)(unsafe.Pointer(&b[0])), len(b)/2)
}
