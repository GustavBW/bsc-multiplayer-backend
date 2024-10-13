package internal

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Deserialize any some data into a struct by the type given in the spec
//
// Based on remainderOnly, it will either expect the entire message (headers and all)
// or only the remainder (body) of the message.
func Deserialize[T any](spec *EventSpecification[T], data []byte, remainderOnly bool) (*T, error) {
	// A specs offset is including the header although it does not itself describe it,
	// so if remainderOnly == true, we need subtract the header size to get the correct offset
	offsetAdjustment := util.Ternary(remainderOnly, MESSAGE_HEADER_SIZE, 0)

	if len(data) < int(spec.ExpectedMinSize)-int(offsetAdjustment) {
		return nil, fmt.Errorf("expected at least %d bytes, got %d", spec.ExpectedMinSize, len(data))
	}

	var dest T // Allocation of new nil-value instantiated copy of T

	t := reflect.TypeOf(dest)

	// If it's a pointer, get the underlying element
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s", t.Kind())
	}

	if t.NumField() != len(spec.Structure) {
		return nil, fmt.Errorf("expected %d fields, got %d", len(spec.Structure), t.NumField())
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		element := spec.Structure[i]
		if field.Type.Kind() != element.Kind {
			return nil, fmt.Errorf("expected field %d to be of kind %s, got %s", i, element.Kind, field.Type.Kind())
		}

		value, err := parseGoTypeFromBytes(data, element.Offset-offsetAdjustment, element.Kind)
		if err != nil {
			return nil, err
		}

		if err := setStructField(&dest, field.Name, value); err != nil {
			return nil, err
		}
	}

	return &dest, nil
}

// Extremely unsafe. Use with caution
func setStructField(strukt interface{}, fieldName string, value interface{}) error {
	structValue := reflect.ValueOf(strukt)
	if structValue.Kind() != reflect.Ptr || structValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("strukt must be a pointer to a struct")
	}
	structValue = structValue.Elem()

	fieldValue := structValue.FieldByName(fieldName)
	if !fieldValue.IsValid() {
		return fmt.Errorf("no such field: %s in struct %v", fieldName, structValue.Type())
	}

	if !fieldValue.CanSet() {
		return fmt.Errorf("cannot set field %s in struct %v", fieldName, structValue.Type())
	}

	fieldType := fieldValue.Type()
	val := reflect.ValueOf(value)
	if fieldType != val.Type() {
		return fmt.Errorf("provided value type didn't match obj field type")
	}

	fieldValue.Set(val)
	return nil
}

// Extremely unsafe. Use with caution
func parseGoTypeFromBytes(data []byte, offset uint32, kind reflect.Kind) (interface{}, error) {
	if sizeOfSerializedKind(kind) > uint32(len(data))-offset {
		return nil, fmt.Errorf("not enough data to parse %s", kind)
	}
	switch kind {
	case reflect.Uint8:
		return uint8(data[offset]), nil
	case reflect.Uint16:
		return binary.BigEndian.Uint16(data[offset:]), nil
	case reflect.Uint32:
		return binary.BigEndian.Uint32(data[offset:]), nil
	case reflect.Uint64:
		return binary.BigEndian.Uint64(data[offset:]), nil
	case reflect.Int8:
		return int8(data[offset]), nil
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(data[offset:])), nil
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(data[offset:])), nil
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(data[offset:])), nil
	case reflect.Float32:
		return math.Float32frombits(binary.BigEndian.Uint32(data[offset:])), nil
	case reflect.Float64:
		return math.Float64frombits(binary.BigEndian.Uint64(data[offset:])), nil
	case reflect.String:
		// For strings, we'll take all remaining bytes from the offset
		remaining := data[offset:]
		if !utf8.Valid(remaining) {
			return nil, fmt.Errorf("invalid UTF-8 string")
		}
		return string(remaining), nil
	default:
		panic(fmt.Sprintf("Unsupported type: %v", kind))
	}
}
