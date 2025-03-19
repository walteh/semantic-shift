package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/semantic-shift/cmd/json-schema-postprocess/postprocess"
)

func main() {
	// Define command-line flags
	schemaFile := flag.String("schema", "", "Path to the JSON schema file")
	modelFile := flag.String("model", "", "Path to the Go model file")
	outputDir := flag.String("output", "", "Output directory for generated files")

	flag.Parse()

	// Validate required arguments
	if *schemaFile == "" {
		fmt.Println("Error: schema file path is required")
		flag.Usage()
		os.Exit(1)
	}
	if *modelFile == "" {
		fmt.Println("Error: model file path is required")
		flag.Usage()
		os.Exit(1)
	}
	if *outputDir == "" {
		fmt.Println("Error: output directory is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create processor and run it
	processor := postprocess.NewProcessor(*schemaFile, *modelFile, *outputDir)
	if err := processor.Process(); err != nil {
		fmt.Printf("Error processing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Schema post-processing completed successfully!")
	fmt.Printf("Generated files written to %s\n", *outputDir)
}

// runWithTestData runs the processor with test data for development/testing
func runWithTestData() error {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return errors.Errorf("getting working directory: %w", err)
	}

	// Set up paths for test data
	testdataDir := filepath.Join(wd, "postprocess", "testdata")
	schemaFile := filepath.Join(testdataDir, "color.schema.json")
	modelFile := filepath.Join(testdataDir, "model.gen.go")
	outputDir := filepath.Join(testdataDir, "output")

	// Create processor and run it
	processor := postprocess.NewProcessor(schemaFile, modelFile, outputDir)
	if err := processor.Process(); err != nil {
		return errors.Errorf("processing schema: %w", err)
	}

	fmt.Println("Test processing completed successfully!")
	fmt.Printf("Generated files written to %s\n", outputDir)
	return nil
}
