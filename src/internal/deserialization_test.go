package internal

import (
	"encoding/binary"
	"math"
	"reflect"
	"testing"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Not real data, will cause issues with anything else than deserialize.
// Simply just used for padding
var placeHolderIDAndEventIDBytes = []byte{
	0, 0, 0, 0, // Placeholder for user ID
	0, 0, 0, 0, // Event ID
}

func TestDeserializePlayerJoinedEvent(t *testing.T) {
	spec := PLAYER_JOINED_EVENT
	baseData := []byte{
		0, 0, 0, 42, // PlayerID
		72, 101, 108, 108, 111, // IGN: "Hello"
	}

	dataWHeader := util.CopyAndAppend(placeHolderIDAndEventIDBytes, baseData)
	expected := &PlayerJoinedEventDTO{
		PlayerID: 42,
		IGN:      "Hello",
	}

	result1, err := Deserialize(spec, dataWHeader, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result1, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result1, expected)
	}

	result, err := Deserialize(spec, baseData, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerLeftEvent(t *testing.T) {
	spec := PLAYER_LEFT_EVENT
	data := []byte{
		0, 0, 0, 24, // PlayerID
		71, 111, 111, 100, 98, 121, 101, // IGN: "Goodbye"
	}
	expected := &PlayerLeftEventDTO{
		PlayerID: 24,
		IGN:      "Goodbye",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeEnterLocationEvent(t *testing.T) {
	spec := ENTER_LOCATION_EVENT
	data := []byte{
		0, 0, 0, 7, // Colony Location ID
	}
	expected := &EnterLocationEventDTO{
		ID: 7,
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerMoveEvent(t *testing.T) {
	spec := PLAYER_MOVE_EVENT
	data := []byte{
		0, 0, 0, 10, // PlayerID
		0, 0, 0, 20, // ColonyLocationID
	}
	expected := &PlayerMoveEventDTO{
		PlayerID:         10,
		ColonyLocationID: 20,
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeDifficultySelectForMinigameEvent(t *testing.T) {
	spec := DIFFICULTY_SELECT_FOR_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 1, // MinigameID
		0, 0, 0, 2, // DifficultyID
		69, 97, 115, 121, // DifficultyName: "Easy"
	}
	expected := &DifficultySelectForMinigameEventDTO{
		MinigameID:     1,
		DifficultyID:   2,
		DifficultyName: "Easy",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerReadyEvent(t *testing.T) {
	spec := PLAYER_READY_EVENT
	data := []byte{
		0, 0, 0, 33, // PlayerID
		82, 101, 97, 100, 121, 80, 108, 97, 121, 101, 114, // IGN: "ReadyPlayer"
	}
	expected := &PlayerReadyEventDTO{
		PlayerID: 33,
		IGN:      "ReadyPlayer",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeDifficultyConfirmedForMinigameEvent(t *testing.T) {
	spec := DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 0, // Placeholder for user ID
		0, 0, 7, 209, // Event ID (2001 for DifficultyConfirmedForMinigame)
		0, 0, 0, 1, // MinigameID
		0, 0, 0, 3, // DifficultyID
		72, 97, 114, 100, // DifficultyName: "Hard"
	}
	expected := &DifficultyConfirmedForMinigameEventDTO{
		MinigameID:     1,
		DifficultyID:   3,
		DifficultyName: "Hard",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerAbortingMinigameEvent(t *testing.T) {
	spec := PLAYER_ABORTING_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 0, // Placeholder for user ID
		0, 0, 7, 212, // Event ID (2004 for PlayerAbortingMinigame)
		0, 0, 0, 42, // PlayerID
		65, 98, 111, 114, 116, 101, 114, // IGN: "Aborter"
	}
	expected := &PlayerAbortingMinigameEventDTO{
		PlayerID: 42,
		IGN:      "Aborter",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerJoinActivityEvent(t *testing.T) {
	spec := PLAYER_JOIN_ACTIVITY_EVENT
	data := []byte{
		0, 0, 0, 0, // Placeholder for user ID
		0, 0, 7, 214, // Event ID (2006 for PlayerJoinActivity)
		0, 0, 0, 55, // PlayerID
		74, 111, 105, 110, 101, 114, // IGN: "Joiner"
	}
	expected := &PlayerJoinActivityEventDTO{
		PlayerID: 55,
		IGN:      "Joiner",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeAbstractCases(t *testing.T) {
	tests := []struct {
		name        string
		spec        *EventSpecification[any]
		data        []byte
		expected    interface{}
		expectError bool
	}{
		{
			name: "Deserialize with all supported types",
			spec: &EventSpecification[any]{
				Structure: ComputedStructure{
					{ByteSize: 1, Offset: 8, FieldName: "BoolField", Kind: reflect.Bool},
					{ByteSize: 1, Offset: 9, FieldName: "Int8Field", Kind: reflect.Int8},
					{ByteSize: 2, Offset: 10, FieldName: "Int16Field", Kind: reflect.Int16},
					{ByteSize: 4, Offset: 12, FieldName: "Int32Field", Kind: reflect.Int32},
					{ByteSize: 8, Offset: 16, FieldName: "Int64Field", Kind: reflect.Int64},
					{ByteSize: 1, Offset: 24, FieldName: "Uint8Field", Kind: reflect.Uint8},
					{ByteSize: 2, Offset: 25, FieldName: "Uint16Field", Kind: reflect.Uint16},
					{ByteSize: 4, Offset: 27, FieldName: "Uint32Field", Kind: reflect.Uint32},
					{ByteSize: 8, Offset: 31, FieldName: "Uint64Field", Kind: reflect.Uint64},
					{ByteSize: 4, Offset: 39, FieldName: "Float32Field", Kind: reflect.Float32},
					{ByteSize: 8, Offset: 43, FieldName: "Float64Field", Kind: reflect.Float64},
					{ByteSize: 0, Offset: 51, FieldName: "StringField", Kind: reflect.String},
				},
			},
			data: func() []byte {
				data := make([]byte, 60)
				data[8] = 1                                                            // Bool: true
				data[9] = 0xFF                                                         // Int8: -1
				binary.BigEndian.PutUint16(data[10:], 0x7FFF)                          // Int16: 32767
				binary.BigEndian.PutUint32(data[12:], 0x7FFFFFFF)                      // Int32: 2147483647
				binary.BigEndian.PutUint64(data[16:], 0x7FFFFFFFFFFFFFFF)              // Int64: 9223372036854775807
				data[24] = 0xFF                                                        // Uint8: 255
				binary.BigEndian.PutUint16(data[25:], 0xFFFF)                          // Uint16: 65535
				binary.BigEndian.PutUint32(data[27:], 0xFFFFFFFF)                      // Uint32: 4294967295
				binary.BigEndian.PutUint64(data[31:], 0xFFFFFFFFFFFFFFFF)              // Uint64: 18446744073709551615
				binary.BigEndian.PutUint32(data[39:], math.Float32bits(3.14))          // Float32: 3.14
				binary.BigEndian.PutUint64(data[43:], math.Float64bits(3.14159265359)) // Float64: 3.14159265359
				copy(data[51:], "Hello, World!")                                       // String: "Hello, World!"
				return data
			}(),
			expected: &struct {
				BoolField    bool
				Int8Field    int8
				Int16Field   int16
				Int32Field   int32
				Int64Field   int64
				Uint8Field   uint8
				Uint16Field  uint16
				Uint32Field  uint32
				Uint64Field  uint64
				Float32Field float32
				Float64Field float64
				StringField  string
			}{
				BoolField:    true,
				Int8Field:    -1,
				Int16Field:   32767,
				Int32Field:   2147483647,
				Int64Field:   9223372036854775807,
				Uint8Field:   255,
				Uint16Field:  65535,
				Uint32Field:  4294967295,
				Uint64Field:  18446744073709551615,
				Float32Field: 3.14,
				Float64Field: 3.14159265359,
				StringField:  "Hello, World!",
			},
		},
		{
			name: "Deserialize with insufficient data",
			spec: &EventSpecification[any]{
				Structure: ComputedStructure{
					{ByteSize: 4, Offset: 8, FieldName: "Field1", Kind: reflect.Uint32},
					{ByteSize: 4, Offset: 12, FieldName: "Field2", Kind: reflect.Uint32},
				},
			},
			data: []byte{
				0, 0, 0, 10, // Field1
			},
			expectError: true,
		},
		{
			name: "Deserialize with mismatched field types",
			spec: &EventSpecification[any]{
				Structure: ComputedStructure{
					{ByteSize: 4, Offset: 8, FieldName: "Field1", Kind: reflect.Int32},
					{ByteSize: 4, Offset: 12, FieldName: "Field2", Kind: reflect.Uint32},
				},
			},
			data: []byte{
				0, 0, 0, 10, // Field1
				0, 0, 0, 20, // Field2
			},
			expectError: true,
		},
		{
			name: "Deserialize with empty string",
			spec: &EventSpecification[any]{
				Structure: ComputedStructure{
					{ByteSize: 4, Offset: 8, FieldName: "IntField", Kind: reflect.Int32},
					{ByteSize: 0, Offset: 12, FieldName: "StringField", Kind: reflect.String},
				},
			},
			data: []byte{
				0, 0, 0, 42, // IntField
				// No data for StringField (empty string)
			},
			expected: &struct {
				IntField    int32
				StringField string
			}{
				IntField:    42,
				StringField: "",
			},
		},
		{
			name: "Deserialize with Unicode string",
			spec: &EventSpecification[any]{
				Structure: ComputedStructure{
					{ByteSize: 4, Offset: 8, FieldName: "IntField", Kind: reflect.Int32},
					{ByteSize: 0, Offset: 12, FieldName: "StringField", Kind: reflect.String},
				},
			},
			data: []byte{
				0, 0, 0, 42, // IntField
				240, 159, 152, 138, // Unicode: 😊 (smiling face with smiling eyes)
			},
			expected: &struct {
				IntField    int32
				StringField string
			}{
				IntField:    42,
				StringField: "😊",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Deserialize(tt.spec, tt.data, true)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, tt.expected)
				}
			}
		})
	}
}
