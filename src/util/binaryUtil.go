package util

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math"
)

// Little Endian order
func BytesOfUint32(value uint32) []byte {
	var arr = []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(arr, value)
	return arr
}

func BytesOfFloat32(f float32) []byte {
	var arr = make([]byte, 4)
	binary.BigEndian.PutUint32(arr, math.Float32bits(f))
	return arr
}

// Copies both and appends them to a third slice which is returned
func CopyAndAppend[T any](arr1 []T, arr2 []T) []T {
	var arr = make([]T, len(arr1)+len(arr2))
	copy(arr, arr1)
	copy(arr[len(arr1):], arr2)
	return arr
}

func EncodeBase16(message []byte) []byte {
	dest := make([]byte, hex.EncodedLen(len(message)))
	hex.Encode(dest, message)
	return dest
}

func EncodeBase64(message []byte) []byte {
	// Create a destination slice with the appropriate length
	dest := make([]byte, base64.StdEncoding.EncodedLen(len(message)))

	// Encode the message in Base64 and store the result in dest
	base64.StdEncoding.Encode(dest, message)

	return dest
}
