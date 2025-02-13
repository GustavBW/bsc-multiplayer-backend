package util

import (
	"fmt"
	"reflect"
	"strconv"
)

type VarType[T any] struct {
	Value T
	Type  reflect.Kind
}

func NewVarType[T any](value T) *VarType[T] {
	return &VarType[T]{Value: value, Type: reflect.TypeOf(value).Kind()}
}

func (v VarType[T]) IsType(t reflect.Kind) bool {
	return v.Type == t
}

type ErrorTypeMismatch struct {
	Expected reflect.Kind
	Actual   reflect.Kind
}

func (e *ErrorTypeMismatch) Error() string {
	return "Type mismatch: expected " + e.Expected.String() + ", got " + e.Actual.String()
}

func (v VarType[T]) Int() (int, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return int(val.Float()), nil
	case reflect.String:
		return strconv.Atoi(val.String())
	}
	return 0, &ErrorTypeMismatch{Expected: reflect.Int, Actual: v.Type}
}
func (v VarType[T]) IntOr(fallback int) int {
	val, err := v.Int()
	if err != nil {
		return fallback
	}
	return val
}

func (v VarType[T]) Uint() (uint, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(val.Uint()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint(val.Int()), nil
	case reflect.Float32, reflect.Float64:
		return uint(val.Float()), nil
	case reflect.String:
		i, err := strconv.Atoi(val.String())
		return uint(i), err
	}
	return 0, &ErrorTypeMismatch{Expected: reflect.Uint, Actual: v.Type}
}

func (v VarType[T]) UintOr(fallback uint) uint {
	val, err := v.Uint()
	if err != nil {
		return fallback
	}
	return val
}

func (v VarType[T]) Float32() (float32, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.Float32, reflect.Float64:
		return float32(val.Float()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float32(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float32(val.Uint()), nil
	case reflect.String:
		f, err := strconv.ParseFloat(val.String(), 32)
		return float32(f), err
	}
	return 0, &ErrorTypeMismatch{Expected: reflect.Float32, Actual: v.Type}
}

func (v VarType[T]) Float32Or(fallback float32) float32 {
	val, err := v.Float32()
	if err != nil {
		return fallback
	}
	return val
}

func (v VarType[T]) Float64() (float64, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.Float32, reflect.Float64:
		return val.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), nil
	case reflect.String:
		return strconv.ParseFloat(val.String(), 64)
	}
	return 0, &ErrorTypeMismatch{Expected: reflect.Float64, Actual: v.Type}
}

func (v VarType[T]) Float64Or(fallback float64) float64 {
	val, err := v.Float64()
	if err != nil {
		return fallback
	}
	return val
}

func (v VarType[T]) String() (string, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.String:
		return val.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return fmt.Sprintf("%v", v.Value), nil
	}
	return "", &ErrorTypeMismatch{Expected: reflect.String, Actual: v.Type}
}

func (v VarType[T]) StringOr(fallback string) string {
	val, err := v.String()
	if err != nil {
		return fallback
	}
	return val
}

func (v VarType[T]) Bool() (bool, error) {
	val := reflect.ValueOf(v.Value)
	switch v.Type {
	case reflect.Bool:
		return val.Bool(), nil
	case reflect.String:
		return strconv.ParseBool(val.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Int() != 0, nil
	}
	return false, &ErrorTypeMismatch{Expected: reflect.Bool, Actual: v.Type}
}

func (v VarType[T]) BoolOr(fallback bool) bool {
	val, err := v.Bool()
	if err != nil {
		return fallback
	}
	return val
}

type MultiTypeMap[T comparable] struct {
	base map[T]*VarType[any]
}

func NewMultiTypeMap[T comparable]() *MultiTypeMap[T] {
	return &MultiTypeMap[T]{
		base: make(map[T]*VarType[any]),
	}
}

func (m *MultiTypeMap[T]) Set(key T, value any) {
	m.base[key] = NewVarType(value)
}

func (m *MultiTypeMap[T]) Get(key T) (*VarType[any], bool) {
	value, exists := m.base[key]
	return value, exists
}
