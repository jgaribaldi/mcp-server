package tools

import (
	"encoding/json"
	"regexp"

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
)

func (v *ToolValidator) ValidateName(name string) error {
	var errors ToolValidationErrors

	v.ValidateRequiredString(name, "name", &errors)
	if errors.HasErrors() {
		return errors
	}

	v.ValidateStringLength(name, "name", 64, &errors)
	v.ValidateStringPattern(name, "name", toolNameRegex, "must start with a letter and contain only alphanumeric characters and underscores", &errors)

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
	v.ValidateCapabilities(factory.GetCapabilities(), &errors)

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
	var errors ToolValidationErrors

	if len(input) == 0 {
		errors.Add("input", "", "input cannot be empty")
		return errors
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		errors.Add("input", string(input), "invalid JSON format")
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}