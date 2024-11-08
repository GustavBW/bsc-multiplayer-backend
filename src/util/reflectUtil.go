package util

import (
	"fmt"
	"reflect"
	"strings"
)

// findFieldByJSONName searches for a struct field that has a JSON tag matching the given name
// Returns the field and true if found, or zero value and false if not found
func FindFieldByJSONTagValue(v reflect.Value, jsonName string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		// Split the tag on comma in case there are options like omitempty
		name := strings.Split(tag, ",")[0]
		if name == jsonName {
			return v.Field(i), true
		}
	}
	return v, false
}

// Include comments during runtime by adding them as tags in the struct
// Example:
//
//	type MyStruct struct {
//		Field1 string `json:"field1" comment:"This is a comment"`
//		Field2 int `json:"field2" comment:"This is another comment"`
//	}
func GetCommentValue(field reflect.StructField) (string, error) {
	tag := field.Tag.Get("comment")
	if tag == "" {
		return "", fmt.Errorf("field %s does not have a comment tag or comment tag is empty", field.Name)
	}
	return tag, nil
}

// GetFieldNameFromTag returns the general field name from the json tag of a struct field
func GetFieldNameFromTag(field reflect.StructField) (string, error) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return "", fmt.Errorf("field %s does not have a json tag or json tag is empty", field.Name)
	}
	// Split the tag on comma in case there are options like omitempty
	name := strings.Split(tag, ",")[0]
	return name, nil
}
