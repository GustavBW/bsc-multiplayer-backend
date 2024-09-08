package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unicode"

	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
)

type OutputFormat string

const (
	TS   OutputFormat = "ts"
	JSON OutputFormat = "json"
)

func WriteEventSpecsToFile(file *os.File, outputFormat OutputFormat) error {
	switch outputFormat {
	case TS:
		return writeEventSpecsToTSFile(file)
	case JSON:
		return writeEventSpecsToJSONFile(file)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

}

type NameAndID struct {
	Name string
	ID   uint32
}

func writeEventSpecsToTSFile(file *os.File) error {
	_, writeErr := file.WriteString("// !!! This content is generated by the multiplayer backend tool. Do not modify manually !!!\n")
	if writeErr != nil {
		return writeErr
	}
	file.WriteString(fmt.Sprintf("// !!! Last Updated (DD/MM/YYYY HH:MM:SS CET): %s !!!\n\n", time.Now().Format("02/01/2006 15:04:05 MST")))
	//TS Types - OriginType enum
	file.WriteString("export enum OriginType {\n")
	file.WriteString("\tServer = \"server\",\n")
	file.WriteString("\tOwner = \"owner\",\n")
	file.WriteString("\tGuest = \"guest\"\n")
	file.WriteString("};\n\n")

	//TS Types - SendPermissions
	file.WriteString("export type SendPermissions = { [key in OriginType]: boolean };\n\n")

	file.WriteString("export type MessageElementDescriptor = {\n")
	file.WriteString("\tbyteSize: number,\n")
	file.WriteString("\toffset: number,\n")
	file.WriteString("\tdescription: string,\n")
	file.WriteString("\ttype: string\n")
	file.WriteString("};\n\n")

	//TS Types - EventSpecification
	file.WriteString("export type EventSpecification = {\n")
	file.WriteString("\tid: number,\n")
	file.WriteString("\tname: string,\n")
	file.WriteString("\tpermissions: SendPermissions,\n")
	file.WriteString("\texpectedMinSize: number\n")
	file.WriteString("\tstructure: MessageElementDescriptor[]\n")
	file.WriteString("};\n\n")

	specs := getOrderedEventSpecs()
	var tsVarNamesAndIDs = make([]NameAndID, 0, len(specs))
	//Content
	for _, spec := range specs {
		varName := formatTSConstantName(&spec)
		tsVarNamesAndIDs = append(tsVarNamesAndIDs, NameAndID{Name: varName, ID: spec.ID})
		insertJSDOCCommentDescribingStructure(file, spec)
		file.WriteString(fmt.Sprintf("export const %s: EventSpecification = {\n", varName))
		file.WriteString(fmt.Sprintf("\tid: %d,\n", spec.ID))
		file.WriteString(fmt.Sprintf("\tname: \"%s\",\n", spec.Name))
		file.WriteString(fmt.Sprintf("\tpermissions: %s,\n", formatTSSendPermissions(spec.SendPermissions)))
		file.WriteString(fmt.Sprintf("\texpectedMinSize: %d,\n", spec.ExpectedMinSize))
		file.WriteString("\tstructure: [\n")
		for i, element := range spec.Structure {
			file.WriteString("\t\t{\n")
			file.WriteString(fmt.Sprintf("\t\t\tbyteSize: %d,\n", element.ByteSize))
			file.WriteString(fmt.Sprintf("\t\t\toffset: %d,\n", element.Offset))
			file.WriteString(fmt.Sprintf("\t\t\tdescription: \"%s\",\n", element.Description))
			file.WriteString(fmt.Sprintf("\t\t\ttype: \"%s\"\n", element.Kind))
			if i == len(spec.Structure)-1 {
				file.WriteString("\t\t}\n")
			} else {
				file.WriteString("\t\t},\n")
			}
		}
		file.WriteString("\t]\n")
		file.WriteString("}\n")
	}
	file.WriteString("\n")
	file.WriteString("export const ALL_EVENTS: {[key: number]: EventSpecification} = {\n")
	for i, nameAndID := range tsVarNamesAndIDs {
		file.WriteString(fmt.Sprintf("\t%d: %s", nameAndID.ID, nameAndID.Name))
		if i == len(tsVarNamesAndIDs)-1 {
			file.WriteString("\n")
		} else {
			file.WriteString(",\n")
		}
	}
	file.WriteString("};\n")
	return nil
}

func insertJSDOCCommentDescribingStructure(file *os.File, spec internal.EventSpecification) {
	file.WriteString(fmt.Sprintf("/** %s Message Structure\n *\n", spec.Name))

	for _, element := range spec.Structure {
		isVariable := element.ByteSize == 0
		if isVariable {
			file.WriteString(fmt.Sprintf(" * *\t%db --> +%sb:\t%-10s:\t%s\n", element.Offset, "N", element.Kind, element.Description))
		} else {
			file.WriteString(fmt.Sprintf(" * *\t%db --> %db:\t%-10s:\t%s\n", element.Offset, element.Offset+element.ByteSize, element.Kind, element.Description))
		}
	}
	file.WriteString(" */\n")
}

func writeEventSpecsToJSONFile(file *os.File) error {
	file.WriteString("[\n")
	specs := getOrderedEventSpecs()
	for index, spec := range specs {
		file.WriteString("{\n")
		file.WriteString(fmt.Sprintf("\t\"id\": %d,\n", spec.ID))
		file.WriteString(fmt.Sprintf("\t\"name\": \"%s\",\n", spec.Name))
		file.WriteString(fmt.Sprintf("\t\"permissions\": %s,\n", formatJSONSendPermissions(spec.SendPermissions)))
		file.WriteString(fmt.Sprintf("\t\"expectedMinSize\": %d\n", spec.ExpectedMinSize))
		if index == len(specs)-1 {
			file.WriteString("}\n")
		} else {
			file.WriteString("},\n")
		}
	}
	file.WriteString("]\n")

	return fmt.Errorf("not implemented")
}

func getOrderedEventSpecs() []internal.EventSpecification {
	// Create a slice of the values from the map
	specs := make([]internal.EventSpecification, 0, len(internal.ALL_EVENTS))
	for _, spec := range internal.ALL_EVENTS {
		specs = append(specs, *spec)
	}

	// Sort the slice by the ID field, lowest to highest
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].ID < specs[j].ID
	})

	return specs
}

func formatTSSendPermissions(permissions map[internal.OriginType]bool) string {
	var result = "{"
	count := 0
	total := len(permissions)
	for key, value := range permissions {
		result += fmt.Sprintf("%s: %t", key, value)
		count++
		if count < total {
			result += ", "
		}
	}
	result += "}"
	return result
}

func formatJSONSendPermissions(permissions map[internal.OriginType]bool) string {
	var result = "{"
	count := 0
	total := len(permissions)
	for key, value := range permissions {
		result += fmt.Sprintf("\"%s\": %t", key, value)
		count++
		if count < total {
			result += ", "
		}
	}
	result += "}"
	return result
}

func GetOutputFormatFromPath(path string) (OutputFormat, error) {
	switch filepath.Ext(path) {
	case ".ts":
		return TS, nil
	case ".json":
		return JSON, nil
	}

	return "", fmt.Errorf("unsupported file extension: %s", filepath.Ext(path))
}

func formatTSConstantName(spec *internal.EventSpecification) string {
	var result []rune

	for i, r := range spec.Name {
		// If it's an uppercase letter and it's not the first character, insert an underscore
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		// Append the uppercase version of the character
		result = append(result, unicode.ToUpper(r))
	}

	// Join the result and append "_EVENT"
	return fmt.Sprintf("%s_EVENT", string(result))
}
