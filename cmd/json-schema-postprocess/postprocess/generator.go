package postprocess

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// CodeGenerator is responsible for generating updated Go code
// based on the schema analysis results
type CodeGenerator struct {
	sourceFile string
	outputFile string
	results    *SchemaResults
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(sourceFile, outputFile string, results *SchemaResults) *CodeGenerator {
	return &CodeGenerator{
		sourceFile: sourceFile,
		outputFile: outputFile,
		results:    results,
	}
}

// Generate performs the code generation
func (cg *CodeGenerator) Generate() error {
	// Parse the source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, cg.sourceFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing source file: %w", err)
	}

	// Get the package name
	packageName := node.Name.Name

	// Generate code for each parent type
	parentCodes := make(map[string]string)
	for parentName, parentInfo := range cg.results.Parents {
		code, err := cg.generateParentCode(parentName, parentInfo)
		if err != nil {
			return fmt.Errorf("generating code for parent %s: %w", parentName, err)
		}
		parentCodes[parentName] = code
	}

	// Generate parse functions for each parent type
	parseFuncCodes := make(map[string]string)
	for parentName, parentInfo := range cg.results.Parents {
		code, err := cg.generateParseFunc(parentName, parentInfo)
		if err != nil {
			return fmt.Errorf("generating parse function for parent %s: %w", parentName, err)
		}
		parseFuncCodes[parentName] = code
	}

	// Generate marshal functions for children with constant fields
	marshalFuncCodes := make(map[string]string)
	for _, parentInfo := range cg.results.Parents {
		if parentInfo.ConstField == "" {
			continue
		}

		for _, childName := range parentInfo.Children {
			code, err := cg.generateMarshalFunc(childName, parentInfo)
			if err != nil {
				return fmt.Errorf("generating marshal function for child %s: %w", childName, err)
			}
			marshalFuncCodes[childName] = code
		}
	}

	// Generate unmarshal overrides for parent callers
	unmarshalOverrideCodes := make(map[string]string)
	for _, callerInfo := range cg.results.ParentCallers {
		code, err := cg.generateUnmarshalOverride(callerInfo)
		if err != nil {
			return fmt.Errorf("generating unmarshal override for caller %s: %w", callerInfo.Name, err)
		}
		unmarshalOverrideCodes[callerInfo.Name] = code
	}

	// Assemble all generated code
	outputCode := fmt.Sprintf("// Code generated by json-schema-postprocess, DO NOT EDIT.\n\n")
	outputCode += fmt.Sprintf("package %s\n\n", packageName)
	outputCode += "import (\n\t\"encoding/json\"\n\t\"fmt\"\n)\n\n"

	// Add parent interface code
	for _, code := range parentCodes {
		outputCode += code + "\n\n"
	}

	// Add parse functions
	for _, code := range parseFuncCodes {
		outputCode += code + "\n\n"
	}

	// Add marshal functions
	for _, code := range marshalFuncCodes {
		outputCode += code + "\n\n"
	}

	// Add unmarshal override functions
	for _, code := range unmarshalOverrideCodes {
		if code != "" {
			outputCode += code + "\n\n"
		}
	}

	// Format the generated code
	formattedBytes, err := format.Source([]byte(outputCode))
	if err != nil {
		return fmt.Errorf("formatting generated code: %w", err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(cg.outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write the generated code to the output file
	if err := os.WriteFile(cg.outputFile, formattedBytes, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	return nil
}

// generateParentCode generates the interface and related code for a parent type
func (cg *CodeGenerator) generateParentCode(parentName string, parentInfo ParentInfo) (string, error) {
	var code strings.Builder

	// Generate interface
	code.WriteString(fmt.Sprintf("type %s interface {\n", parentName))
	code.WriteString(fmt.Sprintf("\tis%s()\n", parentName))

	// Add Model() method if we have a constant field
	if parentInfo.ConstField != "" {
		constFieldName := strings.Title(parentInfo.ConstField)
		code.WriteString(fmt.Sprintf("\t%s() %sModel\n", constFieldName, parentName))
	}

	code.WriteString("}\n\n")

	// Generate slice and map types
	code.WriteString(fmt.Sprintf("type %sSlice []%s\n", parentName, parentName))
	code.WriteString(fmt.Sprintf("type %sMap map[string]%s\n\n", parentName, parentName))

	// Generate interface implementation methods for each child
	for _, childName := range parentInfo.Children {
		code.WriteString(fmt.Sprintf("func (me *%s) is%s() {}\n", childName, parentName))
	}

	// Generate model type and constants if we have a constant field
	if parentInfo.ConstField != "" {
		constFieldName := strings.Title(parentInfo.ConstField)
		code.WriteString(fmt.Sprintf("\ntype %sModel string\n\n", parentName))
		code.WriteString("const (\n")

		// Add constants for each child's constant value
		for _, constValue := range parentInfo.ConstValues {
			// Sanitize const value for use as an identifier
			sanitizedValue := sanitizeIdentifier(constValue)
			constName := fmt.Sprintf("%sModel%s", parentName, strings.Title(sanitizedValue))
			code.WriteString(fmt.Sprintf("\t%s %sModel = \"%s\"\n", constName, parentName, constValue))
		}
		code.WriteString(")\n\n")

		// Generate Model() methods for each child
		for childName, constValue := range parentInfo.ConstValues {
			// Sanitize const value for use as an identifier
			sanitizedValue := sanitizeIdentifier(constValue)
			constName := fmt.Sprintf("%sModel%s", parentName, strings.Title(sanitizedValue))
			code.WriteString(fmt.Sprintf("func (me *%s) %s() %sModel { return %s }\n",
				childName, constFieldName, parentName, constName))
		}
	}

	return code.String(), nil
}

// sanitizeIdentifier makes sure a string can be used as part of a Go identifier
func sanitizeIdentifier(s string) string {
	// Replace dashes and other non-alphanumeric chars with underscores
	result := strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' || r == '.' || !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return '_'
		}
		return r
	}, s)

	// If it starts with a number, prefix with underscore
	if len(result) > 0 && unicode.IsNumber(rune(result[0])) {
		result = "_" + result
	}

	return result
}

// generateParseFunc generates the parse function for a parent type
func (cg *CodeGenerator) generateParseFunc(parentName string, parentInfo ParentInfo) (string, error) {
	var code strings.Builder

	funcName := fmt.Sprintf("parseUnknown%s", parentName)
	code.WriteString(fmt.Sprintf("func %s(b interface{}) (%s, error) {\n", funcName, parentName))
	code.WriteString("\tstr, err := json.Marshal(b)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn nil, err\n")
	code.WriteString("\t}\n\n")

	// If we have a constant field, use it to determine which child to parse
	if parentInfo.ConstField != "" {
		constFieldName := strings.Title(parentInfo.ConstField)
		code.WriteString("\ttype Plain struct {\n")
		code.WriteString(fmt.Sprintf("\t\t%s %sModel `json:\"%s\" yaml:\"%s\" mapstructure:\"%s\"`\n",
			constFieldName, parentName, parentInfo.ConstField, parentInfo.ConstField, parentInfo.ConstField))
		code.WriteString("\t}\n")
		code.WriteString("\tvar plain Plain\n")
		code.WriteString("\tif err := json.Unmarshal(str, &plain); err != nil {\n")
		code.WriteString("\t\treturn nil, err\n")
		code.WriteString("\t}\n\n")
		code.WriteString(fmt.Sprintf("\tswitch plain.%s {\n", constFieldName))

		// Add a case for each child type
		for childName, constValue := range parentInfo.ConstValues {
			// Sanitize const value for use as an identifier
			sanitizedValue := sanitizeIdentifier(constValue)
			constName := fmt.Sprintf("%sModel%s", parentName, strings.Title(sanitizedValue))
			code.WriteString(fmt.Sprintf("\tcase %s:\n", constName))
			code.WriteString(fmt.Sprintf("\t\tvar %s %s\n", strings.ToLower(childName), childName))
			code.WriteString(fmt.Sprintf("\t\terr = json.Unmarshal(str, &%s)\n", strings.ToLower(childName)))
			code.WriteString(fmt.Sprintf("\t\treturn &%s, err\n", strings.ToLower(childName)))
		}

		code.WriteString("\tdefault:\n")
		code.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"invalid %s: %%s\", plain.%s)\n",
			strings.ToLower(parentInfo.ConstField), constFieldName))
		code.WriteString("\t}\n")
	} else {
		// Without a constant field, try each child type until one works
		code.WriteString("\topts := []" + parentName + "{\n")
		for _, childName := range parentInfo.Children {
			code.WriteString(fmt.Sprintf("\t\t&%s{},\n", childName))
		}
		code.WriteString("\t}\n\n")
		code.WriteString("\tfor _, opt := range opts {\n")
		code.WriteString("\t\terr = json.Unmarshal(str, opt)\n")
		code.WriteString("\t\tif err == nil {\n")
		code.WriteString("\t\t\treturn opt, nil\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n\n")
		code.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"invalid %s\")\n", strings.ToLower(parentName)))
	}

	code.WriteString("}\n")
	return code.String(), nil
}

