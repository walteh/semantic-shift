package repostprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_SimpleSchema(t *testing.T) {
	// Set up paths
	schemaPath := filepath.Join("testdata", "simple", "simple.schema.json")
	modelPath := filepath.Join("testdata", "simple", "model.gen.go")

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

	// Define expected file contents based on the postprocess.mdc steps
	expectedFiles := map[string]string{
		"model_interfaces.gen.go": `// Code generated by json-schema-postprocess. DO NOT EDIT.

package simple

// This file contains interface definitions and implementation methods for parent types

// Shape represents the parent type for Shape types
type Shape interface {
	isShape()
	Type() ShapeModel
}

// ShapeSlice is a slice of Shape interfaces
type ShapeSlice []Shape

// ShapeMap is a map of Shape interfaces
type ShapeMap map[string]Shape

// isShape implements the Shape interface
func (me *Circle) isShape() {}

// Type returns the Shape type constant
func (me *Circle) Type() ShapeModel { return ShapeModelCircle }

// isShape implements the Shape interface
func (me *Square) isShape() {}

// Type returns the Shape type constant
func (me *Square) Type() ShapeModel { return ShapeModelSquare }

// isShape implements the Shape interface
func (me *Triangle) isShape() {}

// Type returns the Shape type constant
func (me *Triangle) Type() ShapeModel { return ShapeModelTriangle }

// ShapeModel represents the type model type
type ShapeModel string

// Constants for the different model types
const (
	ShapeModelCircle   ShapeModel = "circle"
	ShapeModelSquare   ShapeModel = "square"
	ShapeModelTriangle ShapeModel = "triangle"
)`,

		"model_unmarshal.gen.go": `// Code generated by json-schema-postprocess. DO NOT EDIT.

package simple

import (
	"encoding/json"
	"fmt"
)

// This file contains unmarshaling and marshaling functions for parent types

// parseUnknownShape parses an unknown Shape type based on its JSON representation
func parseUnknownShape(b interface{}) (Shape, error) {
	str, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	// Use the type field to determine the type
	type Plain struct {
		Type ShapeModel ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
	}
	var plain Plain
	if err := json.Unmarshal(str, &plain); err != nil {
		return nil, err
	}

	switch plain.Type {
	case ShapeModelCircle:
		var circle Circle
		err = json.Unmarshal(str, &circle)
		return &circle, err
	case ShapeModelSquare:
		var square Square
		err = json.Unmarshal(str, &square)
		return &square, err
	case ShapeModelTriangle:
		var triangle Triangle
		err = json.Unmarshal(str, &triangle)
		return &triangle, err
	default:
		return nil, fmt.Errorf("invalid type: %s", plain.Type)
	}
}

// MarshalJSON implements json.Marshaler for Circle
func (j Circle) MarshalJSON() ([]byte, error) {
	// Add the constant field to the output
	type Plain Circle
	myMarshal := struct {
		Type ShapeModel ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
		Plain
	}{
		Type:  j.Type(),
		Plain: Plain(j),
	}
	return json.Marshal(myMarshal)
}

// MarshalJSON implements json.Marshaler for Square
func (j Square) MarshalJSON() ([]byte, error) {
	// Add the constant field to the output
	type Plain Square
	myMarshal := struct {
		Type ShapeModel ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
		Plain
	}{
		Type:  j.Type(),
		Plain: Plain(j),
	}
	return json.Marshal(myMarshal)
}

// MarshalJSON implements json.Marshaler for Triangle
func (j Triangle) MarshalJSON() ([]byte, error) {
	// Add the constant field to the output
	type Plain Triangle
	myMarshal := struct {
		Type ShapeModel ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
		Plain
	}{
		Type:  j.Type(),
		Plain: Plain(j),
	}
	return json.Marshal(myMarshal)
}

// UnmarshalJSON implements json.Unmarshaler for SimpleSchemaJsonConfig
func (j *SimpleSchemaJsonConfig) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["shape"]; raw != nil && !ok {
		return fmt.Errorf("field shape in SimpleSchemaJsonConfig: required")
	}

	type Plain SimpleSchemaJsonConfig
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	// Parse Shape type
	if raw["shape"] != nil {
		parsed, err := parseUnknownShape(raw["shape"])
		if err != nil {
			return err
		}
		plain.Shape = parsed
	}

	*j = SimpleSchemaJsonConfig(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler for SimpleSchemaJson
func (j *SimpleSchemaJson) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	type Plain SimpleSchemaJson
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	// Parse array of Shape types
	if raw["shapes"] != nil {
		arr, ok := raw["shapes"].([]interface{})
		if ok {
			shapes := make(ShapeSlice, 0, len(arr))
			for _, item := range arr {
				parsed, err := parseUnknownShape(item)
				if err != nil {
					return err
				}
				shapes = append(shapes, parsed)
			}
			plain.Shapes = shapes
		}
	}

	*j = SimpleSchemaJson(plain)
	return nil
}`,

		"model_enhanced.gen.go": `// Code generated by json-schema-postprocess. DO NOT EDIT.

package simple

// This file contains enhanced versions of the original model types

// Circle is a shape with a radius
type Circle struct {
	// Radius corresponds to the JSON schema field "radius".
	Radius float64 ` + "`json:\"radius\" yaml:\"radius\" mapstructure:\"radius\"`" + `

	// Type corresponds to the JSON schema field "type".
	Type string ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
}

// Square is a shape with a side length
type Square struct {
	// Side corresponds to the JSON schema field "side".
	Side float64 ` + "`json:\"side\" yaml:\"side\" mapstructure:\"side\"`" + `

	// Type corresponds to the JSON schema field "type".
	Type string ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
}

// Triangle is a shape with a base and height
type Triangle struct {
	// Base corresponds to the JSON schema field "base".
	Base float64 ` + "`json:\"base\" yaml:\"base\" mapstructure:\"base\"`" + `

	// Height corresponds to the JSON schema field "height".
	Height float64 ` + "`json:\"height\" yaml:\"height\" mapstructure:\"height\"`" + `

	// Type corresponds to the JSON schema field "type".
	Type string ` + "`json:\"type\" yaml:\"type\" mapstructure:\"type\"`" + `
}

// SimpleSchemaJson represents the top-level schema
type SimpleSchemaJson struct {
	// Config corresponds to the JSON schema field "config".
	Config *SimpleSchemaJsonConfig ` + "`json:\"config,omitempty\" yaml:\"config,omitempty\" mapstructure:\"config,omitempty\"`" + `

	// Shapes corresponds to the JSON schema field "shapes".
	Shapes ShapeSlice ` + "`json:\"shapes,omitempty\" yaml:\"shapes,omitempty\" mapstructure:\"shapes,omitempty\"`" + `
}

// SimpleSchemaJsonConfig holds the configuration
type SimpleSchemaJsonConfig struct {
	// Shape corresponds to the JSON schema field "shape".
	Shape Shape ` + "`json:\"shape\" yaml:\"shape\" mapstructure:\"shape\"`" + `
}`,
	}

	// Verify output files were created and match expectations
	for filename, expectedContent := range expectedFiles {
		filePath := filepath.Join(outputDir, filename)

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected output file not created: %s", filePath)
			continue
		}

		// Read generated file
		actualContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read output file %s: %v", filePath, err)
			continue
		}

		// Normalize content to handle whitespace differences
		normalizedActual := normalizeContent(string(actualContent))
		normalizedExpected := normalizeContent(expectedContent)

		// Compare content
		if normalizedActual != normalizedExpected {
			// On failure, write files for comparison
			diffDir := filepath.Join(tmpDir, "diff")
			os.MkdirAll(diffDir, 0755)

			actualFile := filepath.Join(diffDir, "actual_"+filename)
			expectedFile := filepath.Join(diffDir, "expected_"+filename)

			os.WriteFile(actualFile, actualContent, 0644)
			os.WriteFile(expectedFile, []byte(expectedContent), 0644)

			// Create diff output
			diff := generateFileDiff(normalizedExpected, normalizedActual)

			t.Errorf("Generated file %s doesn't match expected content\nDiff:\n%s\nFiles written to %s for comparison",
				filename, diff, diffDir)
		}
	}
}

