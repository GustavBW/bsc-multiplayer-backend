package util

import "encoding/binary"

func BytesOfUint32(value uint32) []byte {
	var arr = []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(arr, value)
	return arr
}

func CopyAndAppend(arr1 []byte, arr2 []byte) []byte {
	var arr = make([]byte, len(arr1)+len(arr2))
	copy(arr, arr1)
	copy(arr[len(arr1):], arr2)
	return arr
}
