// Package validation provides input validation utilities for API handlers.
package validation

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a collection of validation errors.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	msgs := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		msgs[i] = fmt.Sprintf("%s: %s", fe.Field, fe.Message)
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validator collects validation errors.
type Validator struct {
	errors []FieldError
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{
		errors: make([]FieldError, 0),
	}
}

// AddError adds a validation error for a field.
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, FieldError{Field: field, Message: message})
}

// HasErrors returns true if there are validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Error returns the validation error if there are errors, nil otherwise.
func (v *Validator) Error() error {
	if !v.HasErrors() {
		return nil
	}
	return &ValidationError{Errors: v.errors}
}

// Required validates that a string field is not empty.
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "is required")
	}
}

// RequiredInt validates that an int field is not zero.
func (v *Validator) RequiredInt(field string, value int) {
	if value == 0 {
		v.AddError(field, "is required")
	}
}

// RequiredInt64 validates that an int64 field is not zero.
func (v *Validator) RequiredInt64(field string, value int64) {
	if value == 0 {
		v.AddError(field, "is required")
	}
}

// MaxLength validates that a string doesn't exceed max length (in runes).
func (v *Validator) MaxLength(field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		v.AddError(field, fmt.Sprintf("must be at most %d characters", max))
	}
}

// MinLength validates that a string has at least min length (in runes).
func (v *Validator) MinLength(field, value string, min int) {
	if utf8.RuneCountInString(value) < min {
		v.AddError(field, fmt.Sprintf("must be at least %d characters", min))
	}
}

// Range validates that an int is within range (inclusive).
func (v *Validator) Range(field string, value, min, max int) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

// RangeFloat validates that a float64 is within range (inclusive).
func (v *Validator) RangeFloat(field string, value, min, max float64) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %.2f and %.2f", min, max))
	}
}

// Positive validates that an int is positive (> 0).
func (v *Validator) Positive(field string, value int) {
	if value <= 0 {
		v.AddError(field, "must be positive")
	}
}

// PositiveFloat validates that a float64 is positive (> 0).
func (v *Validator) PositiveFloat(field string, value float64) {
	if value <= 0 {
		v.AddError(field, "must be positive")
	}
}

// NonNegative validates that an int is non-negative (>= 0).
func (v *Validator) NonNegative(field string, value int) {
	if value < 0 {
		v.AddError(field, "must not be negative")
	}
}

// NonNegativeFloat validates that a float64 is non-negative (>= 0).
func (v *Validator) NonNegativeFloat(field string, value float64) {
	if value < 0 {
		v.AddError(field, "must not be negative")
	}
}

// Email validates that a string is a valid email address.
func (v *Validator) Email(field, value string) {
	if value == "" {
		return // Use Required for required fields
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		v.AddError(field, "must be a valid email address")
	}
}

// URL validates that a string looks like a URL (basic check).
func (v *Validator) URL(field, value string) {
	if value == "" {
		return // Use Required for required fields
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		v.AddError(field, "must be a valid URL starting with http:// or https://")
	}
}

// OneOf validates that a string is one of the allowed values.
func (v *Validator) OneOf(field, value string, allowed []string) {
	if value == "" {
		return // Use Required for required fields
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	v.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// NoDangerousChars validates that a string doesn't contain potentially dangerous characters.
// This is a basic sanitization check for fields that shouldn't contain special chars.
func (v *Validator) NoDangerousChars(field, value string) {
	dangerous := []string{"<", ">", "&", "'", "\"", "\x00"}
	for _, d := range dangerous {
		if strings.Contains(value, d) {
			v.AddError(field, "contains invalid characters")
			return
		}
	}
}

// NoControlChars validates that a string doesn't contain control characters (except common whitespace).
func (v *Validator) NoControlChars(field, value string) {
	for _, r := range value {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			v.AddError(field, "contains invalid control characters")
			return
		}
	}
}
