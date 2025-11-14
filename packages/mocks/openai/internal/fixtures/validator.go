// Package fixtures provides response fixture management.
// This file implements validation for fixture files and responses.
package fixtures

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator validates fixture files and individual fixtures.
type Validator struct {
	// strictMode enables strict validation rules
	strictMode bool

	// allowedRoles defines valid message roles
	allowedRoles map[string]bool

	// allowedFinishReasons defines valid finish reasons
	allowedFinishReasons map[string]bool
}

// NewValidator creates a new fixture validator.
func NewValidator(strictMode bool) *Validator {
	return &Validator{
		strictMode: strictMode,
		allowedRoles: map[string]bool{
			"assistant": true,
			"user":      true,
			"system":    true,
			"function":  true,
		},
		allowedFinishReasons: map[string]bool{
			"stop":           true,
			"length":         true,
			"content_filter": true,
			"function_call":  true,
			"tool_calls":     true,
		},
	}
}

// ValidateFile validates an entire fixture file.
func (v *Validator) ValidateFile(file *FixtureFile) []ValidationError {
	var errors []ValidationError

	// Validate description
	if file.Description == "" && v.strictMode {
		errors = append(errors, ValidationError{
			Field:   "description",
			Message: "description is empty",
			Severity: "warning",
		})
	}

	// Validate category
	if file.Category == "" && v.strictMode {
		errors = append(errors, ValidationError{
			Field:   "category",
			Message: "category is empty",
			Severity: "warning",
		})
	}

	// Validate responses
	if len(file.Responses) == 0 {
		errors = append(errors, ValidationError{
			Field:   "responses",
			Message: "no responses defined",
			Severity: "error",
		})
		return errors
	}

	// Validate each fixture
	for i, fixture := range file.Responses {
		fixtureErrors := v.ValidateFixture(&fixture)
		for _, err := range fixtureErrors {
			err.Field = fmt.Sprintf("responses[%d].%s", i, err.Field)
			errors = append(errors, err)
		}
	}

	// Check for duplicate IDs
	idMap := make(map[string]int)
	for i, fixture := range file.Responses {
		if prevIdx, exists := idMap[fixture.ID]; exists {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("responses[%d].id", i),
				Message: fmt.Sprintf("duplicate ID '%s' (also at index %d)", fixture.ID, prevIdx),
				Severity: "error",
			})
		}
		idMap[fixture.ID] = i
	}

	return errors
}

// ValidateFixture validates a single fixture.
func (v *Validator) ValidateFixture(fixture *Fixture) []ValidationError {
	var errors []ValidationError

	// Validate ID
	if fixture.ID == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "id is required",
			Severity: "error",
		})
	} else if !v.isValidID(fixture.ID) {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "id contains invalid characters (use only a-z, 0-9, -, _)",
			Severity: "error",
		})
	}

	// Validate content or function_call
	if fixture.Content == "" && fixture.FunctionCall == nil {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "either content or function_call must be provided",
			Severity: "error",
		})
	}

	// Validate role
	if fixture.Role != "" && !v.allowedRoles[fixture.Role] {
		errors = append(errors, ValidationError{
			Field:   "role",
			Message: fmt.Sprintf("invalid role '%s' (allowed: assistant, user, system, function)", fixture.Role),
			Severity: "error",
		})
	}

	// Validate finish_reason
	if fixture.FinishReason != "" && !v.allowedFinishReasons[fixture.FinishReason] {
		errors = append(errors, ValidationError{
			Field:   "finish_reason",
			Message: fmt.Sprintf("invalid finish_reason '%s'", fixture.FinishReason),
			Severity: "error",
		})
	}

	// Validate weight
	if fixture.Weight < 0 {
		errors = append(errors, ValidationError{
			Field:   "weight",
			Message: "weight cannot be negative",
			Severity: "error",
		})
	}

	// Validate pattern (if present)
	if fixture.Pattern != "" {
		if _, err := regexp.Compile(fixture.Pattern); err != nil {
			errors = append(errors, ValidationError{
				Field:   "pattern",
				Message: fmt.Sprintf("invalid regex pattern: %s", err.Error()),
				Severity: "error",
			})
		}
	}

	// Validate function_call (if present)
	if fixture.FunctionCall != nil {
		if fixture.FunctionCall.Name == "" {
			errors = append(errors, ValidationError{
				Field:   "function_call.name",
				Message: "function name is required",
				Severity: "error",
			})
		}

		if fixture.FunctionCall.Arguments == "" {
			errors = append(errors, ValidationError{
				Field:   "function_call.arguments",
				Message: "function arguments are required (use {} for empty)",
				Severity: "warning",
			})
		}
	}

	// Strict mode validations
	if v.strictMode {
		// Check content length
		if len(fixture.Content) == 0 && fixture.FunctionCall == nil {
			errors = append(errors, ValidationError{
				Field:   "content",
				Message: "content is empty",
				Severity: "warning",
			})
		}

		// Check for very long content
		if len(fixture.Content) > 10000 {
			errors = append(errors, ValidationError{
				Field:   "content",
				Message: fmt.Sprintf("content is very long (%d characters)", len(fixture.Content)),
				Severity: "warning",
			})
		}

		// Check for placeholder content
		if v.containsPlaceholder(fixture.Content) {
			errors = append(errors, ValidationError{
				Field:   "content",
				Message: "content contains placeholder text (TODO, FIXME, etc.)",
				Severity: "warning",
			})
		}
	}

	return errors
}