// generateMarshalFunc generates the marshal function for a child type with a constant field
func (cg *CodeGenerator) generateMarshalFunc(childName string, parentInfo ParentInfo) (string, error) {
	if parentInfo.ConstField == "" {
		return "", nil // No constant field, no marshal function needed
	}

	var code strings.Builder

	code.WriteString(fmt.Sprintf("func (j %s) MarshalJSON() ([]byte, error) {\n", childName))
	code.WriteString("\t// we need to add the constant field to the output\n")
	code.WriteString(fmt.Sprintf("\ttype Plain %s\n", childName))
	code.WriteString("\tmyMarshal := struct {\n")
	code.WriteString(fmt.Sprintf("\t\t%s %sModel `json:\"%s\" yaml:\"%s\" mapstructure:\"%s\"`\n",
		strings.Title(parentInfo.ConstField), parentInfo.Name, parentInfo.ConstField, parentInfo.ConstField, parentInfo.ConstField))
	code.WriteString("\t\tPlain\n")
	code.WriteString("\t}{\n")
	code.WriteString(fmt.Sprintf("\t\t%s: j.%s(),\n", strings.Title(parentInfo.ConstField), strings.Title(parentInfo.ConstField)))
	code.WriteString("\t\tPlain: Plain(j),\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn json.Marshal(myMarshal)\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generateUnmarshalOverride generates the unmarshal override for a parent caller
func (cg *CodeGenerator) generateUnmarshalOverride(callerInfo ParentCallerInfo) (string, error) {
	var code strings.Builder

	// Generate UnmarshalJSON function signature
	code.WriteString(fmt.Sprintf("func (j *%s) UnmarshalJSON(b []byte) error {\n", callerInfo.Name))
	code.WriteString("\tvar raw map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(b, &raw); err != nil {\n")
	code.WriteString("\t\treturn err\n")
	code.WriteString("\t}\n")

	// Check required fields
	if callerInfo.IsRequired {
		code.WriteString(fmt.Sprintf("\tif _, ok := raw[\"%s\"]; raw != nil && !ok {\n", callerInfo.Field))
		code.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"field %s in %s: required\")\n", callerInfo.Field, callerInfo.Name))
		code.WriteString("\t}\n")
	}

	// Unmarshal the base structure
	code.WriteString("\ttype Plain " + callerInfo.Name + "\n")
	code.WriteString("\tvar plain Plain\n")
	code.WriteString("\tif err := json.Unmarshal(b, &plain); err != nil {\n")
	code.WriteString("\t\treturn err\n")
	code.WriteString("\t}\n\n")

	// Parse the parent field using the appropriate parse function
	parentField := callerInfo.Field
	parentType := callerInfo.ParentRef

	if callerInfo.IsArray {
		// Handle array field
		code.WriteString(fmt.Sprintf("\t// Process array of %s objects\n", parentType))
		code.WriteString(fmt.Sprintf("\tif raw[\"%s\"] != nil {\n", parentField))
		code.WriteString(fmt.Sprintf("\t\tarr, ok := raw[\"%s\"].([]interface{})\n", parentField))
		code.WriteString("\t\tif ok {\n")
		code.WriteString(fmt.Sprintf("\t\t\tresult := make([]%s, len(arr))\n", parentType))
		code.WriteString("\t\t\tfor i, item := range arr {\n")
		code.WriteString(fmt.Sprintf("\t\t\t\tparsed, err := parseUnknown%s(item)\n", parentType))
		code.WriteString("\t\t\t\tif err != nil {\n")
		code.WriteString(fmt.Sprintf("\t\t\t\t\treturn fmt.Errorf(\"parsing item %%d in %s: %%w\", i, err)\n", parentField))
		code.WriteString("\t\t\t\t}\n")
		code.WriteString("\t\t\t\tresult[i] = parsed\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString(fmt.Sprintf("\t\t\tplain.%s = result\n", strings.Title(parentField)))
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n")
	} else {
		// Handle direct field
		code.WriteString(fmt.Sprintf("\t// Process %s object\n", parentType))
		code.WriteString(fmt.Sprintf("\tif raw[\"%s\"] != nil {\n", parentField))
		code.WriteString(fmt.Sprintf("\t\tparsed, err := parseUnknown%s(raw[\"%s\"])\n", parentType, parentField))
		code.WriteString("\t\tif err != nil {\n")
		code.WriteString(fmt.Sprintf("\t\t\treturn fmt.Errorf(\"parsing %s: %%w\", err)\n", parentField))
		code.WriteString("\t\t}\n")
		code.WriteString(fmt.Sprintf("\t\tplain.%s = parsed\n", strings.Title(parentField)))
		code.WriteString("\t}\n")
	}

	// Assign the result back to the original structure
	code.WriteString("\n\t*j = " + callerInfo.Name + "(plain)\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n")

	return code.String(), nil
}
