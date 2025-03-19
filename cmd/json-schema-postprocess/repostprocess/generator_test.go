package repostprocess

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
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

	// Interface declaration and methods for Color
	checkForContent(t, interfacesStr, "type Color interface {")
	checkForContent(t, interfacesStr, "isColor()")
	checkForContent(t, interfacesStr, "Model() ColorModel")

	// Implementation methods for all Color children
	checkForContent(t, interfacesStr, "func (me *HSLValue) isColor()")
	checkForContent(t, interfacesStr, "func (me *RGBValue) isColor()")

	// ColorModel implementations for each color type - check simplified versions
	checkForContent(t, interfacesStr, "func (me *HSLValue) Model() ColorModel")
	checkForContent(t, interfacesStr, "func (me *RGBValue) Model() ColorModel")

	// Check slice and map type declarations
	checkForContent(t, interfacesStr, "type ColorSlice []Color")
	checkForContent(t, interfacesStr, "type ColorMap map[string]Color")

	// Interface declaration and methods for Palette
	checkForContent(t, interfacesStr, "type Palette interface {")
	checkForContent(t, interfacesStr, "isPalette()")

	// Implementation methods for Palette children
	checkForContent(t, interfacesStr, "func (me *CategoricalPalette) isPalette()")
	checkForContent(t, interfacesStr, "func (me *DiscreteScalePalette) isPalette()")

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
	checkForContent(t, unmarshalStr, "func parseUnknownPalette(b interface{}) (Palette, error)")

	// Check the constant field usage in parseUnknownColor - simplified checks
	checkForContent(t, unmarshalStr, "Model ColorModel `json:\"model\"")
	checkForContent(t, unmarshalStr, "switch plain.Model {")

	// Step 3: Check for MarshalJSON methods
	checkForContent(t, unmarshalStr, "func (j LCHValue) MarshalJSON() ([]byte, error)")
	checkForContent(t, unmarshalStr, "Model: j.Model()")
	checkForContent(t, unmarshalStr, "type Plain LCHValue")

	// Check that all color types have MarshalJSON methods - simplified checks
	checkForContent(t, unmarshalStr, "func (j HSLValue) MarshalJSON() ([]byte, error)")
	checkForContent(t, unmarshalStr, "func (j RGBValue) MarshalJSON() ([]byte, error)")

	// Step 4: Check type adjustments in enhanced model
	enhancedContent, err := os.ReadFile(filepath.Join(outputDir, "model_enhanced.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read enhanced model file: %v", err)
	}

	enhancedStr := string(enhancedContent)

	// Check for proper field type modifications - array parent callers
	checkForContent(t, enhancedStr, "Colors ColorSlice")
	checkForContent(t, enhancedStr, "Palettes PaletteSlice")

	// Check for proper field type modifications - direct parent callers
	checkForContent(t, enhancedStr, "Value Color")

	// Step 5: Check UnmarshalJSON modifications in parent callers
	checkForContent(t, unmarshalStr, "func (j *ColorConfig) UnmarshalJSON(b []byte) error")
	checkForContent(t, unmarshalStr, "parsed, err := parseUnknownColor(raw[\"value\"])")
	checkForContent(t, unmarshalStr, "plain.Value = parsed")

	// Check array parent callers
	checkForContent(t, unmarshalStr, "func (j *AssetPack) UnmarshalJSON(b []byte) error")
	checkForContent(t, unmarshalStr, "palettes := make(PaletteSlice")
	checkForContent(t, unmarshalStr, "parsed, err := parseUnknownPalette(item)")

	// Additional checks per user request

	// Check enhanced file doesn't contain type definitions that should have been replaced
	checkForAbsence(t, enhancedStr, "Type string `json:\"type\" yaml:\"type\" mapstructure:\"type\"`")
	checkForAbsence(t, enhancedStr, "type Palette interface{}")

	// Check unmarshal file doesn't contain functions it shouldn't
	checkForAbsence(t, unmarshalStr, "(j *MatrixPalette) UnmarshalJSON(")

	// Check unmarshal file contains specific validation
	checkForContent(t, unmarshalStr, "if _, ok := raw[\"semantic\"]; raw != nil && !ok {")
}

