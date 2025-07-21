package registry

import (
	"fmt"
	"regexp"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s' (value: '%s'): %s", e.Field, e.Value, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s (and %d more)", len(e), e[0].Error(), len(e)-1)
}

// Add appends a validation error
func (e *ValidationErrors) Add(field, value, message string) {
	*e = append(*e, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// BaseValidator provides common validation functionality
type BaseValidator struct {
	logger *logger.Logger
	config *config.Config
}

// NewBaseValidator creates a new base validator
func NewBaseValidator(cfg *config.Config, log *logger.Logger) *BaseValidator {
	return &BaseValidator{
		logger: log,
		config: cfg,
	}
}

// ValidateRequiredString validates that a string field is not empty
func (v *BaseValidator) ValidateRequiredString(value, fieldName string, errors *ValidationErrors) {
	if value == "" {
		errors.Add(fieldName, value, fmt.Sprintf("%s cannot be empty", fieldName))
	}
}

// ValidateStringLength validates string length constraints
func (v *BaseValidator) ValidateStringLength(value, fieldName string, maxLength int, errors *ValidationErrors) {
	if len(value) > maxLength {
		errors.Add(fieldName, value, fmt.Sprintf("%s too long: %d characters (max: %d)", fieldName, len(value), maxLength))
	}
}

// ValidateStringPattern validates that a string matches a pattern
func (v *BaseValidator) ValidateStringPattern(value, fieldName string, pattern *regexp.Regexp, patternDesc string, errors *ValidationErrors) {
	if !pattern.MatchString(value) {
		errors.Add(fieldName, value, fmt.Sprintf("%s %s", fieldName, patternDesc))
	}
}

// ValidateCapabilities validates that capabilities are provided and not empty
func (v *BaseValidator) ValidateCapabilities(capabilities []string, errors *ValidationErrors) {
	if len(capabilities) == 0 {
		errors.Add("capabilities", fmt.Sprintf("%v", capabilities), "at least one capability must be specified")
	}
	
	for i, capability := range capabilities {
		if capability == "" {
			errors.Add("capabilities", fmt.Sprintf("index %d", i), "capability cannot be empty")
		}
	}
}

// ValidateVersion validates version string format (basic semantic versioning check)
func (v *BaseValidator) ValidateVersion(version string, errors *ValidationErrors) {
	if version == "" {
		errors.Add("version", version, "version cannot be empty")
		return
	}
	
	// Basic semantic version pattern (X.Y.Z with optional suffixes)
	versionRegex := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([a-zA-Z0-9\-\.]*)?$`)
	if !versionRegex.MatchString(version) {
		errors.Add("version", version, "version must follow semantic versioning (e.g., 1.0.0)")
	}
}

// LogValidationResult logs the result of validation
func (v *BaseValidator) LogValidationResult(success bool, entityType, identifier string, errorCount int) {
	if success {
		v.logger.Debug(fmt.Sprintf("%s validation passed", entityType), "identifier", identifier)
	} else {
		v.logger.Error(fmt.Sprintf("%s validation failed", entityType), "identifier", identifier, "errors", errorCount)
	}
}