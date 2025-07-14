package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// ToolValidator validates tool implementations
type ToolValidator struct {
	logger *logger.Logger
	config *config.Config
}

// NewToolValidator creates a new tool validator
func NewToolValidator(cfg *config.Config, log *logger.Logger) *ToolValidator {
	return &ToolValidator{
		logger: log,
		config: cfg,
	}
}

var (
	// Tool name must be alphanumeric with underscores, 1-64 characters
	toolNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)
)

// ValidateName validates a tool name
func (v *ToolValidator) ValidateName(name string) error {
	var errors ToolValidationErrors

	// Check if name is empty
	if name == "" {
		errors.Add("name", name, "tool name cannot be empty")
		return errors
	}

	// Check length
	if len(name) > 64 {
		errors.Add("name", name, "tool name cannot exceed 64 characters")
	}

	// Check format
	if !toolNameRegex.MatchString(name) {
		errors.Add("name", name, "tool name must start with a letter and contain only alphanumeric characters and underscores")
	}

	// Check for reserved names
	reservedNames := []string{
		"system", "internal", "mcp", "server", "registry", "health", "status", "admin",
	}
	lowerName := strings.ToLower(name)
	for _, reserved := range reservedNames {
		if lowerName == reserved {
			errors.Add("name", name, fmt.Sprintf("tool name '%s' is reserved", reserved))
			break
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ValidateFactory validates a tool factory
func (v *ToolValidator) ValidateFactory(factory ToolFactory) error {
	var errors ToolValidationErrors

	// Validate factory name
	if err := v.ValidateName(factory.Name()); err != nil {
		if valErrs, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, valErrs...)
		} else {
			errors.Add("factory.name", factory.Name(), err.Error())
		}
	}

	// Validate description
	if factory.Description() == "" {
		errors.Add("factory.description", "", "tool description cannot be empty")
	} else if len(factory.Description()) > 500 {
		errors.Add("factory.description", factory.Description(), "tool description cannot exceed 500 characters")
	}

	// Validate version
	if factory.Version() == "" {
		errors.Add("factory.version", "", "tool version cannot be empty")
	} else if len(factory.Version()) > 32 {
		errors.Add("factory.version", factory.Version(), "tool version cannot exceed 32 characters")
	}

	// Validate capabilities
	capabilities := factory.Capabilities()
	if len(capabilities) > 20 {
		errors.Add("factory.capabilities", fmt.Sprintf("%d items", len(capabilities)), "cannot have more than 20 capabilities")
	}

	for i, capability := range capabilities {
		if capability == "" {
			errors.Add("factory.capabilities", fmt.Sprintf("index %d", i), "capability cannot be empty")
		} else if len(capability) > 64 {
			errors.Add("factory.capabilities", capability, "capability cannot exceed 64 characters")
		}
	}

	// Validate requirements
	requirements := factory.Requirements()
	if len(requirements) > 20 {
		errors.Add("factory.requirements", fmt.Sprintf("%d items", len(requirements)), "cannot have more than 20 requirements")
	}

	for key, value := range requirements {
		if key == "" {
			errors.Add("factory.requirements", "empty key", "requirement key cannot be empty")
		} else if len(key) > 64 {
			errors.Add("factory.requirements", key, "requirement key cannot exceed 64 characters")
		}

		if len(value) > 256 {
			errors.Add("factory.requirements", value, "requirement value cannot exceed 256 characters")
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// ValidateTool validates a tool implementation
func (v *ToolValidator) ValidateTool(tool mcp.Tool) error {
	var errors ToolValidationErrors

	// Validate tool name
	if err := v.ValidateName(tool.Name()); err != nil {
		if valErrs, ok := err.(ToolValidationErrors); ok {
			errors = append(errors, valErrs...)
		} else {
			errors.Add("tool.name", tool.Name(), err.Error())
		}
	}

	// Validate description
	if tool.Description() == "" {
		errors.Add("tool.description", "", "tool description cannot be empty")
	} else if len(tool.Description()) > 500 {
		errors.Add("tool.description", tool.Description(), "tool description cannot exceed 500 characters")
	}

	// Validate parameters (JSON schema)
	if tool.Parameters() != nil {
		if err := v.validateJSONSchema(tool.Parameters()); err != nil {
			errors.Add("tool.parameters", string(tool.Parameters()), fmt.Sprintf("invalid JSON schema: %v", err))
		}
	}

	// Validate handler exists
	if tool.Handler() == nil {
		errors.Add("tool.handler", "nil", "tool handler cannot be nil")
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// validateJSONSchema validates that parameters conform to JSON schema format
func (v *ToolValidator) validateJSONSchema(params json.RawMessage) error {
	if len(params) == 0 {
		return nil // Empty parameters are valid
	}

	// Parse as generic JSON first
	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Basic JSON Schema validation
	if err := v.validateSchemaStructure(schema, ""); err != nil {
		return err
	}

	return nil
}

// validateSchemaStructure performs basic JSON schema structure validation
func (v *ToolValidator) validateSchemaStructure(schema map[string]interface{}, path string) error {
	// Check for valid schema type
	if schemaType, exists := schema["type"]; exists {
		typeStr, ok := schemaType.(string)
		if !ok {
			return fmt.Errorf("type at %s must be a string", path)
		}

		validTypes := []string{"object", "array", "string", "number", "integer", "boolean", "null"}
		isValid := false
		for _, validType := range validTypes {
			if typeStr == validType {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid type '%s' at %s", typeStr, path)
		}
	}

	// Validate properties if it's an object type
	if properties, exists := schema["properties"]; exists {
		propsMap, ok := properties.(map[string]interface{})
		if !ok {
			return fmt.Errorf("properties at %s must be an object", path)
		}

		for propName, propSchema := range propsMap {
			propSchemaMap, ok := propSchema.(map[string]interface{})
			if !ok {
				return fmt.Errorf("property '%s' at %s must be an object", propName, path)
			}

			newPath := path + ".properties." + propName
			if err := v.validateSchemaStructure(propSchemaMap, newPath); err != nil {
				return err
			}
		}
	}

	// Validate required array
	if required, exists := schema["required"]; exists {
		requiredArray, ok := required.([]interface{})
		if !ok {
			return fmt.Errorf("required at %s must be an array", path)
		}

		for i, req := range requiredArray {
			if _, ok := req.(string); !ok {
				return fmt.Errorf("required[%d] at %s must be a string", i, path)
			}
		}
	}

	// Validate items for array type
	if items, exists := schema["items"]; exists {
		itemsMap, ok := items.(map[string]interface{})
		if !ok {
			return fmt.Errorf("items at %s must be an object", path)
		}

		newPath := path + ".items"
		if err := v.validateSchemaStructure(itemsMap, newPath); err != nil {
			return err
		}
	}

	return nil
}

// ValidateToolConfig validates tool configuration
func (v *ToolValidator) ValidateToolConfig(config ToolConfig) error {
	var errors ToolValidationErrors

	// Validate timeout
	if config.Timeout < 0 {
		errors.Add("config.timeout", fmt.Sprintf("%d", config.Timeout), "timeout cannot be negative")
	} else if config.Timeout > 3600 {
		errors.Add("config.timeout", fmt.Sprintf("%d", config.Timeout), "timeout cannot exceed 3600 seconds")
	}

	// Validate max retries
	if config.MaxRetries < 0 {
		errors.Add("config.max_retries", fmt.Sprintf("%d", config.MaxRetries), "max retries cannot be negative")
	} else if config.MaxRetries > 10 {
		errors.Add("config.max_retries", fmt.Sprintf("%d", config.MaxRetries), "max retries cannot exceed 10")
	}

	// Validate config map size
	if len(config.Config) > 50 {
		errors.Add("config.config", fmt.Sprintf("%d items", len(config.Config)), "configuration cannot have more than 50 items")
	}

	// Validate config keys
	for key, value := range config.Config {
		if key == "" {
			errors.Add("config.config", "empty key", "configuration key cannot be empty")
		} else if len(key) > 64 {
			errors.Add("config.config", key, "configuration key cannot exceed 64 characters")
		}

		// Validate value (basic check)
		if value == nil {
			continue // nil values are allowed
		}

		// Check for overly complex values
		if valueStr := fmt.Sprintf("%v", value); len(valueStr) > 1024 {
			errors.Add("config.config", key, "configuration value cannot exceed 1024 characters when stringified")
		}
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}