func TestCodeGenerator_GenerateConfusing(t *testing.T) {
	// Set up paths
	schemaPath := filepath.Join("testdata", "confusing", "confusing.schema.json")
	modelPath := filepath.Join("testdata", "confusing", "model.gen.go")

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
	checkForContent(t, interfacesStr, "type Horse interface {")
	checkForContent(t, interfacesStr, "isHorse()")
	checkForContent(t, interfacesStr, "Rice() HorseModel")

	// Check for implementation methods - just check a couple
	checkForContent(t, interfacesStr, "func (me *HSLVarient) isHorse()")
	checkForContent(t, interfacesStr, "func (me *RGBVarient) isHorse()")

	// Check for constant type definitions
	checkForContent(t, interfacesStr, "type HorseModel string")
	checkForContent(t, interfacesStr, "const (")
	checkForContent(t, interfacesStr, "HorseModelHsl")

	// Step 2: Check if parseUnknown functions are generated
	unmarshalContent, err := os.ReadFile(filepath.Join(outputDir, "model_unmarshal.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read unmarshal file: %v", err)
	}

	unmarshalStr := string(unmarshalContent)

	// Check for parseUnknown functions
	checkForContent(t, unmarshalStr, "func parseUnknownHorse(b interface{}) (Horse, error)")

	// Check for the use of the "rice" field instead of "model"
	checkForContent(t, unmarshalStr, "Rice HorseModel `json:\"rice\"")

	// Step 3: Check for MarshalJSON methods - just verify the function exists
	checkForContent(t, unmarshalStr, "func (j LCHVarient) MarshalJSON()")

	// Step 4: Check type adjustments in enhanced model
	enhancedContent, err := os.ReadFile(filepath.Join(outputDir, "model_enhanced.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read enhanced model file: %v", err)
	}

	enhancedStr := string(enhancedContent)

	// Check for proper field type modifications - 'Ices' is the equivalent of 'Colors' in confusing schema
	checkForContent(t, enhancedStr, "Ices HorseSlice")
	checkForContent(t, enhancedStr, "Rope Horse")
}

func TestJSONRoundTrip(t *testing.T) {
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

	// The rest of the test is a simulation of how the generated code would be used
	// We can't actually import and use the generated code directly in tests,
	// so we'll verify the generation of the serialization code

	// Verify marshaling functions for constants
	unmarshalContent, err := os.ReadFile(filepath.Join(outputDir, "model_unmarshal.gen.go"))
	if err != nil {
		t.Fatalf("Failed to read unmarshal file: %v", err)
	}
	unmarshalStr := string(unmarshalContent)

	// Verify the structure of the MarshalJSON methods - just check a few
	allColorTypes := []string{"HSLValue", "RGBValue", "LCHValue"}
	for _, colorType := range allColorTypes {
		// Check that each color type has a MarshalJSON method
		checkForContent(t, unmarshalStr, fmt.Sprintf("func (j %s) MarshalJSON()", colorType))
	}

	// Check basic marshaling elements
	checkForContent(t, unmarshalStr, "Model: j.Model()")
	checkForContent(t, unmarshalStr, "Plain: Plain(j)")

	// Verify the parseUnknown functions that would handle unmarshaling
	checkForContent(t, unmarshalStr, "func parseUnknownColor(b interface{}) (Color, error)")
	checkForContent(t, unmarshalStr, "switch plain.Model {")

	// Verify that a few cases in the switch statement correctly handle specific color types
	// Just pick a few representative cases
	checkForContent(t, unmarshalStr, "case ColorModelHsl:")
	checkForContent(t, unmarshalStr, "case ColorModelRgb:")

	// Check for modified UnmarshalJSON methods in parent callers
	checkForContent(t, unmarshalStr, "func (j *ColorConfig) UnmarshalJSON(b []byte) error")
	checkForContent(t, unmarshalStr, "parsed, err := parseUnknownColor(raw[\"value\"])")
	checkForContent(t, unmarshalStr, "plain.Value = parsed")
}

