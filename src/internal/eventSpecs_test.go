package internal

import (
	"reflect"
	"testing"
)

type testType1 = struct {
	Field1 string `json:"field1" comment:"This is a comment"`
	Field2 int    `json:"field2" comment:"This is another comment"`
}

func TestDeriveReferenceStructure(t *testing.T) {
	ref, err := DeriveReferenceDescriptionFromT[testType1]()
	if err != nil {
		t.Error(err)
	}
	if len(ref) != 2 {
		t.Error("Expected 2 elements in reference structure")
	}
	if ref[0].FieldName != "field1" {
		t.Error("Expected field1 as fieldName in first element")
	}
	if ref[1].FieldName != "field2" {
		t.Error("Expected field2 as fieldName in second element")
	}
	if ref[0].Description != "This is a comment" {
		t.Error("Expected 'This is a comment' as description for Field1")
	}
	if ref[1].Description != "This is another comment" {
		t.Error("Expected 'This is another comment' as description for Field2")
	}
	if ref[0].Kind != reflect.String {
		t.Error("Expected KindString for Field1")
	}
	if ref[1].Kind != reflect.Int {
		t.Error("Expected KindInt for Field2")
	}
}