// normalizeContent removes whitespace variations for more robust comparison
func normalizeContent(content string) string {
	// Remove all whitespace
	content = strings.ReplaceAll(content, " ", "")
	content = strings.ReplaceAll(content, "\t", "")
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "\r", "")
	return content
}

// generateFileDiff creates a simple line-by-line diff between expected and actual content
func generateFileDiff(expected, actual string) string {
	if expected == actual {
		return "No differences"
	}

	// Simple diff implementation - in a real environment, you might use a more sophisticated diff algorithm
	expectedLines := strings.Split(expected, "")
	actualLines := strings.Split(actual, "")

	minLen := len(expectedLines)
	if len(actualLines) < minLen {
		minLen = len(actualLines)
	}

	var diff strings.Builder
	diff.WriteString("Character differences:\n")

	// Find first difference
	firstDiff := -1
	for i := 0; i < minLen; i++ {
		if expectedLines[i] != actualLines[i] {
			firstDiff = i
			break
		}
	}

	if firstDiff >= 0 {
		start := firstDiff - 10
		if start < 0 {
			start = 0
		}
		end := firstDiff + 10
		if end > minLen {
			end = minLen
		}

		expectedContext := strings.Join(expectedLines[start:end], "")
		actualContext := strings.Join(actualLines[start:end], "")

		diff.WriteString(fmt.Sprintf("First difference at position %d:\n", firstDiff))
		diff.WriteString(fmt.Sprintf("Expected: ...%s...\n", expectedContext))
		diff.WriteString(fmt.Sprintf("Actual:   ...%s...\n", actualContext))
	}

	// Report length differences
	if len(expectedLines) != len(actualLines) {
		diff.WriteString(fmt.Sprintf("\nLength difference: expected %d, got %d characters\n",
			len(expectedLines), len(actualLines)))
	}

	return diff.String()
}
