package internal

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

func Deserialize[T any](spec *EventSpecification[T], data []byte) (*T, error) {
	var dest T

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

		value, err := parseGoTypeFromBytes(data, element.Offset, element.Kind)
		if err != nil {
			return nil, err
		}

		if err := setStructField(&dest, field.Name, value); err != nil {
			return nil, err
		}
	}

	return &dest, nil
}

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

func parseGoTypeFromBytes(data []byte, offset uint32, kind reflect.Kind) (interface{}, error) {
	if sizeOfSerializedKind(kind) > uint32(len(data))-offset {
		return nil, fmt.Errorf("not enough data to parse %s", kind)
	}
	switch kind {
	case reflect.Uint8:
		return uint8(data[offset]), nil
	case reflect.Uint16:
		return binary.LittleEndian.Uint16(data[offset:]), nil
	case reflect.Uint32:
		return binary.LittleEndian.Uint32(data[offset:]), nil
	case reflect.Uint64:
		return binary.LittleEndian.Uint64(data[offset:]), nil
	case reflect.Int8:
		return int8(data[offset]), nil
	case reflect.Int16:
		return int16(binary.LittleEndian.Uint16(data[offset:])), nil
	case reflect.Int32:
		return int32(binary.LittleEndian.Uint32(data[offset:])), nil
	case reflect.Int64:
		return int64(binary.LittleEndian.Uint64(data[offset:])), nil
	case reflect.Float32:
		return math.Float32frombits(binary.LittleEndian.Uint32(data[offset:])), nil
	case reflect.Float64:
		return math.Float64frombits(binary.LittleEndian.Uint64(data[offset:])), nil
	default:
		panic(fmt.Sprintf("Unsupported type: %v", kind))
	}
}
