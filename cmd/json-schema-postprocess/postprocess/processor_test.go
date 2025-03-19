package postprocess

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessor(t *testing.T) {
	// Get the test schema path
	schemaPath := filepath.Join("testdata", "color.schema.json")
	modelPath := filepath.Join("testdata", "model.gen.go")

	// Setup a temporary output directory
	testOutputDir := filepath.Join(t.TempDir(), "output")
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create test output directory: %v", err)
	}

	// Create processor
	processor := NewProcessor(schemaPath, modelPath, testOutputDir)

	// Run processing
	if err := processor.Process(); err != nil {
		t.Fatalf("Failed to process schema: %v", err)
	}

	// Verify that the output files exist
	fixesPath := filepath.Join(testOutputDir, "schema_fixes.gen.go")
	if _, err := os.Stat(fixesPath); os.IsNotExist(err) {
		t.Error("Schema fixes file was not created")
	}

	unmarshalPath := filepath.Join(testOutputDir, "schema_unmarshal.gen.go")
	if _, err := os.Stat(unmarshalPath); os.IsNotExist(err) {
		t.Error("Schema unmarshal overrides file was not created")
	}
}
