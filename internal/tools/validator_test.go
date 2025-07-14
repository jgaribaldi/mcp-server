package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

func createTestValidator() *ToolValidator {
	cfg := &config.Config{}
	log, _ := logger.NewDefault()
	return NewToolValidator(cfg, log)
}

func TestToolValidator_ValidateName(t *testing.T) {
	validator := createTestValidator()

	tests := []struct {
		name      string
		toolName  string
		wantError bool
	}{
		{"valid simple name", "test_tool", false},
		{"valid name with numbers", "tool123", false},
		{"valid name starting with letter", "a_tool_name", false},
		{"empty name", "", true},
		{"name with hyphen", "test-tool", true},
		{"name with space", "test tool", true},
		{"name starting with number", "123tool", true},
		{"name with special chars", "tool@name", true},
		{"reserved name", "system", true},
		{"reserved name case insensitive", "INTERNAL", true},
		{"too long name", strings.Repeat("a", 65), true},
		{"max length name", strings.Repeat("a", 64), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateName(tt.toolName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestToolValidator_ValidateFactory(t *testing.T) {
	validator := createTestValidator()

	// Valid factory
	validFactory := &mockToolFactory{
		name:         "test_tool",
		description:  "A test tool",
		version:      "1.0.0",
		capabilities: []string{"read", "write"},
		requirements: map[string]string{"runtime": "go"},
	}

	err := validator.ValidateFactory(validFactory)
	if err != nil {
		t.Errorf("Expected no error for valid factory, got: %v", err)
	}

	// Factory with invalid name
	invalidNameFactory := &mockToolFactory{
		name:        "invalid-name",
		description: "A test tool",
		version:     "1.0.0",
	}

	err = validator.ValidateFactory(invalidNameFactory)
	if err == nil {
		t.Error("Expected error for invalid name, got nil")
	}

	// Factory with empty description
	emptyDescFactory := &mockToolFactory{
		name:        "test_tool",
		description: "",
		version:     "1.0.0",
	}

	err = validator.ValidateFactory(emptyDescFactory)
	if err == nil {
		t.Error("Expected error for empty description, got nil")
	}

	// Factory with long description
	longDescFactory := &mockToolFactory{
		name:        "test_tool",
		description: strings.Repeat("a", 501),
		version:     "1.0.0",
	}

	err = validator.ValidateFactory(longDescFactory)
	if err == nil {
		t.Error("Expected error for long description, got nil")
	}

	// Factory with empty version
	emptyVersionFactory := &mockToolFactory{
		name:        "test_tool",
		description: "A test tool",
		version:     "",
	}

	err = validator.ValidateFactory(emptyVersionFactory)
	if err == nil {
		t.Error("Expected error for empty version, got nil")
	}

	// Factory with too many capabilities
	tooManyCapsFactory := &mockToolFactory{
		name:         "test_tool",
		description:  "A test tool",
		version:      "1.0.0",
		capabilities: make([]string, 21), // More than 20
	}
	for i := range tooManyCapsFactory.capabilities {
		tooManyCapsFactory.capabilities[i] = "cap"
	}

	err = validator.ValidateFactory(tooManyCapsFactory)
	if err == nil {
		t.Error("Expected error for too many capabilities, got nil")
	}

	// Factory with empty capability
	emptyCapsFactory := &mockToolFactory{
		name:         "test_tool",
		description:  "A test tool",
		version:      "1.0.0",
		capabilities: []string{"valid", ""},
	}

	err = validator.ValidateFactory(emptyCapsFactory)
	if err == nil {
		t.Error("Expected error for empty capability, got nil")
	}

	// Factory with too many requirements
	tooManyReqsFactory := &mockToolFactory{
		name:         "test_tool",
		description:  "A test tool",
		version:      "1.0.0",
		requirements: make(map[string]string),
	}
	for i := 0; i < 21; i++ {
		tooManyReqsFactory.requirements[fmt.Sprintf("req%d", i)] = "value"
	}

	err = validator.ValidateFactory(tooManyReqsFactory)
	if err == nil {
		t.Error("Expected error for too many requirements, got nil")
	}
}

func TestToolValidator_ValidateTool(t *testing.T) {
	validator := createTestValidator()

	// Valid tool
	validParams, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	})

	validTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  validParams,
		handler:     &mockToolHandler{},
	}

	err := validator.ValidateTool(validTool)
	if err != nil {
		t.Errorf("Expected no error for valid tool, got: %v", err)
	}

	// Tool with invalid name
	invalidNameTool := &mockTool{
		name:        "invalid-name!",
		description: "A test tool",
		parameters:  validParams,
		handler:     &mockToolHandler{},
	}

	err = validator.ValidateTool(invalidNameTool)
	if err == nil {
		t.Error("Expected error for invalid name, got nil")
	}

	// Tool with empty description
	emptyDescTool := &mockTool{
		name:        "test_tool",
		description: "",
		parameters:  validParams,
		handler:     &mockToolHandler{},
	}

	err = validator.ValidateTool(emptyDescTool)
	if err == nil {
		t.Error("Expected error for empty description, got nil")
	}

	// Tool with invalid JSON parameters
	invalidJSONTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  json.RawMessage(`{invalid json`),
		handler:     &mockToolHandler{},
	}

	err = validator.ValidateTool(invalidJSONTool)
	if err == nil {
		t.Error("Expected error for invalid JSON parameters, got nil")
	}

	// Tool with nil handler
	nilHandlerTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		parameters:  validParams,
		handler:     nil,
	}

	err = validator.ValidateTool(nilHandlerTool)
	if err == nil {
		t.Error("Expected error for nil handler, got nil")
	}
}

