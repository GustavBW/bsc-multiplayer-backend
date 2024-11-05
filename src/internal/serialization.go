package internal

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Serializes the provided data according to the specification
// Includes the event id part of the header prefixed (i.e. only needs the sender id to be appended before sending)
func Serialize[T any](spec *EventSpecification[T], data T) ([]byte, error) {
	// Calculate the exact size needed
	messageSize, err := ComputeMessageSize(spec, data)
	if err != nil {
		return nil, fmt.Errorf("failed to compute message size: %s", err.Error())
	}

	// Validate against minimum size
	if messageSize < spec.ExpectedMinSize {
		return nil, fmt.Errorf("computed message size %d is less than expected minimum size %d",
			messageSize, spec.ExpectedMinSize)
	}

	// Pre-allocate the exact size needed
	message := make([]byte, 0, messageSize)

	// Add event ID header
	message = append(message, spec.IDBytes...)

	// Get the value of data
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Create a buffer for binary encoding
	buffer := make([]byte, 8)

	// Serialize fields according to spec
	for _, element := range spec.Structure {
		field, _ := util.FindFieldByJSONTagValue(v, element.FieldName)
		// We don't need to check found because we already validated in ComputeMessageSize

		message, err = appendValue(message, buffer, field)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %s", element.FieldName, err.Error())
		}
	}

	return message, nil
}

// ComputeMessageSize calculates the total size in bytes this message will occupy when serialized
// Not including message header
func ComputeMessageSize[T any](spec *EventSpecification[T], data T) (uint32, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("expected a struct, got %s", v.Kind())
	}
	var size uint32 = 0
	// Go through spec structure to calculate size
	for i, element := range spec.Structure {
		// Find field by JSON tag name
		field, found := util.FindFieldByJSONTagValue(v, element.FieldName)
		if !found {
			return 0, fmt.Errorf("field with JSON tag '%s' not found in struct", element.FieldName)
		}
		// Verify field type matches specification
		if field.Kind() != element.Kind {
			return 0, fmt.Errorf("field '%s': expected kind %s, got %s",
				element.FieldName, element.Kind, field.Kind())
		}

		if element.ByteSize == 0 {
			// Variable size field (currently only strings)
			if element.Kind != reflect.String {
				return 0, fmt.Errorf("unsupported variable size field type: %s", element.Kind)
			}
			// Variable size fields must be at the end
			if i != len(spec.Structure)-1 {
				return 0, fmt.Errorf("variable size field '%s' must be the last field",
					element.FieldName)
			}
			// Add string length
			size += uint32(len(field.String()))
		} else {
			// Fixed size field
			size += element.ByteSize
		}
	}

	return size, nil
}

func appendValue(message []byte, buffer []byte, value reflect.Value) ([]byte, error) {
	switch value.Kind() {
	case reflect.Uint8:
		return append(message, uint8(value.Uint())), nil

	case reflect.Uint16:
		binary.BigEndian.PutUint16(buffer, uint16(value.Uint()))
		return append(message, buffer[:2]...), nil

	case reflect.Uint32:
		binary.BigEndian.PutUint32(buffer, uint32(value.Uint()))
		return append(message, buffer[:4]...), nil

	case reflect.Uint64:
		binary.BigEndian.PutUint64(buffer, value.Uint())
		return append(message, buffer[:8]...), nil

	case reflect.Int8:
		return append(message, uint8(value.Int())), nil

	case reflect.Int16:
		binary.BigEndian.PutUint16(buffer, uint16(value.Int()))
		return append(message, buffer[:2]...), nil

	case reflect.Int32:
		binary.BigEndian.PutUint32(buffer, uint32(value.Int()))
		return append(message, buffer[:4]...), nil

	case reflect.Int64:
		binary.BigEndian.PutUint64(buffer, uint64(value.Int()))
		return append(message, buffer[:8]...), nil

	case reflect.Float32:
		binary.BigEndian.PutUint32(buffer, math.Float32bits(float32(value.Float())))
		return append(message, buffer[:4]...), nil

	case reflect.Float64:
		binary.BigEndian.PutUint64(buffer, math.Float64bits(value.Float()))
		return append(message, buffer[:8]...), nil

	case reflect.String:
		return append(message, []byte(value.String())...), nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", value.Kind())
	}
}
