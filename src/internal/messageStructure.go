package internal

import (
	"fmt"
	"reflect"
)

// For use in statically computing elements at the very start of the application

type MessageElementDescriptor struct {
	ByteSize    uint32 //Byte size of 0 means variable size
	Offset      uint32
	Description string
	Kind        reflect.Kind //We do not intend to encode structs, so this is fine
}
type ShortElementDescriptor struct {
	Description string
	Kind        reflect.Kind //We do not intend to encode structs, so this is fine
}

// In order slice of elements
type ReferenceStructure []ShortElementDescriptor

var REFERENCE_STRUCTURE_EMPTY = ReferenceStructure{}

// In order slice of computed elements
type ComputedStructure []MessageElementDescriptor

// PANICS if the kind is not supported, or the ReferenceStructure does not adhere to simplified message format
//
// Returns the minimum total size of any message of this description as well as the full computed structure
func ComputeStructure(messageName string, shortDescription ReferenceStructure) (uint32, ComputedStructure) {
	var computedStructure ComputedStructure
	var offset uint32 = 0
	var hasVariableSizeElement bool = false
	var minimumTotalSize uint32 = 0

	for index, element := range shortDescription {
		if err := isValidKind(element.Kind); err != nil {
			panic(err)
		}

		var isVariable bool = isKindOfVariableSize(element.Kind)
		if isVariable && hasVariableSizeElement {
			panic(fmt.Errorf("message %s has multiple variable size elements", messageName))
		}

		// Any variable elements should be on the end of the message
		if isVariable && index != len(shortDescription)-1 {
			panic(fmt.Errorf("message %s has a variable size element that is not the last element", messageName))
		}

		// Extract the actual value from the interface and use unsafe.Sizeof
		sizeOfElement := sizeOfSerializedKind(element.Kind)

		computedStructure = append(computedStructure, MessageElementDescriptor{
			ByteSize:    sizeOfElement,
			Offset:      offset,
			Description: element.Description,
			Kind:        element.Kind,
		})
		offset += sizeOfElement
	}
	return minimumTotalSize, computedStructure
}

// In terms of expected message contents
func isValidKind(kind reflect.Kind) error {
	switch kind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return nil
	default:
		return fmt.Errorf("kind %s is not supported", kind)
	}
}

func isKindOfVariableSize(kind reflect.Kind) bool {
	switch kind {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Interface, reflect.Struct:
		return true
	}

	return false
}

// sizeOfSerializedKind returns the size in bytes of the type when serialized.
//
// Returns 0 for variable length kinds like Array, Slice, String
func sizeOfSerializedKind(kind reflect.Kind) uint32 {
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
