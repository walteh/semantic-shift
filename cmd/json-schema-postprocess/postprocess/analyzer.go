package postprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SchemaAnalyzer is responsible for analyzing the JSON schema
// and identifying the structure of "parents" and their "children"
type SchemaAnalyzer struct {
	schemaData map[string]interface{}
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
	Name         string
	ChildrenRefs []string
	Children     []string
	ConstField   string
	ConstValues  map[string]string
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
		return nil, fmt.Errorf("reading schema file: %w", err)
	}

	var schemaData map[string]interface{}
	if err := json.Unmarshal(schemaFile, &schemaData); err != nil {
		return nil, fmt.Errorf("parsing schema JSON: %w", err)
	}

	return &SchemaAnalyzer{
		schemaData: schemaData,
	}, nil
}

// Analyze performs the schema analysis and returns the results
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
		return nil, fmt.Errorf("schema doesn't have definitions section")
	}

	// Step 1: Identify parents (schemas with anyOf)
	for defName, defObj := range definitions {
		defMap, ok := defObj.(map[string]interface{})
		if !ok {
			continue
		}

		anyOf, ok := defMap["anyOf"].([]interface{})
		if !ok || len(anyOf) == 0 {
			continue
		}

		// This is a parent schema
		parent := ParentInfo{
			Name:         defName,
			ChildrenRefs: make([]string, 0, len(anyOf)),
			Children:     make([]string, 0, len(anyOf)),
			ConstValues:  make(map[string]string),
		}

		// Extract the child references
		for _, childRef := range anyOf {
			childRefMap, ok := childRef.(map[string]interface{})
			if !ok {
				continue
			}

			ref, ok := childRefMap["$ref"].(string)
			if !ok {
				continue
			}

			parent.ChildrenRefs = append(parent.ChildrenRefs, ref)
			childName := extractRefName(ref)
			parent.Children = append(parent.Children, childName)
		}

		results.Parents[defName] = parent
	}

	// Step 2: Analyze children for common constant fields
	for parentName, parentInfo := range results.Parents {
		if len(parentInfo.Children) == 0 {
			continue
		}

		// Get the first child to check for fields
		firstChildDef, ok := definitions[parentInfo.Children[0]]
		if !ok {
			continue
		}

		firstChildMap, ok := firstChildDef.(map[string]interface{})
		if !ok {
			continue
		}

		properties, ok := firstChildMap["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check each property in the first child for const fields
		for propName, propVal := range properties {
			propMap, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}

			_, hasConst := propMap["const"]
			if !hasConst {
				continue
			}

			// Found a const field, check if all children have it
			allHaveField := true
			constValues := make(map[string]string)

			for _, childName := range parentInfo.Children {
				childDef, ok := definitions[childName]
				if !ok {
					allHaveField = false
					break
				}

				childMap, ok := childDef.(map[string]interface{})
				if !ok {
					allHaveField = false
					break
				}

				childProps, ok := childMap["properties"].(map[string]interface{})
				if !ok {
					allHaveField = false
					break
				}

				childProp, ok := childProps[propName]
				if !ok {
					allHaveField = false
					break
				}

				childPropMap, ok := childProp.(map[string]interface{})
				if !ok {
					allHaveField = false
					break
				}

				constVal, ok := childPropMap["const"]
				if !ok {
					allHaveField = false
					break
				}

				constStr, ok := constVal.(string)
				if !ok {
					allHaveField = false
					break
				}

				constValues[childName] = constStr
			}

			if allHaveField {
				// We found a common constant field in all children
				parent := results.Parents[parentName]
				parent.ConstField = propName
				parent.ConstValues = constValues
				results.Parents[parentName] = parent
				results.ConstantFieldNames[parentName] = propName
				break
			}
		}
	}

	// Step 3: Find all references to parents
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
				itemRef, ok := items["$ref"].(string)
				if ok {
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

			// Could add map reference check here for future use
		}
	}

	return results, nil
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
