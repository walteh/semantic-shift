# JSON Schema Postprocessor

This tool processes Go code generated from JSON schemas to enhance it with proper interfaces, type handling, and marshaling/unmarshaling functionality.

## Overview

The JSON Schema Postprocessor analyzes a JSON schema and its generated Go code to:

1. Identify schemas that consist entirely of a list of `anyOf` references (called "parents")
2. Find the references of each parent (called "children")
3. Identify if children of a given parent share a common 'const' string field
4. Find schemas that refer to the parents (called "parent-callers")
5. Categorize parent-callers as direct, array, or map callers

Based on this analysis, it generates enhanced Go code that:

-   Converts parents into interfaces
-   Adds proper type assertions
-   Improves marshaling/unmarshaling with discriminator fields
-   Makes polymorphic collections easier to work with

## Usage

```
json-schema-postprocess -schema=<schema-file> -model=<model-file> -output=<output-dir>
```

Arguments:

-   `-schema`: Path to the JSON schema file
-   `-model`: Path to the generated Go model file
-   `-output`: Directory where the enhanced code will be written

## Example

For a schema with:

```json
"Color": {
  "anyOf": [
    { "$ref": "#/definitions/HSLValue" },
    { "$ref": "#/definitions/RGBValue" }
  ]
}

"HSLValue": {
  "properties": {
    "model": {
      "const": "hsl",
      "type": "string"
    },
    // other properties
  }
}

"RGBValue": {
  "properties": {
    "model": {
      "const": "rgb",
      "type": "string"
    },
    // other properties
  }
}
```

The postprocessor will generate:

1. An interface for `Color`
2. Helper methods for type identification
3. Constants for model values
4. Custom marshal/unmarshal functions that properly handle the discriminator field

## Implementation

The postprocessor is organized into the following modules:

-   `analyzer.go`: Analyzes the JSON schema to identify patterns
-   `generator.go`: Generates enhanced Go code
-   `processor.go`: Coordinates the workflow
-   `main.go`: Command-line interface
