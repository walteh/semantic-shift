package repostprocess

import (
	"encoding/json"
	"os"
	"strings"

	"gitlab.com/tozd/go/errors"
)

// SchemaAnalyzer is responsible for analyzing the JSON schema
// and identifying the structure of "parents" and their "children"
type SchemaAnalyzer struct {
	schemaData map[string]interface{}
	schemaPath string
}

// SchemaResults contains the analysis results
type SchemaResults struct {
	Parents             map[string]ParentInfo
	ConstantFieldNames  map[string]string
	ParentCallers       map[string]ParentCallerInfo
	DirectParentCallers map[string]ParentCallerInfo
	ArrayParentCallers  map[string]ParentCallerInfo
	MapParentCallers    map[string]ParentCallerInfo
}

// ParentInfo holds information about a parent schema
type ParentInfo struct {
	Name           string
	ChildrenRefs   []string
	Children       []string
	ConstantField  string
	ConstantValues map[string]string
}

// ParentCallerInfo holds information about schemas that reference parents
type ParentCallerInfo struct {
	Name        string
	Field       string
	ParentRef   string
	IsArray     bool
	IsMap       bool
	IsRequired  bool
	ParentNames []string
}

// NewSchemaAnalyzer creates a new schema analyzer
func NewSchemaAnalyzer(schemaPath string) (*SchemaAnalyzer, error) {
	schemaFile, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, errors.Errorf("reading schema file: %w", err)
	}

	var schemaData map[string]interface{}
	if err := json.Unmarshal(schemaFile, &schemaData); err != nil {
		return nil, errors.Errorf("parsing schema JSON: %w", err)
	}

	return &SchemaAnalyzer{
		schemaData: schemaData,
		schemaPath: schemaPath,
	}, nil
}

// NewSchemaResults creates a new SchemaResults with initialized maps
func NewSchemaResults() *SchemaResults {
	return &SchemaResults{
		Parents:             make(map[string]ParentInfo),
		ConstantFieldNames:  make(map[string]string),
		ParentCallers:       make(map[string]ParentCallerInfo),
		DirectParentCallers: make(map[string]ParentCallerInfo),
		ArrayParentCallers:  make(map[string]ParentCallerInfo),
		MapParentCallers:    make(map[string]ParentCallerInfo),
	}
}

