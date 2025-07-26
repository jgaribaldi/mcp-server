package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/registry"
)

type ToolValidator struct {
	*registry.BaseValidator
}

func NewToolValidator(cfg *config.Config, log *logger.Logger) *ToolValidator {
	return &ToolValidator{
		BaseValidator: registry.NewBaseValidator(cfg, log),
	}
}

var (
	toolNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)
	reservedNames = map[string]bool{
		"system":   true,
		"internal": true,
		"admin":    true,
		"root":     true,
		"api":      true,
	}
)

func (v *ToolValidator) ValidateName(name string) error {
	var errors ToolValidationErrors

	v.ValidateRequiredString(name, "name", &errors)
	if errors.HasErrors() {
		return errors
	}

	v.ValidateStringLength(name, "name", 64, &errors)
	v.ValidateStringPattern(name, "name", toolNameRegex, "must start with a letter and contain only alphanumeric characters and underscores", &errors)
	
	// Check for reserved names (case-insensitive)
	if reservedNames[strings.ToLower(name)] {
		errors.Add("name", name, "name is reserved and cannot be used")
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

func (v *ToolValidator) ValidateDescription(description string) error {
	var errors ToolValidationErrors

	v.ValidateRequiredString(description, "description", &errors)
	v.ValidateStringLength(description, "description", 500, &errors)

	if errors.HasErrors() {
		return errors
	}

	return nil
}

func (v *ToolValidator) ValidateFactory(factory ToolFactory) error {
	var errors ToolValidationErrors

	if err := v.ValidateName(factory.GetName()); err != nil {
		if validationErrors, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, validationErrors...)
		} else {
			errors.Add("name", factory.GetName(), err.Error())
		}
	}

	if err := v.ValidateDescription(factory.GetDescription()); err != nil {
		if validationErrors, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, validationErrors...)
		} else {
			errors.Add("description", factory.GetDescription(), err.Error())
		}
	}

	v.ValidateVersion(factory.GetVersion(), &errors)
	
	// Validate capability count limit
	capabilities := factory.GetCapabilities()
	if len(capabilities) > 20 {
		errors.Add("capabilities", fmt.Sprintf("%d capabilities", len(capabilities)), 
			"too many capabilities (max: 20)")
	}
	
	v.ValidateCapabilities(capabilities, &errors)

	config := ToolConfig{
		Enabled: true,
		Config:  make(map[string]interface{}),
	}
	if err := factory.Validate(config); err != nil {
		errors.Add("config", "", err.Error())
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

func (v *ToolValidator) ValidateTool(tool mcp.Tool) error {
	var errors ToolValidationErrors

	if err := v.ValidateName(tool.Name()); err != nil {
		if validationErrors, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, validationErrors...)
		} else {
			errors.Add("name", tool.Name(), err.Error())
		}
	}

	if err := v.ValidateDescription(tool.Description()); err != nil {
		if validationErrors, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, validationErrors...)
		} else {
			errors.Add("description", tool.Description(), err.Error())
		}
	}

	if tool.Handler() == nil {
		errors.Add("handler", "", "tool handler cannot be nil")
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

func (v *ToolValidator) validateJSONSchema(schema map[string]interface{}) error {
	if len(schema) == 0 {
		return nil
	}
	
	if err := v.validateSchemaType(schema); err != nil {
		return err
	}
	if err := v.validateSchemaProperties(schema); err != nil {
		return err
	}
	if err := v.validateSchemaRequired(schema); err != nil {
		return err
	}
	if err := v.validateSchemaItems(schema); err != nil {
		return err
	}
	
	return nil
}

func (v *ToolValidator) validateSchemaType(schema map[string]interface{}) error {
	schemaType, exists := schema["type"]
	if !exists {
		return nil
	}
	
	typeStr, ok := schemaType.(string)
	if !ok {
		return fmt.Errorf("schema type must be a string")
	}
	
	validTypes := map[string]bool{
		"object":  true,
		"array":   true,
		"string":  true,
		"number":  true,
		"boolean": true,
	}
	if !validTypes[typeStr] {
		return fmt.Errorf("invalid schema type: %s", typeStr)
	}
	
	return nil
}

func (v *ToolValidator) validateSchemaProperties(schema map[string]interface{}) error {
	properties, exists := schema["properties"]
	if !exists {
		return nil
	}
	
	if _, ok := properties.(map[string]interface{}); !ok {
		return fmt.Errorf("properties must be an object")
	}
	
	return nil
}

func (v *ToolValidator) validateSchemaRequired(schema map[string]interface{}) error {
	required, exists := schema["required"]
	if !exists {
		return nil
	}
	
	reqSlice, ok := required.([]interface{})
	if !ok {
		return fmt.Errorf("required must be an array")
	}
	
	for _, req := range reqSlice {
		if _, ok := req.(string); !ok {
			return fmt.Errorf("required array must contain only strings")
		}
	}
	
	return nil
}

func (v *ToolValidator) validateSchemaItems(schema map[string]interface{}) error {
	items, exists := schema["items"]
	if !exists {
		return nil
	}
	
	if _, ok := items.(map[string]interface{}); !ok {
		return fmt.Errorf("items must be an object")
	}
	
	return nil
}

func (v *ToolValidator) ValidateConfig(config ToolConfig) error {
	var errors ToolValidationErrors

	if config.Timeout < 0 {
		errors.Add("timeout", string(rune(config.Timeout)), "timeout cannot be negative")
	}

	if config.Timeout > 3600 {
		errors.Add("timeout", string(rune(config.Timeout)), "timeout cannot exceed 3600 seconds")
	}

	if config.MaxRetries < 0 {
		errors.Add("max_retries", string(rune(config.MaxRetries)), "max retries cannot be negative")
	}

	if config.MaxRetries > 10 {
		errors.Add("max_retries", string(rune(config.MaxRetries)), "max retries cannot exceed 10")
	}

	if config.Config != nil {
		for key, value := range config.Config {
			if key == "" {
				errors.Add("config", key, "configuration key cannot be empty")
			}
			if value == nil {
				errors.Add("config", key, "configuration value cannot be nil")
			}
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

func (v *ToolValidator) ValidateJSONInput(input []byte) error {
	var schema map[string]interface{}
	
	// First validate that it's valid JSON
	if err := json.Unmarshal(input, &schema); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	
	return v.validateJSONSchema(schema)
}