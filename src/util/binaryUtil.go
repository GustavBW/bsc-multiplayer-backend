package util

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
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

// writeValueToBytes writes a reflect.Value to a byte slice according to its kind
func WriteValueToBytes(dest []byte, value reflect.Value) error {
	kind := value.Kind()
	if SizeOfSerializedKind(kind) > uint32(len(dest)) {
		return fmt.Errorf("buffer underflow: (len %d) too small for %s", len(dest), kind)
	}

	switch value.Kind() {
	case reflect.Uint8:
		dest[0] = uint8(value.Uint())
	case reflect.Uint16:
		binary.BigEndian.PutUint16(dest, uint16(value.Uint()))
	case reflect.Uint32:
		binary.BigEndian.PutUint32(dest, uint32(value.Uint()))
	case reflect.Uint64:
		binary.BigEndian.PutUint64(dest, value.Uint())
	case reflect.Int8:
		dest[0] = uint8(value.Int())
	case reflect.Int16:
		binary.BigEndian.PutUint16(dest, uint16(value.Int()))
	case reflect.Int32:
		binary.BigEndian.PutUint32(dest, uint32(value.Int()))
	case reflect.Int64:
		binary.BigEndian.PutUint64(dest, uint64(value.Int()))
	case reflect.Float32:
		binary.BigEndian.PutUint32(dest, math.Float32bits(float32(value.Float())))
	case reflect.Float64:
		binary.BigEndian.PutUint64(dest, math.Float64bits(value.Float()))
	case reflect.String:
		strBytes := []byte(value.String())
		copy(dest, strBytes)
	default:
		return fmt.Errorf("unsupported type: %s", kind)
	}

	return nil
}

// SizeOfSerializedKind returns the size in bytes of the type when serialized.
//
// Returns 0 for variable length kinds like Array, Slice, String
func SizeOfSerializedKind(kind reflect.Kind) uint32 {
	switch kind {
	case reflect.Bool:
		return 1
	case reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64, reflect.Complex64:
		return 8
	case reflect.Complex128:
		return 16
	case reflect.String, reflect.Slice, reflect.Array:
		return 0
	default:
		return 0
	}
}
