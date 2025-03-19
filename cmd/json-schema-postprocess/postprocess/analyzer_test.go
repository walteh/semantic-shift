package postprocess

import (
	"path/filepath"
	"testing"
)

func TestSchemaAnalyzer(t *testing.T) {
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

	// Verify parent detection
	if len(results.Parents) == 0 {
		t.Error("No parents found in the schema")
	}

	// Check if Color is identified as a parent
	color, exists := results.Parents["Color"]
	if !exists {
		t.Error("Color not identified as a parent")
	} else {
		// Check if Color has children
		if len(color.Children) == 0 {
			t.Error("Color has no children")
		}

		// Check if the constant field is identified
		if color.ConstField != "model" {
			t.Errorf("Expected constant field 'model' for Color, got '%s'", color.ConstField)
		}

		// Check if the constant values are correctly extracted
		if len(color.ConstValues) == 0 {
			t.Error("No constant values found for Color children")
		}

		// Check for specific children
		found := false
		for _, child := range color.Children {
			if child == "HSLValue" || child == "RGBValue" || child == "HSVValue" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected common color types not found in children")
		}
	}

	// Check if Palette is identified as a parent
	palette, exists := results.Parents["Palette"]
	if !exists {
		t.Error("Palette not identified as a parent")
	} else {
		// Check if Palette has children
		if len(palette.Children) == 0 {
			t.Error("Palette has no children")
		}

		// In our schema, 'type' is actually a valid constant field for Palette
		// Since it distinguishes between different palette types
		if palette.ConstField != "type" {
			t.Errorf("Expected constant field 'type' for Palette, got '%s'", palette.ConstField)
		}

		// Check for specific children
		found := false
		for _, child := range palette.Children {
			if child == "CategoricalPalette" || child == "DiscreteScalePalette" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected palette types not found in children")
		}
	}

	// Verify parent caller detection
	if len(results.ParentCallers) == 0 {
		t.Error("No parent callers found in the schema")
	}

	// Check direct and array callers
	if len(results.DirectParentCallers) == 0 {
		t.Error("No direct parent callers found")
	}

	if len(results.ArrayParentCallers) == 0 {
		t.Error("No array parent callers found")
	}
}
