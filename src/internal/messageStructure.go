package internal

import (
	"fmt"
	"reflect"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// For use in statically computing elements at the very start of the application
type MessageElementDescriptor struct {
	ByteSize uint32 //Byte size of 0 means variable size
	// accounting for message header
	Offset      uint32
	FieldName   string
	Description string
	Kind        reflect.Kind //We do not intend to encode structs, so this is fine
}
type ShortElementDescriptor struct {
	Description string
	FieldName   string
	Kind        reflect.Kind //We do not intend to encode structs, so this is fine
}

// description is a human readable description of the element, appears as a comment in generated code
//
// fieldName is the name of the corresponding field in some in some type repressenting this message
func NewElementDescriptor(description string, fieldName string, kind reflect.Kind) ShortElementDescriptor {
	return ShortElementDescriptor{
		Description: description,
		FieldName:   fieldName,
		Kind:        kind,
	}
}

// In order slice of elements
type ReferenceStructure []ShortElementDescriptor

var REFERENCE_STRUCTURE_EMPTY = ReferenceStructure{}

// In order slice of computed elements
type ComputedStructure []MessageElementDescriptor

// in bytes
const MESSAGE_HEADER_SIZE uint32 = 8

// PANICS if the kind is not supported, or the ReferenceStructure does not adhere to simplified message format
//
// Returns the minimum total size of any message of this description as well as the full computed structure
// the min size does not include the message header
func ComputeStructure(messageName string, shortDescription ReferenceStructure) (uint32, ComputedStructure) {
	var computedStructure ComputedStructure
	var offset uint32 = MESSAGE_HEADER_SIZE
	var hasVariableSizeElement bool = false
	var minimumTotalSize uint32 = 0

	for index, element := range shortDescription {
		if err := isValidKind(element.Kind); err != nil {
			panic(err)
		}

		var isVariable bool = IsKindOfVariableSize(element.Kind)
		if isVariable && hasVariableSizeElement {
			panic(fmt.Errorf("message %s has multiple variable size elements", messageName))
		}

		// Any variable elements should be on the end of the message
		if isVariable && index != len(shortDescription)-1 {
			panic(fmt.Errorf("message %s has a variable size element that is not the last element", messageName))
		}

		// Extract the actual value from the interface and use unsafe.Sizeof
		sizeOfElement := util.SizeOfSerializedKind(element.Kind)

		computedStructure = append(computedStructure, MessageElementDescriptor{
			ByteSize:    sizeOfElement,
			Offset:      offset,
			FieldName:   element.FieldName,
			Description: element.Description,
			Kind:        element.Kind,
		})
		offset += sizeOfElement
		minimumTotalSize += sizeOfElement
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

var TypesAllowed = []reflect.Kind{
	reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String,
}

func IsKindOfVariableSize(kind reflect.Kind) bool {
	switch kind {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Interface, reflect.Struct:
		return true
	}

	return false
}
