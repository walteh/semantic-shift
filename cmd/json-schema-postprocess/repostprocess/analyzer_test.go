package repostprocess

import (
	"path/filepath"
	"testing"
)

func TestSchemaAnalyzer_Identify(t *testing.T) {
	// Get the test schema path
	schemaPath := filepath.Join("testdata", "color.schema.json")

	// Create analyzer
	analyzer, err := NewSchemaAnalyzer(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	// Run analysis
	results, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Failed to analyze schema: %v", err)
	}

	// Test: Identify schemas that entirely consist of a list of anyOfs (parents)
	if len(results.Parents) == 0 {
		t.Error("No parents found in the schema")
	}

	// Test: Check if Color is identified as a parent
	color, exists := results.Parents["Color"]
	if !exists {
		t.Error("Color not identified as a parent")
	}

	// Test: Identify references of each parent (children)
	if len(color.Children) == 0 {
		t.Error("Color has no children")
	}

	// Test: Identify if each child has a shared constant field
	if color.ConstantField != "model" {
		t.Errorf("Expected constant field 'model' for Color, got '%s'", color.ConstantField)
	}

	// Test: Check constant values
	if len(color.ConstantValues) == 0 {
		t.Error("No constant values found for Color children")
	}

	// Test: Identify specific children
	expectedChildren := []string{"HSLValue", "HSVValue", "HSIValue", "RGBValue", "RGBAValue", "LABValue", "LCHValue", "CMYKValue"}
	for _, expectedChild := range expectedChildren {
		found := false
		for _, child := range color.Children {
			if child == expectedChild {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected child %s not found in Color children", expectedChild)
		}
	}

	// Test: Check Palette (a parent without constant fields)
	if _, exists := results.Parents["Palette"]; !exists {
		t.Error("Palette not identified as a parent")
	}

	// Test: Identify parent callers
	if len(results.ParentCallers) == 0 {
		t.Error("No parent callers found in the schema")
	}

	// Test: Check direct parent callers
	if len(results.DirectParentCallers) == 0 {
		t.Error("No direct parent callers found")
	}

	// Test: Check array parent callers
	if len(results.ArrayParentCallers) == 0 {
		t.Error("No array parent callers found")
	}

	// Check specific parent-caller relationships
	colorConfig, exists := results.DirectParentCallers["ColorConfig.value"]
	if !exists {
		t.Error("ColorConfig.value not identified as direct parent caller for Color")
	} else if colorConfig.ParentRef != "Color" {
		t.Errorf("ColorConfig.value references %s, expected Color", colorConfig.ParentRef)
	}

	// Check for array parent caller
	categoricalPalette, exists := results.ArrayParentCallers["CategoricalPalette.colors"]
	if !exists {
		t.Error("CategoricalPalette.colors not identified as array parent caller")
	} else if categoricalPalette.ParentRef != "Color" {
		t.Errorf("CategoricalPalette.colors references %s, expected Color", categoricalPalette.ParentRef)
	}
}
