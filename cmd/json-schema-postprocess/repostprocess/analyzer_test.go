package repostprocess

import (
	"path/filepath"
	"testing"
)

func TestSchemaAnalyzer_Identify(t *testing.T) {
	// Create analyzer for color schema
	schemaPath := filepath.Join("testdata", "color", "color.schema.json")
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	// Run the analysis
	results, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Check the parents were correctly identified
	if _, ok := results.Parents["Color"]; !ok {
		t.Error("Expected Color parent not identified")
	}

	if _, ok := results.Parents["Palette"]; !ok {
		t.Error("Expected Palette parent not identified")
	}

	// Check that model field was correctly identified
	colorInfo := results.Parents["Color"]
	if colorInfo.ConstantField != "model" {
		t.Errorf("Expected Color.ConstantField to be 'model', got '%s'", colorInfo.ConstantField)
	}
}

func TestSchemaAnalyzer_SimpleSchema(t *testing.T) {
	// Create analyzer for simple schema
	schemaPath := filepath.Join("testdata", "simple", "simple.schema.json")
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	// Run the analysis
	results, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Check the parents were correctly identified
	if _, ok := results.Parents["Shape"]; !ok {
		t.Error("Expected Shape parent not identified")
	}

	// Check that type field was correctly identified
	shapeInfo := results.Parents["Shape"]
	if shapeInfo.ConstantField != "type" {
		t.Errorf("Expected Shape.ConstantField to be 'type', got '%s'", shapeInfo.ConstantField)
	}

	// Check children
	expectedChildren := []string{"Circle", "Square", "Triangle"}
	for _, child := range expectedChildren {
		found := false
		for _, actualChild := range shapeInfo.Children {
			if actualChild == child {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected child %s not found in Shape children", child)
		}
	}

	// Check constant values
	expectedValues := map[string]string{
		"Circle":   "circle",
		"Square":   "square",
		"Triangle": "triangle",
	}

	for child, expectedValue := range expectedValues {
		actualValue, ok := shapeInfo.ConstantValues[child]
		if !ok {
			t.Errorf("No constant value found for %s", child)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected constant value for %s to be '%s', got '%s'",
				child, expectedValue, actualValue)
		}
	}

	// Check parent callers
	directCallerFound := false
	for _, caller := range results.DirectParentCallers {
		if caller.ParentRef == "Shape" {
			if caller.Name == "config" && caller.Field == "shape" {
				directCallerFound = true
				break
			}
		}
	}
	if !directCallerFound {
		t.Error("Expected direct parent caller config.shape not found")
	}

	arrayCallerFound := false
	for _, caller := range results.ArrayParentCallers {
		if caller.ParentRef == "Shape" {
			if caller.Field == "shapes" {
				arrayCallerFound = true
				break
			}
		}
	}
	if !arrayCallerFound {
		t.Error("Expected array parent caller for shapes not found")
	}
}
