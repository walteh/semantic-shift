package postprocess

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProcessJSONSchema takes a JSON schema file and generates enhanced Go code with unmarshal overrides
func ProcessJSONSchema(schemaPath string, modelPath string, outputPath string) error {
	// Create the analyzer to validate the schema
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		return fmt.Errorf("creating schema analyzer: %w", err)
	}

	// Validate the schema
	if _, err := analyzer.Analyze(); err != nil {
		return fmt.Errorf("analyzing schema: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Create the processor
	processor := NewProcessor(schemaPath, modelPath, outputPath)

	// Process everything
	if err := processor.Process(); err != nil {
		return fmt.Errorf("processing schema: %w", err)
	}

	return nil
}

// ProcessJSONSchemaFromArgs processes command line arguments and calls ProcessJSONSchema
func ProcessJSONSchemaFromArgs() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: %s <schema-path> <model-path> <output-path>", os.Args[0])
	}

	schemaPath := os.Args[1]
	modelPath := os.Args[2]
	outputPath := os.Args[3]

	// Convert to absolute paths
	var err error
	schemaPath, err = filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("resolving schema path: %w", err)
	}

	modelPath, err = filepath.Abs(modelPath)
	if err != nil {
		return fmt.Errorf("resolving model path: %w", err)
	}

	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	// Process the schema
	return ProcessJSONSchema(schemaPath, modelPath, outputPath)
}
