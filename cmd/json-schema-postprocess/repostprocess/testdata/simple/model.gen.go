// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.

package simple

import "encoding/json"
import "fmt"

type Circle struct {
	// Radius corresponds to the JSON schema field "radius".
	Radius float64 `json:"radius" yaml:"radius" mapstructure:"radius"`

	// Type corresponds to the JSON schema field "type".
	Type string `json:"type" yaml:"type" mapstructure:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Circle) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["radius"]; raw != nil && !ok {
		return fmt.Errorf("field radius in Circle: required")
	}
	if _, ok := raw["type"]; raw != nil && !ok {
		return fmt.Errorf("field type in Circle: required")
	}
	type Plain Circle
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Circle(plain)
	return nil
}

type Shape map[string]interface{}

type SimpleSchemaJson struct {
	// Config corresponds to the JSON schema field "config".
	Config *SimpleSchemaJsonConfig `json:"config,omitempty" yaml:"config,omitempty" mapstructure:"config,omitempty"`

	// Shapes corresponds to the JSON schema field "shapes".
	Shapes []Shape `json:"shapes,omitempty" yaml:"shapes,omitempty" mapstructure:"shapes,omitempty"`
}

type SimpleSchemaJsonConfig struct {
	// Shape corresponds to the JSON schema field "shape".
	Shape Shape `json:"shape" yaml:"shape" mapstructure:"shape"`
}

// UnmarshalJSON implements json.Unmarshaler.
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
	*j = SimpleSchemaJsonConfig(plain)
	return nil
}

type Square struct {
	// Side corresponds to the JSON schema field "side".
	Side float64 `json:"side" yaml:"side" mapstructure:"side"`

	// Type corresponds to the JSON schema field "type".
	Type string `json:"type" yaml:"type" mapstructure:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Square) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["side"]; raw != nil && !ok {
		return fmt.Errorf("field side in Square: required")
	}
	if _, ok := raw["type"]; raw != nil && !ok {
		return fmt.Errorf("field type in Square: required")
	}
	type Plain Square
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Square(plain)
	return nil
}

type Triangle struct {
	// Base corresponds to the JSON schema field "base".
	Base float64 `json:"base" yaml:"base" mapstructure:"base"`

	// Height corresponds to the JSON schema field "height".
	Height float64 `json:"height" yaml:"height" mapstructure:"height"`

	// Type corresponds to the JSON schema field "type".
	Type string `json:"type" yaml:"type" mapstructure:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Triangle) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["base"]; raw != nil && !ok {
		return fmt.Errorf("field base in Triangle: required")
	}
	if _, ok := raw["height"]; raw != nil && !ok {
		return fmt.Errorf("field height in Triangle: required")
	}
	if _, ok := raw["type"]; raw != nil && !ok {
		return fmt.Errorf("field type in Triangle: required")
	}
	type Plain Triangle
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Triangle(plain)
	return nil
}
