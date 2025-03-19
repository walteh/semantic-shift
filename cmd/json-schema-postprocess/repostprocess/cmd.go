package repostprocess

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/tozd/go/errors"
)

// ProcessJSONSchema takes a JSON schema file and generates enhanced Go code
func ProcessJSONSchema(schemaPath string, modelPath string, outputPath string) error {
	// Validate paths
	absSchemaPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return errors.Errorf("resolving schema path: %w", err)
	}

	absModelPath, err := filepath.Abs(modelPath)
	if err != nil {
		return errors.Errorf("resolving model path: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return errors.Errorf("resolving output path: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(absOutputPath, 0755); err != nil {
		return errors.Errorf("creating output directory: %w", err)
	}

	// Create the processor
	processor := NewProcessor(absSchemaPath, absModelPath, absOutputPath)

	// Process everything
	if err := processor.Process(); err != nil {
		return errors.Errorf("processing schema: %w", err)
	}

	return nil
}

// ProcessJSONSchemaFromArgs processes command line arguments
func ProcessJSONSchemaFromArgs() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: %s <schema-path> <model-path> <output-path>", os.Args[0])
	}

	schemaPath := os.Args[1]
	modelPath := os.Args[2]
	outputPath := os.Args[3]

	// Process the schema
	return ProcessJSONSchema(schemaPath, modelPath, outputPath)
}
