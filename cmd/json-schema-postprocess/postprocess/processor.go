package postprocess

import (
	"fmt"
)

// Processor coordinates the schema analysis and code generation
type Processor struct {
	SchemaPath    string
	ModelPath     string
	OutputDirPath string
}

// NewProcessor creates a new processor
func NewProcessor(schemaPath, modelPath, outputDirPath string) *Processor {
	return &Processor{
		SchemaPath:    schemaPath,
		ModelPath:     modelPath,
		OutputDirPath: outputDirPath,
	}
}

// Process performs the entire postprocessing workflow
func (p *Processor) Process() error {
	// Step 1: Analyze the schema
	analyzer, err := NewSchemaAnalyzer(p.SchemaPath)
	if err != nil {
		return fmt.Errorf("creating schema analyzer: %w", err)
	}

	results, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("analyzing schema: %w", err)
	}

	// Step 2: Generate all the code
	generator := NewCodeGenerator(p.ModelPath, p.OutputDirPath, results)
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	return nil
}