// ValidatePattern validates a pattern configuration.
func (v *Validator) ValidatePattern(config PatternConfig) []ValidationError {
	var errors []ValidationError

	// Validate name
	if config.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required",
			Severity: "error",
		})
	}

	// Validate regex
	if config.Regex == "" {
		errors = append(errors, ValidationError{
			Field:   "regex",
			Message: "regex is required",
			Severity: "error",
		})
	} else {
		pattern := config.Regex
		if config.CaseInsensitive {
			pattern = "(?i)" + pattern
		}

		if _, err := regexp.Compile(pattern); err != nil {
			errors = append(errors, ValidationError{
				Field:   "regex",
				Message: fmt.Sprintf("invalid regex: %s", err.Error()),
				Severity: "error",
			})
		}
	}

	// Validate fixture path
	if config.Fixture == "" {
		errors = append(errors, ValidationError{
			Field:   "fixture",
			Message: "fixture path is required",
			Severity: "error",
		})
	}

	return errors
}

// isValidID checks if an ID contains only valid characters.
func (v *Validator) isValidID(id string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", id)
	return match
}

// containsPlaceholder checks if content contains placeholder text.
func (v *Validator) containsPlaceholder(content string) bool {
	placeholders := []string{
		"TODO",
		"FIXME",
		"XXX",
		"PLACEHOLDER",
		"<insert",
		"[insert",
	}

	contentUpper := strings.ToUpper(content)
	for _, placeholder := range placeholders {
		if strings.Contains(contentUpper, placeholder) {
			return true
		}
	}

	return false
}

// ValidationError represents a validation error.
type ValidationError struct {
	// Field is the field that failed validation
	Field string

	// Message is the error message
	Message string

	// Severity is the error severity (error, warning, info)
	Severity string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Severity, e.Field, e.Message)
}

// IsError returns true if this is an error (not warning).
func (e ValidationError) IsError() bool {
	return e.Severity == "error"
}

// IsWarning returns true if this is a warning.
func (e ValidationError) IsWarning() bool {
	return e.Severity == "warning"
}

// ValidationResult contains validation results.
type ValidationResult struct {
	// Valid indicates if validation passed
	Valid bool

	// Errors contains validation errors
	Errors []ValidationError

	// Warnings contains validation warnings
	Warnings []ValidationError
}

// HasErrors returns true if there are any errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ErrorCount returns the number of errors.
func (r *ValidationResult) ErrorCount() int {
	return len(r.Errors)
}

// WarningCount returns the number of warnings.
func (r *ValidationResult) WarningCount() int {
	return len(r.Warnings)
}

// FormatErrors formats all errors as a string.
func (r *ValidationResult) FormatErrors() string {
	var builder strings.Builder

	for _, err := range r.Errors {
		builder.WriteString(err.Error())
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatWarnings formats all warnings as a string.
func (r *ValidationResult) FormatWarnings() string {
	var builder strings.Builder

	for _, warn := range r.Warnings {
		builder.WriteString(warn.Error())
		builder.WriteString("\n")
	}

	return builder.String()
}

// Validate performs full validation and returns a result.
func (v *Validator) Validate(file *FixtureFile) *ValidationResult {
	allErrors := v.ValidateFile(file)

	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationError, 0),
	}

	for _, err := range allErrors {
		if err.IsError() {
			result.Errors = append(result.Errors, err)
			result.Valid = false
		} else if err.IsWarning() {
			result.Warnings = append(result.Warnings, err)
		}
	}

	return result
}