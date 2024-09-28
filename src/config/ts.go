package config

import "fmt"

func FormatTSEnum[T any](name string, data []T, formatter func(T) (string, string)) string {
	result := fmt.Sprintf("export enum %s {\n", name)
	for _, item := range data {
		name, value := formatter(item)
		result += fmt.Sprintf("\t%s = %s,\n", name, value)
	}
	result += "};\n\n"
	return result
}