func TestGeneratedCodeValidation(t *testing.T) {
	testCases := []struct {
		name       string
		schemaPath string
		modelPath  string
	}{
		{
			name:       "ColorSchema",
			schemaPath: filepath.Join("testdata", "color", "color.schema.json"),
			modelPath:  filepath.Join("testdata", "color", "model.gen.go"),
		},
		{
			name:       "ConfusingSchema",
			schemaPath: filepath.Join("testdata", "confusing", "confusing.schema.json"),
			modelPath:  filepath.Join("testdata", "confusing", "model.gen.go"),
		},
		{
			name:       "SimpleSchema",
			schemaPath: filepath.Join("testdata", "simple", "simple.schema.json"),
			modelPath:  filepath.Join("testdata", "simple", "model.gen.go"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputDir := filepath.Join(tmpDir, "output")

			// Ensure output directory exists
			err := os.MkdirAll(outputDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			// Generate code
			analyzer, err := NewSchemaAnalyzer(tc.schemaPath)
			if err != nil {
				t.Fatalf("Failed to create analyzer: %v", err)
			}

			results, err := analyzer.Analyze()
			if err != nil {
				t.Fatalf("Failed to analyze schema: %v", err)
			}

			generator := NewCodeGenerator(tc.modelPath, outputDir, results)
			err = generator.Generate()
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// List all generated files
			outputFiles := []string{
				filepath.Join(outputDir, "model_enhanced.gen.go"),
				filepath.Join(outputDir, "model_interfaces.gen.go"),
				filepath.Join(outputDir, "model_unmarshal.gen.go"),
			}

			for _, file := range outputFiles {
				// Check that file exists
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Fatalf("Expected output file not created: %s", file)
				}

				// Run the Go parser to check if the generated code is valid Go
				validateGoSyntax(t, file)

				// Format the file and check if it changes (indicates non-standard formatting)
				validateGoFormat(t, file)
			}

			// Check for package name consistency
			verifyPackageConsistency(t, outputFiles)
		})
	}
}

// verifyPackageConsistency checks that all files have the same package name
func verifyPackageConsistency(t *testing.T, files []string) {
	t.Helper()

	var packageName string
	packagesInFiles := make(map[string]string)

	for _, file := range files {
		// Read the file content
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", file, err)
		}

		// Parse the file to extract package name
		fset := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fset, file, content, parser.PackageClauseOnly)
		if err != nil {
			t.Errorf("Failed to parse file %s: %v", file, err)
			continue
		}

		currentPackage := parsedFile.Name.Name
		packagesInFiles[file] = currentPackage

		if packageName == "" {
			packageName = currentPackage
		} else if currentPackage != packageName {
			t.Errorf("Package name inconsistency: %s has package %q, but expected %q",
				filepath.Base(file), currentPackage, packageName)
		}
	}

	// If there was a failure, log the package names for debugging
	if t.Failed() {
		t.Logf("Package names by file:")
		for file, pkg := range packagesInFiles {
			t.Logf("  %s: %s", filepath.Base(file), pkg)
		}
	}
}

func checkForContent(t *testing.T, content, expected string) {
	if !contains(content, expected) {
		t.Errorf("Expected content not found: %s", expected)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// validateGoSyntax checks if the file contains valid Go syntax by parsing it
func validateGoSyntax(t *testing.T, filePath string) {
	t.Helper()

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	// Parse the file to check for syntax errors
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, filePath, content, parser.AllErrors)
	if err != nil {
		t.Errorf("Generated file %s contains invalid Go syntax: %v", filePath, err)
	}
}

// validateGoFormat checks if the file follows standard Go formatting
func validateGoFormat(t *testing.T, filePath string) {
	t.Helper()

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath, err)
	}

	// Format the file using go/format
	formatted, err := format.Source(content)
	if err != nil {
		t.Errorf("Failed to format file %s: %v", filePath, err)
		return
	}

	// Compare original with formatted
	if string(content) != string(formatted) {
		// Write the formatted content to a temp file for debugging
		formattedPath := filePath + ".formatted"
		err = os.WriteFile(formattedPath, formatted, 0644)
		if err != nil {
			t.Logf("Failed to write formatted file for comparison: %v", err)
		}

		t.Errorf("File %s is not properly formatted according to Go standards", filePath)
	}
}

func checkForAbsence(t *testing.T, content, expected string) {
	if contains(content, expected) {
		t.Errorf("Unexpected content found: %s", expected)
	}
}
