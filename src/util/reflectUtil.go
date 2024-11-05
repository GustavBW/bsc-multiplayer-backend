package util

import (
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
	return reflect.Value{}, false
}
