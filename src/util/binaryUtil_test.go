package util

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestBytesOfUint32(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected []byte
	}{
		{"Zero", 0, []byte{0, 0, 0, 0}},
		{"One", 1, []byte{1, 0, 0, 0}},
		{"Max uint32", 4294967295, []byte{255, 255, 255, 255}},
		{"Random value", 305419896, []byte{120, 86, 52, 18}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesOfUint32(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("BytesOfUint32(%d) = %v, want %v", tt.input, result, tt.expected)
			}

			// Verify Little Endian order
			reconstructed := binary.BigEndian.Uint32(result)
			if reconstructed != tt.input {
				t.Errorf("Reconstructed value %d does not match input %d", reconstructed, tt.input)
			}
		})
	}
}

func TestCopyAndAppend(t *testing.T) {
	tests := []struct {
		name     string
		arr1     []int
		arr2     []int
		expected []int
	}{
		{"Empty arrays", []int{}, []int{}, []int{}},
		{"First array empty", []int{}, []int{1, 2, 3}, []int{1, 2, 3}},
		{"Second array empty", []int{1, 2, 3}, []int{}, []int{1, 2, 3}},
		{"Both arrays non-empty", []int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CopyAndAppend(tt.arr1, tt.arr2)
			if !equalSlices(result, tt.expected) {
				t.Errorf("CopyAndAppend(%v, %v) = %v, want %v", tt.arr1, tt.arr2, result, tt.expected)
			}

			// Check if the original slices are unmodified
			if !equalSlices(tt.arr1, tt.arr1) || !equalSlices(tt.arr2, tt.arr2) {
				t.Errorf("Original slices were modified")
			}
		})
	}
}

func TestEncodeBase16(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"Empty input", []byte{}, []byte{}},
		{"Single byte", []byte{0xFF}, []byte("ff")},
		{"Multiple bytes", []byte{0x01, 0x23, 0x45, 0x67}, []byte("01234567")},
		{"ASCII string", []byte("Hello"), []byte("48656c6c6f")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeBase16(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("EncodeBase16(%v) = %s, want %s", tt.input, result, tt.expected)
			}

			// Verify decoding
			decoded, err := hex.DecodeString(string(result))
			if err != nil {
				t.Errorf("Failed to decode result: %v", err)
			}
			if !bytes.Equal(decoded, tt.input) {
				t.Errorf("Decoded result %v does not match input %v", decoded, tt.input)
			}
		})
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"Empty input", []byte{}, []byte{}},
		{"Single byte", []byte{0xFF}, []byte("/w==")},
		{"Multiple bytes", []byte{0x01, 0x23, 0x45, 0x67}, []byte("ASNFZw==")},
		{"ASCII string", []byte("Hello, World!"), []byte("SGVsbG8sIFdvcmxkIQ==")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeBase64(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("EncodeBase64(%v) = %s, want %s", tt.input, result, tt.expected)
			}

			// Verify decoding
			decoded, err := base64.StdEncoding.DecodeString(string(result))
			if err != nil {
				t.Errorf("Failed to decode result: %v", err)
			}
			if !bytes.Equal(decoded, tt.input) {
				t.Errorf("Decoded result %v does not match input %v", decoded, tt.input)
			}
		})
	}
}

// Helper function to compare slices
func equalSlices[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
