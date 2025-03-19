package postprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func TestCodeGenerator(t *testing.T) {
	// Get the test schema path
	schemaPath := filepath.Join("testdata", "color.schema.json")
	modelPath := filepath.Join("testdata", "model.gen.go")

	// Setup a temporary output directory
	testOutputDir := filepath.Join(t.TempDir(), "output")
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create test output directory: %v", err)
	}

	// Create analyzer and run analysis
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	results, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Create generator
	outputPath := filepath.Join(testOutputDir, "schema_fixes.gen.go")
	generator := NewCodeGenerator(modelPath, outputPath, results)

	// Run generation
	if err := generator.Generate(); err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Verify that the output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Read the output file contents
	outputContents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Basic verification
	if len(outputContents) == 0 {
		t.Error("Generated file is empty")
	}

	// Check for some expected content
	outputStr := string(outputContents)

	// Check for Parent interfaces
	for parentName := range results.Parents {
		interfaceDecl := fmt.Sprintf("type %s interface {", parentName)
		if !strings.Contains(outputStr, interfaceDecl) {
			t.Errorf("Generated code doesn't contain %s interface", parentName)
		}

		// Check for implementation methods for each child
		for _, childName := range results.Parents[parentName].Children {
			implMethod := fmt.Sprintf("func (me *%s) is%s()", childName, parentName)
			if !strings.Contains(outputStr, implMethod) {
				t.Errorf("Generated code doesn't contain implementation method for %s", childName)
			}
		}
	}

	// Check for parse functions
	for parentName := range results.Parents {
		parseFunc := fmt.Sprintf("func parseUnknown%s", parentName)
		if !strings.Contains(outputStr, parseFunc) {
			t.Errorf("Generated code doesn't contain parse function for %s", parentName)
		}
	}

	// Check for marshal functions for types with constants
	for parentName, parentInfo := range results.Parents {
		if parentInfo.ConstField != "" {
			// Check for model type
			modelType := fmt.Sprintf("type %sModel string", parentName)
			if !strings.Contains(outputStr, modelType) {
				t.Errorf("Generated code doesn't contain model type for %s", parentName)
			}

			// Check for constants with sanitized identifiers
			for _, constValue := range parentInfo.ConstValues {
				sanitizedValue := strings.Map(func(r rune) rune {
					if r == '-' || r == ' ' || r == '.' || !unicode.IsLetter(r) && !unicode.IsNumber(r) {
						return '_'
					}
					return r
				}, constValue)

				if len(sanitizedValue) > 0 && unicode.IsNumber(rune(sanitizedValue[0])) {
					sanitizedValue = "_" + sanitizedValue
				}

				constDecl := fmt.Sprintf("%sModel%s", parentName, strings.Title(sanitizedValue))
				if !strings.Contains(outputStr, constDecl) {
					t.Errorf("Generated code doesn't contain sanitized constant %s", constDecl)
				}
			}
		}
	}
}
