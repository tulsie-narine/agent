// Package validation provides schema validation utilities for the Windows Inventory Agent API
package validation

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator provides JSON schema validation functionality
type Validator struct {
	schemas map[string]*jsonschema.Schema
}

// NewValidator creates a new schema validator instance
func NewValidator() *Validator {
	return &Validator{
		schemas: make(map[string]*jsonschema.Schema),
	}
}

// LoadSchema loads and compiles a JSON schema from a file path
func (v *Validator) LoadSchema(name, schemaPath string) error {
	schema, err := jsonschema.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}
	v.schemas[name] = schema
	return nil
}

// LoadSchemaFromBytes loads and compiles a JSON schema from byte data
func (v *Validator) LoadSchemaFromBytes(name string, schemaData []byte) error {
	schema, err := jsonschema.Compile(string(schemaData))
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}
	v.schemas[name] = schema
	return nil
}

// Validate validates data against a named schema
func (v *Validator) Validate(name string, data interface{}) error {
	schema, exists := v.schemas[name]
	if !exists {
		return fmt.Errorf("schema %s not found", name)
	}

	// Convert data to JSON for validation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for validation: %w", err)
	}

	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal data for validation: %w", err)
	}

	return schema.Validate(jsonObj)
}

// ValidateWithResult validates data and returns detailed validation results
func (v *Validator) ValidateWithResult(name string, data interface{}) (*ValidationResult, error) {
	schema, exists := v.schemas[name]
	if !exists {
		return nil, fmt.Errorf("schema %s not found", name)
	}

	// Convert data to JSON for validation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data for validation: %w", err)
	}

	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for validation: %w", err)
	}

	err = schema.Validate(jsonObj)
	result := &ValidationResult{
		Valid: err == nil,
	}

	if err != nil {
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			result.Errors = convertValidationErrors(validationErr)
		} else {
			result.Errors = []ValidationError{
				{
					Field:   "root",
					Message: err.Error(),
					Code:    "VALIDATION_ERROR",
				},
			}
		}
	}

	return result, nil
}

// ValidationResult contains the result of a validation operation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// convertValidationErrors converts jsonschema validation errors to our format
func convertValidationErrors(err *jsonschema.ValidationError) []ValidationError {
	var errors []ValidationError

	// Add the main error
	errors = append(errors, ValidationError{
		Field:   err.InstanceLocation,
		Message: err.Message,
		Code:    err.KeywordLocation,
	})

	// Add nested errors recursively
	for _, childErr := range err.Causes {
		childErrors := convertValidationErrors(childErr)
		errors = append(errors, childErrors...)
	}

	return errors
}

// ValidateTelemetry validates telemetry data against the telemetry schema
func (v *Validator) ValidateTelemetry(data interface{}) error {
	return v.Validate("telemetry", data)
}

// ValidatePolicy validates policy data against the policy schema
func (v *Validator) ValidatePolicy(data interface{}) error {
	return v.Validate("policy", data)
}

// ValidateCommand validates command data against the command schema
func (v *Validator) ValidateCommand(data interface{}) error {
	return v.Validate("command", data)
}

// ValidateTelemetryWithResult validates telemetry data and returns detailed results
func (v *Validator) ValidateTelemetryWithResult(data interface{}) (*ValidationResult, error) {
	return v.ValidateWithResult("telemetry", data)
}

// ValidatePolicyWithResult validates policy data and returns detailed results
func (v *Validator) ValidatePolicyWithResult(data interface{}) (*ValidationResult, error) {
	return v.ValidateWithResult("policy", data)
}

// ValidateCommandWithResult validates command data and returns detailed results
func (v *Validator) ValidateCommandWithResult(data interface{}) (*ValidationResult, error) {
	return v.ValidateWithResult("command", data)
}