package repostprocess

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessor_Process(t *testing.T) {
	// Set up paths
	schemaPath := filepath.Join("testdata", "color.schema.json")
	modelPath := filepath.Join("testdata", "model.gen.go")
	outputDir := filepath.Join("testdata", "output")

	// Ensure output directory exists and is clean
	_ = os.RemoveAll(outputDir)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create processor
	processor := NewProcessor(schemaPath, modelPath, outputDir)

	// Process everything
	err = processor.Process()
	if err != nil {
		t.Fatalf("Failed to process: %v", err)
	}

	// Verify output files were created
	expectedFiles := []string{
		filepath.Join(outputDir, "model_enhanced.gen.go"),
		filepath.Join(outputDir, "model_interfaces.gen.go"),
		filepath.Join(outputDir, "model_unmarshal.gen.go"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected output file not created: %s", file)
		}
	}
}
