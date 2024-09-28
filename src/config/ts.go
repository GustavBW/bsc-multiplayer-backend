package config

import (
	"fmt"
	"reflect"
)

func FormatTSEnum[T any](name string, data []T, formatter func(T) (string, string)) string {
	result := fmt.Sprintf("export enum %s {\n", name)
	for _, item := range data {
		name, value := formatter(item)
		result += fmt.Sprintf("\t%s = %s,\n", name, value)
	}
	result += "};\n\n"
	return result
}

// PANICS if the kind is not supported
func TSTypeOf(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16, reflect.Int32,
		reflect.Uint32, reflect.Int64, reflect.Uint64, reflect.Int, reflect.Uint,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return "number"
	case reflect.String:
		return "string"
	default:
		panic(fmt.Errorf("kind %s is not supported", kind))
	}
}