func TestToolValidator_ValidateJSONSchema(t *testing.T) {
	validator := createTestValidator()

	tests := []struct {
		name      string
		schema    string
		wantError bool
	}{
		{
			"valid object schema",
			`{"type": "object", "properties": {"name": {"type": "string"}}}`,
			false,
		},
		{
			"valid array schema",
			`{"type": "array", "items": {"type": "string"}}`,
			false,
		},
		{
			"valid string schema",
			`{"type": "string"}`,
			false,
		},
		{
			"empty schema",
			`{}`,
			false,
		},
		{
			"invalid JSON",
			`{invalid json`,
			true,
		},
		{
			"invalid type",
			`{"type": "invalid_type"}`,
			true,
		},
		{
			"non-string type",
			`{"type": 123}`,
			true,
		},
		{
			"invalid properties",
			`{"type": "object", "properties": "not_an_object"}`,
			true,
		},
		{
			"invalid required",
			`{"type": "object", "required": "not_an_array"}`,
			true,
		},
		{
			"non-string in required",
			`{"type": "object", "required": [123]}`,
			true,
		},
		{
			"invalid items",
			`{"type": "array", "items": "not_an_object"}`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateJSONSchema(json.RawMessage(tt.schema))
			if (err != nil) != tt.wantError {
				t.Errorf("validateJSONSchema() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestToolValidator_ValidateToolConfig(t *testing.T) {
	validator := createTestValidator()

	// Valid config
	validConfig := ToolConfig{
		Enabled:    true,
		Config:     map[string]interface{}{"key": "value"},
		Timeout:    30,
		MaxRetries: 3,
	}

	err := validator.ValidateToolConfig(validConfig)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Negative timeout
	negativeTimeoutConfig := ToolConfig{
		Enabled:    true,
		Timeout:    -1,
		MaxRetries: 3,
	}

	err = validator.ValidateToolConfig(negativeTimeoutConfig)
	if err == nil {
		t.Error("Expected error for negative timeout, got nil")
	}

	// Too high timeout
	highTimeoutConfig := ToolConfig{
		Enabled:    true,
		Timeout:    3601,
		MaxRetries: 3,
	}

	err = validator.ValidateToolConfig(highTimeoutConfig)
	if err == nil {
		t.Error("Expected error for too high timeout, got nil")
	}

	// Negative max retries
	negativeRetriesConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: -1,
	}

	err = validator.ValidateToolConfig(negativeRetriesConfig)
	if err == nil {
		t.Error("Expected error for negative max retries, got nil")
	}

	// Too high max retries
	highRetriesConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: 11,
	}

	err = validator.ValidateToolConfig(highRetriesConfig)
	if err == nil {
		t.Error("Expected error for too high max retries, got nil")
	}

	// Too many config items
	tooManyConfigConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: 3,
		Config:     make(map[string]interface{}),
	}
	for i := 0; i < 51; i++ {
		tooManyConfigConfig.Config[fmt.Sprintf("key%d", i)] = "value"
	}

	err = validator.ValidateToolConfig(tooManyConfigConfig)
	if err == nil {
		t.Error("Expected error for too many config items, got nil")
	}

	// Empty config key
	emptyKeyConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: 3,
		Config:     map[string]interface{}{"": "value"},
	}

	err = validator.ValidateToolConfig(emptyKeyConfig)
	if err == nil {
		t.Error("Expected error for empty config key, got nil")
	}

	// Too long config key
	longKeyConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: 3,
		Config:     map[string]interface{}{strings.Repeat("a", 65): "value"},
	}

	err = validator.ValidateToolConfig(longKeyConfig)
	if err == nil {
		t.Error("Expected error for too long config key, got nil")
	}

	// Too long config value
	longValueConfig := ToolConfig{
		Enabled:    true,
		Timeout:    30,
		MaxRetries: 3,
		Config:     map[string]interface{}{"key": strings.Repeat("a", 1025)},
	}

	err = validator.ValidateToolConfig(longValueConfig)
	if err == nil {
		t.Error("Expected error for too long config value, got nil")
	}
}

func TestToolValidationErrors(t *testing.T) {
	var errors ToolValidationErrors

	// Test empty errors
	if errors.HasErrors() {
		t.Error("Expected no errors for empty slice")
	}

	if errors.Error() != "" {
		t.Error("Expected empty string for no errors")
	}

	// Add single error
	errors.Add("field1", "value1", "message1")

	if !errors.HasErrors() {
		t.Error("Expected errors after adding one")
	}

	if errors.Error() != "validation error in field 'field1' (value: 'value1'): message1" {
		t.Errorf("Unexpected single error message: %s", errors.Error())
	}

	// Add second error
	errors.Add("field2", "value2", "message2")

	errorMsg := errors.Error()
	if !strings.Contains(errorMsg, "2 validation errors") {
		t.Errorf("Expected multiple errors message, got: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "and 1 more") {
		t.Errorf("Expected 'and 1 more' in message, got: %s", errorMsg)
	}
}

func TestToolValidationError(t *testing.T) {
	err := ToolValidationError{
		Field:   "test_field",
		Value:   "test_value",
		Message: "test message",
	}

	expected := "validation error in field 'test_field' (value: 'test_value'): test message"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}