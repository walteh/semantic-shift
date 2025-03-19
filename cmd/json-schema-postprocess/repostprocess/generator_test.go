package repostprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeGenerator_Generate(t *testing.T) {
	// Set up paths
	schemaPath := filepath.Join("testdata", "color", "color.schema.json")
	modelPath := filepath.Join("testdata", "color", "model.gen.go")

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	// Ensure output directory exists
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create analyzer and get results
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	results, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Create and run generator
	generator := NewCodeGenerator(modelPath, outputDir, results)
	err = generator.Generate()
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
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

	// Test specific functionality based on steps in postprocess.mdc

	// Step 1: Check if interfaces are properly generated
	interfacesContent, err := os.ReadFile(filepath.Join(outputDir, "model_interfaces.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read interfaces file: %v", err)
	}

	interfacesStr := string(interfacesContent)

	// Check for interface declaration and methods
	checkForContent(t, interfacesStr, "type Color interface {")
	checkForContent(t, interfacesStr, "isColor()")
	checkForContent(t, interfacesStr, "Model() ColorModel")

	// Check for implementation methods
	checkForContent(t, interfacesStr, "func (me *HSLValue) isColor()")
	checkForContent(t, interfacesStr, "func (me *RGBValue) isColor()")

	// Check for constant type definitions
	checkForContent(t, interfacesStr, "type ColorModel string")
	checkForContent(t, interfacesStr, "const (")
	checkForContent(t, interfacesStr, "ColorModelHsl")

	// Step 2: Check if parseUnknown functions are generated
	unmarshalContent, err := os.ReadFile(filepath.Join(outputDir, "model_unmarshal.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read unmarshal file: %v", err)
	}

	unmarshalStr := string(unmarshalContent)

	// Check for parseUnknown functions
	checkForContent(t, unmarshalStr, "func parseUnknownColor(b interface{}) (Color, error)")

	// Step 3: Check for MarshalJSON methods
	checkForContent(t, unmarshalStr, "func (j LCHValue) MarshalJSON() ([]byte, error)")

	// Step 4: Check type adjustments in enhanced model
	enhancedContent, err := os.ReadFile(filepath.Join(outputDir, "model_enhanced.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read enhanced model file: %v", err)
	}

	enhancedStr := string(enhancedContent)

	// Check for proper field type modifications
	checkForContent(t, enhancedStr, "Colors ColorSlice")
	checkForContent(t, enhancedStr, "Value Color")
}

func checkForContent(t *testing.T, content, expected string) {
	if !contains(content, expected) {
		t.Errorf("Expected content not found: %s", expected)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
