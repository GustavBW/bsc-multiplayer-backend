package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func HandleToolRequest(args []string) error {
	// Print the event specs
	if len(args) == 0 {
		return fmt.Errorf("no tool specified")
	}

	for _, arg := range args {
		if arg == "--print-event-specs" {
			log.Println("[config] --print-event-specs flag found, printing event specs")
			return handleEventSpecRequest(args[1:])
		}
	}

	return nil
}

func handleEventSpecRequest(args []string) error {
	var programVersion = GetOr("PROGRAM_VERSION", "UNKOWN")
	var outputPath = "./EventSpecifications-v" + programVersion + ".ts"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--output=") {
			// Extract the output path
			outputPath = strings.TrimPrefix(arg, "--output=")
			// Optionally, remove the quotes if they are present
			outputPath = strings.Trim(outputPath, `"`)
			break
		}
	}

	if mkDirErr := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); mkDirErr != nil {
		return fmt.Errorf("error creating directory for output file: %s", mkDirErr.Error())
	}

	// Create the file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	outputFormat, outputFormatErr := GetOutputFormatFromPath(outputPath)
	if outputFormatErr != nil {
		return outputFormatErr
	}
	// Write the event specs to the file
	if writeErr := WriteEventSpecsToFile(file, outputFormat); writeErr != nil {
		return writeErr
	}

	return nil
}