// Analyze parses the schema and identifies parents, children, and constant fields
func (sa *SchemaAnalyzer) Analyze() (*SchemaResults, error) {
	results := &SchemaResults{
		Parents:             make(map[string]ParentInfo),
		ConstantFieldNames:  make(map[string]string),
		ParentCallers:       make(map[string]ParentCallerInfo),
		DirectParentCallers: make(map[string]ParentCallerInfo),
		ArrayParentCallers:  make(map[string]ParentCallerInfo),
		MapParentCallers:    make(map[string]ParentCallerInfo),
	}

	// Get the definitions section from the schema
	definitions, ok := sa.schemaData["definitions"].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("schema doesn't have definitions section")
	}

	// Step 1: Identify parents (schemas with anyOf)
	if err := sa.identifyParents(definitions, results); err != nil {
		return nil, err
	}

	// Step 2: Analyze children for common constant fields
	if err := sa.identifyConstantFields(definitions, results); err != nil {
		return nil, err
	}

	// Step 3: Find all references to parents
	if err := sa.identifyParentCallers(definitions, results); err != nil {
		return nil, err
	}

	// Step 4: Check for parent callers at the top level
	if topProps, ok := sa.schemaData["properties"].(map[string]interface{}); ok {
		if err := sa.identifyTopLevelParentCallers(topProps, results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// identifyTopLevelParentCallers looks for references to parent types in the top level properties
func (sa *SchemaAnalyzer) identifyTopLevelParentCallers(properties map[string]interface{}, results *SchemaResults) error {
	// Track parent callers in the top level properties
	for propName, propVal := range properties {
		propMap, ok := propVal.(map[string]interface{})
		if !ok {
			// Check if this is a nested object with its own properties
			if nestedObj, ok := propVal.(map[string]interface{}); ok {
				if props, ok := nestedObj["properties"].(map[string]interface{}); ok {
					// This is a nested object, check its properties for parent callers
					if err := sa.identifyNestedParentCallers(propName, props, results); err != nil {
						return err
					}
				}
			}
			continue
		}

		// Check for direct reference
		ref, hasRef := propMap["$ref"].(string)
		if hasRef {
			refName := extractRefName(ref)
			if _, isParent := results.Parents[refName]; isParent {
				// This is a direct parent caller at top level
				caller := ParentCallerInfo{
					Name:        propName, // Use property name as the struct name
					Field:       propName, // Field is the same as name at top level
					ParentRef:   refName,
					IsRequired:  true, // Assume required at top level
					ParentNames: []string{refName},
				}
				results.ParentCallers[propName+"."+propName] = caller
				results.DirectParentCallers[propName+"."+propName] = caller
			}
			continue
		}

		// Check for array reference
		items, ok := propMap["items"].(map[string]interface{})
		if ok {
			if itemRef, ok := items["$ref"].(string); ok {
				refName := extractRefName(itemRef)
				if _, isParent := results.Parents[refName]; isParent {
					// This is an array parent caller at top level
					caller := ParentCallerInfo{
						Name:        "", // Empty name for top level
						Field:       propName,
						ParentRef:   refName,
						IsArray:     true,
						IsRequired:  true, // Assume required at top level
						ParentNames: []string{refName},
					}
					results.ParentCallers[propName] = caller
					results.ArrayParentCallers[propName] = caller
				}
			}
		}
	}

	return nil
}

// identifyNestedParentCallers looks for references to parent types in nested object properties
func (sa *SchemaAnalyzer) identifyNestedParentCallers(parentName string, properties map[string]interface{}, results *SchemaResults) error {
	// Check each property in the nested object
	for propName, propVal := range properties {
		propMap, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for direct reference
		ref, hasRef := propMap["$ref"].(string)
		if hasRef {
			refName := extractRefName(ref)
			if _, isParent := results.Parents[refName]; isParent {
				// This is a direct parent caller in a nested object
				caller := ParentCallerInfo{
					Name:        parentName, // Parent object name
					Field:       propName,   // Field within the parent
					ParentRef:   refName,
					IsRequired:  true, // Assume required
					ParentNames: []string{refName},
				}
				results.ParentCallers[parentName+"."+propName] = caller
				results.DirectParentCallers[parentName+"."+propName] = caller
			}
			continue
		}

		// Check for array reference
		items, ok := propMap["items"].(map[string]interface{})
		if ok {
			if itemRef, ok := items["$ref"].(string); ok {
				refName := extractRefName(itemRef)
				if _, isParent := results.Parents[refName]; isParent {
					// This is an array parent caller in a nested object
					caller := ParentCallerInfo{
						Name:        parentName, // Parent object name
						Field:       propName,   // Field within the parent
						ParentRef:   refName,
						IsArray:     true,
						IsRequired:  true, // Assume required
						ParentNames: []string{refName},
					}
					results.ParentCallers[parentName+"."+propName] = caller
					results.ArrayParentCallers[parentName+"."+propName] = caller
				}
			}
		}
	}

	return nil
}

// identifyParentCallers identifies schemas that refer to parent schemas
func (sa *SchemaAnalyzer) identifyParentCallers(definitions map[string]interface{}, results *SchemaResults) error {
	for defName, defObj := range definitions {
		defMap, ok := defObj.(map[string]interface{})
		if !ok {
			continue
		}

		properties, ok := defMap["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check required fields
		required := make(map[string]bool)
		requiredArr, ok := defMap["required"].([]interface{})
		if ok {
			for _, req := range requiredArr {
				reqStr, ok := req.(string)
				if ok {
					required[reqStr] = true
				}
			}
		}

		// Check each property for references to parents
		for propName, propVal := range properties {
			propMap, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}

			// Check for direct reference
			ref, hasRef := propMap["$ref"].(string)
			if hasRef {
				refName := extractRefName(ref)
				if _, isParent := results.Parents[refName]; isParent {
					// This is a direct parent caller
					caller := ParentCallerInfo{
						Name:        defName,
						Field:       propName,
						ParentRef:   refName,
						IsRequired:  required[propName],
						ParentNames: []string{refName},
					}
					results.ParentCallers[defName+"."+propName] = caller
					results.DirectParentCallers[defName+"."+propName] = caller
				}
				continue
			}

			// Check for array reference
			items, ok := propMap["items"].(map[string]interface{})
			if ok {
				// Check for nested array
				if nestedItems, ok := items["items"].(map[string]interface{}); ok {
					// This is a nested array
					if itemRef, ok := nestedItems["$ref"].(string); ok {
						refName := extractRefName(itemRef)
						if _, isParent := results.Parents[refName]; isParent {
							// This is a nested array parent caller
							caller := ParentCallerInfo{
								Name:        defName,
								Field:       propName,
								ParentRef:   refName,
								IsArray:     true,
								IsRequired:  required[propName],
								ParentNames: []string{refName},
							}
							results.ParentCallers[defName+"."+propName] = caller
							results.ArrayParentCallers[defName+"."+propName] = caller
						}
					}
				} else {
					// Check for single array reference
					if itemRef, ok := items["$ref"].(string); ok {
						refName := extractRefName(itemRef)
						if _, isParent := results.Parents[refName]; isParent {
							// This is an array parent caller
							caller := ParentCallerInfo{
								Name:        defName,
								Field:       propName,
								ParentRef:   refName,
								IsArray:     true,
								IsRequired:  required[propName],
								ParentNames: []string{refName},
							}
							results.ParentCallers[defName+"."+propName] = caller
							results.ArrayParentCallers[defName+"."+propName] = caller
						}
					}
				}
			}

			// Check for map reference
			additionalProperties, ok := propMap["additionalProperties"].(map[string]interface{})
			if ok {
				if itemRef, ok := additionalProperties["$ref"].(string); ok {
					refName := extractRefName(itemRef)
					if _, isParent := results.Parents[refName]; isParent {
						// This is a map parent caller
						caller := ParentCallerInfo{
							Name:        defName,
							Field:       propName,
							ParentRef:   refName,
							IsMap:       true,
							IsRequired:  required[propName],
							ParentNames: []string{refName},
						}
						results.ParentCallers[defName+"."+propName] = caller
						results.MapParentCallers[defName+"."+propName] = caller
					}
				}
			}
		}
	}
	return nil
}

// identifyParents identifies schemas with anyOf that are parent types
func (sa *SchemaAnalyzer) identifyParents(definitions map[string]interface{}, results *SchemaResults) error {
	for defName, defObj := range definitions {
		defMap, ok := defObj.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for anyOf which indicates a parent type
		anyOf, ok := defMap["anyOf"].([]interface{})
		if !ok {
			continue
		}

		// This is a parent schema with multiple child options
		parent := ParentInfo{
			Name:           defName,
			ChildrenRefs:   []string{},
			Children:       []string{},
			ConstantValues: make(map[string]string),
		}

		// Extract references to children
		for _, child := range anyOf {
			childMap, ok := child.(map[string]interface{})
			if !ok {
				continue
			}

			ref, ok := childMap["$ref"].(string)
			if !ok {
				continue
			}

			childName := extractRefName(ref)
			parent.ChildrenRefs = append(parent.ChildrenRefs, childName)
			parent.Children = append(parent.Children, childName)
		}

		if len(parent.Children) > 0 {
			results.Parents[defName] = parent
		}
	}

	return nil
}

// identifyConstantFields analyzes children schemas to find common "type" or other constant fields
func (sa *SchemaAnalyzer) identifyConstantFields(definitions map[string]interface{}, results *SchemaResults) error {
	// For each parent, check its children for constant fields
	for parentName, parent := range results.Parents {
		if len(parent.Children) == 0 {
			continue
		}

		// Check first child for potential constant fields
		firstChildName := parent.Children[0]
		firstChild, ok := definitions[firstChildName].(map[string]interface{})
		if !ok {
			continue
		}

		properties, ok := firstChild["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check each property to see if it's a potential constant field
		for propName, propVal := range properties {
			propMap, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if this property has enum or const, which indicates it could be a constant field
			_, hasEnum := propMap["enum"]
			_, hasConst := propMap["const"]

			// If this is a potential constant field, check all children to see if they have it
			if hasEnum || hasConst || propName == "type" {
				allHaveIt := true
				potentialValues := make(map[string]string)

				// Check if all children have this property as a constant
				for _, childName := range parent.Children {
					child, ok := definitions[childName].(map[string]interface{})
					if !ok {
						allHaveIt = false
						break
					}

					childProps, ok := child["properties"].(map[string]interface{})
					if !ok {
						allHaveIt = false
						break
					}

					childProp, ok := childProps[propName].(map[string]interface{})
					if !ok {
						allHaveIt = false
						break
					}

					// Check for enum or const value
					var constantValue string
					if enum, ok := childProp["enum"].([]interface{}); ok && len(enum) == 1 {
						if strVal, ok := enum[0].(string); ok {
							constantValue = strVal
						}
					} else if constVal, ok := childProp["const"]; ok {
						if strVal, ok := constVal.(string); ok {
							constantValue = strVal
						}
					} else if propName == "type" && childProp["type"] == "string" {
						// For "type" fields, use the child name as a default value
						constantValue = strings.ToLower(childName)
					}

					if constantValue == "" {
						allHaveIt = false
						break
					}

					potentialValues[childName] = constantValue
				}

				if allHaveIt && len(potentialValues) == len(parent.Children) {
					// This is a valid constant field, update the parent info
					updatedParent := parent
					updatedParent.ConstantField = propName
					updatedParent.ConstantValues = potentialValues
					results.Parents[parentName] = updatedParent
					results.ConstantFieldNames[parentName] = propName
					break // Found a constant field, no need to check others
				}
			}
		}
	}

	return nil
}

// extractRefName extracts the schema name from a reference string
// Example: "#/definitions/Color" -> "Color"
func extractRefName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
