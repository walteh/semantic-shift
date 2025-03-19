package repostprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessor_Process(t *testing.T) {
	// Set up paths
	schemaPath := filepath.Join("testdata", "color", "color.schema.json")
	modelPath := filepath.Join("testdata", "color", "model.gen.go")
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

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

func TestProcessor_ProcessMultipleSchemas(t *testing.T) {
	// Set up temporary directory
	tmpDir := t.TempDir()

	// Define schemas to process
	schemas := []struct {
		name       string
		schemaPath string
		modelPath  string
	}{
		{
			name:       "color",
			schemaPath: filepath.Join("testdata", "color", "color.schema.json"),
			modelPath:  filepath.Join("testdata", "color", "model.gen.go"),
		},
		{
			name:       "confusing",
			schemaPath: filepath.Join("testdata", "confusing", "confusing.schema.json"),
			modelPath:  filepath.Join("testdata", "confusing", "model.gen.go"),
		},
	}

	// Process each schema
	for _, schema := range schemas {
		outputDir := filepath.Join(tmpDir, schema.name)
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create output directory for %s: %v", schema.name, err)
		}

		// Process the schema
		processor := NewProcessor(schema.schemaPath, schema.modelPath, outputDir)
		err = processor.Process()
		if err != nil {
			t.Fatalf("Failed to process schema %s: %v", schema.name, err)
		}

		// Verify output files were created
		expectedFiles := []string{
			filepath.Join(outputDir, "model_enhanced.gen.go"),
			filepath.Join(outputDir, "model_interfaces.gen.go"),
			filepath.Join(outputDir, "model_unmarshal.gen.go"),
		}

		for _, file := range expectedFiles {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Errorf("Expected output file not created for %s: %s", schema.name, file)
			}
		}
	}

	// Verify that color schema processing generated appropriate interface
	colorInterfacesPath := filepath.Join(tmpDir, "color", "model_interfaces.gen.go")
	colorInterfacesContent, err := os.ReadFile(colorInterfacesPath)
	if err != nil {
		t.Fatalf("Failed to read interfaces file for color: %v", err)
	}
	colorInterfacesStr := string(colorInterfacesContent)

	if !strings.Contains(colorInterfacesStr, "type Color interface {") {
		t.Errorf("Color interface not found in color output")
	}
	if !strings.Contains(colorInterfacesStr, "Model() ColorModel") {
		t.Errorf("Model() method not found in Color interface")
	}

	// Verify that confusing schema processing generated appropriate interface
	confusingInterfacesPath := filepath.Join(tmpDir, "confusing", "model_interfaces.gen.go")
	confusingInterfacesContent, err := os.ReadFile(confusingInterfacesPath)
	if err != nil {
		t.Fatalf("Failed to read interfaces file for confusing: %v", err)
	}
	confusingInterfacesStr := string(confusingInterfacesContent)

	if !strings.Contains(confusingInterfacesStr, "type Horse interface {") {
		t.Errorf("Horse interface not found in confusing output")
	}
	if !strings.Contains(confusingInterfacesStr, "Rice() HorseModel") {
		t.Errorf("Rice() method not found in Horse interface")
	}
}
