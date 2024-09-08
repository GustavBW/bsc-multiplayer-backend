package config

import (
	"fmt"
	"strings"
)

// Following name="value"
func retrieveValueOfKVArg(arg string) (string, error) {
	if !strings.Contains(arg, "=") {
		return "", fmt.Errorf("invalid key-value argument: %s", arg)
	}

	split := strings.Split(arg, "=")[1]
	if len(split) == 0 {
		return "", fmt.Errorf("empty value in key-value argument: %s", arg)
	}

	return strings.Trim(strings.ReplaceAll(split, "\"", ""), " "), nil
}
