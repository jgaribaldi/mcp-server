package echo

import (
	"fmt"
	"strings"
)

type EchoService struct{}

func NewEchoService() *EchoService {
	return &EchoService{}
}

func (s *EchoService) Transform(message, prefix, suffix string, uppercase bool) string {
	result := prefix + message + suffix
	if uppercase {
		result = strings.ToUpper(result)
	}
	return result
}

func (s *EchoService) Validate(message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	
	if len(message) > 1000 {
		return fmt.Errorf("message too long: %d characters (maximum 1000)", len(message))
	}
	
	return nil
}

func (s *EchoService) ValidatePrefix(prefix string) error {
	if len(prefix) > 100 {
		return fmt.Errorf("prefix too long: %d characters (maximum 100)", len(prefix))
	}
	return nil
}

func (s *EchoService) ValidateSuffix(suffix string) error {
	if len(suffix) > 100 {
		return fmt.Errorf("suffix too long: %d characters (maximum 100)", len(suffix))
	}
	return nil
}

func (s *EchoService) ValidateAll(message, prefix, suffix string) error {
	if err := s.Validate(message); err != nil {
		return err
	}
	
	if err := s.ValidatePrefix(prefix); err != nil {
		return err
	}
	
	if err := s.ValidateSuffix(suffix); err != nil {
		return err
	}
	
	return nil
}
