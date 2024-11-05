package internal

import (
	"encoding/binary"
	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Test structs
type BasicMessage struct {
	Value uint32 `json:"value"`
}

type AllFixedTypesMessage struct {
	U8  uint8   `json:"u8"`
	U16 uint16  `json:"u16"`
	U32 uint32  `json:"u32"`
	U64 uint64  `json:"u64"`
	I8  int8    `json:"i8"`
	I16 int16   `json:"i16"`
	I32 int32   `json:"i32"`
	I64 int64   `json:"i64"`
	F32 float32 `json:"f32"`
	F64 float64 `json:"f64"`
}

type StringMessage struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

type DifferentTagsMessage struct {
	InternalID   uint32 `json:"id,omitempty"`
	InternalName string `json:"name"`
}

// Helper function to create test specifications
// Does not account for message header sender id
func createTestSpec[T any](id uint32, structure []ShortElementDescriptor) *EventSpecification[T] {
	minSize, computed := ComputeStructure("TestMessage", structure)
	return &EventSpecification[T]{
		ID:              id,
		IDBytes:         util.BytesOfUint32(id),
		ExpectedMinSize: minSize,
		Structure:       computed,
	}
}

// unsafeCastSpec performs type erasure on EventSpecification to convert it to EventSpecification[any]
func unsafeCastSpec[T any](spec *EventSpecification[T]) *EventSpecification[any] {
	// Create a pointer to EventSpecification[any]
	unsafePtr := reflect.NewAt(reflect.TypeOf((*EventSpecification[any])(nil)).Elem(),
		unsafe.Pointer(reflect.ValueOf(spec).Pointer()))
	return unsafePtr.Interface().(*EventSpecification[any])
}

func TestComputeMessageSize(t *testing.T) {
	tests := []struct {
		name        string
		spec        interface{}
		data        interface{}
		wantSize    uint32
		wantErr     bool
		errContains string
	}{
		{
			name: "basic uint32 message",
			spec: createTestSpec[BasicMessage](1, []ShortElementDescriptor{
				{FieldName: "value", Kind: reflect.Uint32},
			}),
			data:     BasicMessage{Value: 42},
			wantSize: 4, //  4 (uint32)
			wantErr:  false,
		},
		{
			name: "all fixed types",
			spec: createTestSpec[AllFixedTypesMessage](1, []ShortElementDescriptor{
				{FieldName: "u8", Kind: reflect.Uint8},    // 1 byte
				{FieldName: "u16", Kind: reflect.Uint16},  // 2 bytes
				{FieldName: "u32", Kind: reflect.Uint32},  // 4 bytes
				{FieldName: "u64", Kind: reflect.Uint64},  // 8 bytes
				{FieldName: "i8", Kind: reflect.Int8},     // 1 byte
				{FieldName: "i16", Kind: reflect.Int16},   // 2 bytes
				{FieldName: "i32", Kind: reflect.Int32},   // 4 bytes
				{FieldName: "i64", Kind: reflect.Int64},   // 8 bytes
				{FieldName: "f32", Kind: reflect.Float32}, // 4 bytes
				{FieldName: "f64", Kind: reflect.Float64}, // 8 bytes
			}),
			data: AllFixedTypesMessage{
				U8: 1, U16: 2, U32: 3, U64: 4,
				I8: 5, I16: 6, I32: 7, I64: 8,
				F32: 9.0, F64: 10.0,
			},
			wantSize: 42, // all fields
			wantErr:  false,
		},
		{
			name: "message with string",
			spec: createTestSpec[StringMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data:     StringMessage{ID: 1, Name: "test"},
			wantSize: 8, // 4 (uint32) + 4 (string length)
			wantErr:  false,
		},
		{
			name: "message with omitempty tag",
			spec: createTestSpec[DifferentTagsMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data:     DifferentTagsMessage{InternalID: 1, InternalName: "test"},
			wantSize: 8, // 4 (uint32) + 4 (string length)
			wantErr:  false,
		},
		{
			name: "invalid field type",
			spec: createTestSpec[BasicMessage](1, []ShortElementDescriptor{
				{FieldName: "value", Kind: reflect.Int32}, // Doesn't match struct
			}),
			data:        BasicMessage{Value: 42},
			wantErr:     true,
			errContains: "expected kind",
		},
		{
			name: "missing field",
			spec: createTestSpec[BasicMessage](1, []ShortElementDescriptor{
				{FieldName: "nonexistent", Kind: reflect.Uint32},
			}),
			data:        BasicMessage{Value: 42},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use type assertion to get the original generic type
			var genericSpec interface{} = tt.spec
			var size uint32
			var err error

			// Handle different possible types
			switch s := genericSpec.(type) {
			case *EventSpecification[BasicMessage]:
				size, err = ComputeMessageSize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[AllFixedTypesMessage]:
				size, err = ComputeMessageSize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[StringMessage]:
				size, err = ComputeMessageSize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[DifferentTagsMessage]:
				size, err = ComputeMessageSize(unsafeCastSpec(s), tt.data)
			default:
				t.Fatalf("unhandled spec type: %T", s)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if size != tt.wantSize {
				t.Errorf("got size %d, want %d", size, tt.wantSize)
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	tests := []struct {
		name        string
		spec        interface{}
		data        interface{}
		validate    func(t *testing.T, result []byte)
		wantErr     bool
		errContains string
	}{
		{
			name: "basic uint32 message",
			spec: createTestSpec[BasicMessage](1, []ShortElementDescriptor{
				{FieldName: "value", Kind: reflect.Uint32},
			}),
			data: BasicMessage{Value: 42},
			validate: func(t *testing.T, result []byte) {
				if len(result) != 8 {
					t.Errorf("got length %d, want 8", len(result))
					return
				}
				if binary.BigEndian.Uint32(result[4:]) != 42 {
					t.Errorf("got value %d, want 42", binary.BigEndian.Uint32(result[4:]))
				}
			},
		},
		{
			name: "all numeric types",
			spec: createTestSpec[AllFixedTypesMessage](1, []ShortElementDescriptor{
				{FieldName: "u8", Kind: reflect.Uint8},
				{FieldName: "u16", Kind: reflect.Uint16},
				{FieldName: "u32", Kind: reflect.Uint32},
				{FieldName: "u64", Kind: reflect.Uint64},
				{FieldName: "i8", Kind: reflect.Int8},
				{FieldName: "i16", Kind: reflect.Int16},
				{FieldName: "i32", Kind: reflect.Int32},
				{FieldName: "i64", Kind: reflect.Int64},
				{FieldName: "f32", Kind: reflect.Float32},
				{FieldName: "f64", Kind: reflect.Float64},
			}),
			data: AllFixedTypesMessage{
				U8: 255, U16: 65535, U32: 4294967295, U64: 18446744073709551615,
				I8: -128, I16: -32768, I32: -2147483648, I64: -9223372036854775808,
				F32: math.Pi, F64: math.Pi,
			},
			validate: func(t *testing.T, result []byte) {
				offset := 4 // Skip header
				if result[offset] != 255 {
					t.Errorf("u8: got %d, want 255", result[offset])
				}
				offset++
				if binary.BigEndian.Uint16(result[offset:]) != 65535 {
					t.Errorf("u16: got %d, want 65535", binary.BigEndian.Uint16(result[offset:]))
				}
				offset += 2
				if binary.BigEndian.Uint32(result[offset:]) != 4294967295 {
					t.Errorf("u32: got %d, want 4294967295", binary.BigEndian.Uint32(result[offset:]))
				}
				offset += 4
				if binary.BigEndian.Uint64(result[offset:]) != 18446744073709551615 {
					t.Errorf("u64: got %d, want 18446744073709551615", binary.BigEndian.Uint64(result[offset:]))
				}
				offset += 8
				if int8(result[offset]) != -128 {
					t.Errorf("i8: got %d, want -128", int8(result[offset]))
				}
				offset++
				if int16(binary.BigEndian.Uint16(result[offset:])) != -32768 {
					t.Errorf("i16: got %d, want -32768", int16(binary.BigEndian.Uint16(result[offset:])))
				}
				offset += 2
				if int32(binary.BigEndian.Uint32(result[offset:])) != -2147483648 {
					t.Errorf("i32: got %d, want -2147483648", int32(binary.BigEndian.Uint32(result[offset:])))
				}
				offset += 4
				if int64(binary.BigEndian.Uint64(result[offset:])) != -9223372036854775808 {
					t.Errorf("i64: got %d, want -9223372036854775808", int64(binary.BigEndian.Uint64(result[offset:])))
				}
				offset += 8
				if math.Float32frombits(binary.BigEndian.Uint32(result[offset:])) != math.Pi {
					t.Errorf("f32: got %f, want %f", math.Float32frombits(binary.BigEndian.Uint32(result[offset:])), math.Pi)
				}
				offset += 4
				if math.Float64frombits(binary.BigEndian.Uint64(result[offset:])) != math.Pi {
					t.Errorf("f64: got %f, want %f", math.Float64frombits(binary.BigEndian.Uint64(result[offset:])), math.Pi)
				}
			},
		},
		{
			name: "string message",
			spec: createTestSpec[StringMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data: StringMessage{ID: 1, Name: "test"},
			validate: func(t *testing.T, result []byte) {
				if binary.BigEndian.Uint32(result[4:8]) != 1 {
					t.Errorf("got id %d, want 1", binary.BigEndian.Uint32(result[4:8]))
				}
				if string(result[8:]) != "test" {
					t.Errorf("got string %q, want \"test\"", string(result[8:]))
				}
			},
		},
		{
			name: "empty string",
			spec: createTestSpec[StringMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data: StringMessage{ID: 1, Name: ""},
			validate: func(t *testing.T, result []byte) {
				if len(result) != 8 { // header + uint32, empty string adds no bytes
					t.Errorf("got length %d, want 8", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var genericSpec interface{} = tt.spec
			var result []byte
			var err error

			switch s := genericSpec.(type) {
			case *EventSpecification[BasicMessage]:
				result, err = Serialize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[AllFixedTypesMessage]:
				result, err = Serialize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[StringMessage]:
				result, err = Serialize(unsafeCastSpec(s), tt.data)
			case *EventSpecification[DifferentTagsMessage]:
				result, err = Serialize(unsafeCastSpec(s), tt.data)
			default:
				t.Fatalf("unhandled spec type: %T", s)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			tt.validate(t, result)
		})
	}
}

// TestRoundTrip verifies that serializing and then deserializing returns the original data
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		spec interface{}
		data interface{}
	}{
		{
			name: "basic message",
			spec: createTestSpec[BasicMessage](1, []ShortElementDescriptor{
				{FieldName: "value", Kind: reflect.Uint32},
			}),
			data: BasicMessage{Value: 42},
		},
		{
			name: "string message",
			spec: createTestSpec[StringMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data: StringMessage{ID: 1, Name: "test"},
		},
		{
			name: "all fixed types",
			spec: createTestSpec[AllFixedTypesMessage](1, []ShortElementDescriptor{
				{FieldName: "u8", Kind: reflect.Uint8},
				{FieldName: "u16", Kind: reflect.Uint16},
				{FieldName: "u32", Kind: reflect.Uint32},
				{FieldName: "u64", Kind: reflect.Uint64},
				{FieldName: "i8", Kind: reflect.Int8},
				{FieldName: "i16", Kind: reflect.Int16},
				{FieldName: "i32", Kind: reflect.Int32},
				{FieldName: "i64", Kind: reflect.Int64},
				{FieldName: "f32", Kind: reflect.Float32},
				{FieldName: "f64", Kind: reflect.Float64},
			}),
			data: AllFixedTypesMessage{
				U8: 255, U16: 65535, U32: 4294967295, U64: 18446744073709551615,
				I8: -128, I16: -32768, I32: -2147483648, I64: -9223372036854775808,
				F32: math.Pi, F64: math.Pi,
			},
		},
		{
			name: "empty string message",
			spec: createTestSpec[StringMessage](1, []ShortElementDescriptor{
				{FieldName: "id", Kind: reflect.Uint32},
				{FieldName: "name", Kind: reflect.String},
			}),
			data: StringMessage{ID: 1, Name: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var genericSpec interface{} = tt.spec
			var serialized []byte
			var deserialized interface{}
			var err error

			// Serialize
			switch s := genericSpec.(type) {
			case *EventSpecification[BasicMessage]:
				serialized, err = Serialize(unsafeCastSpec(s), tt.data)
				if err != nil {
					t.Fatalf("failed to serialize: %v", err)
				}
				deserialized, err = Deserialize(unsafeCastSpec(s), serialized, true)
			case *EventSpecification[StringMessage]:
				serialized, err = Serialize(unsafeCastSpec(s), tt.data)
				if err != nil {
					t.Fatalf("failed to serialize: %v", err)
				}
				deserialized, err = Deserialize(unsafeCastSpec(s), serialized, true)
			case *EventSpecification[AllFixedTypesMessage]:
				serialized, err = Serialize(unsafeCastSpec(s), tt.data)
				if err != nil {
					t.Fatalf("failed to serialize: %v", err)
				}
				deserialized, err = Deserialize(unsafeCastSpec(s), serialized, true)
			default:
				t.Fatalf("unhandled spec type: %T", s)
			}

			if err != nil {
				t.Fatalf("failed to deserialize: %v", err)
			}

			// Compare the original data with the deserialized data
			if !reflect.DeepEqual(tt.data, deserialized) {
				t.Errorf("data changed during round trip:\noriginal:     %+v\ndeserialized: %+v",
					tt.data, deserialized)
			}
		})
	}
}

func TestVerySpecificComputeSize(t *testing.T) {
	data := AssignPlayerDataMessageDTO{
		ID:       1,
		X:        1,
		Y:        1,
		TankType: 1,
		CharCode: "Ch",
	}

	_, err := Serialize(ASSIGN_PLAYER_DATA_EVENT, data)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}
}
