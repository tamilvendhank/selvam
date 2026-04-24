package config

import (
	"fmt"
	"strings"
)

var (
	ErrUnsupportedConfigFormat  = fmt.Errorf("unsupported config format")
	ErrUnsupportedSchemaVersion = fmt.Errorf("unsupported config schema version")
)

type ValidationError struct {
	Field   string
	Message string
}

func (err ValidationError) Error() string {
	if strings.TrimSpace(err.Field) == "" {
		return err.Message
	}
	return fmt.Sprintf("%s: %s", err.Field, err.Message)
}

type ValidationErrors []ValidationError

func (errs *ValidationErrors) Add(field, message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	*errs = append(*errs, ValidationError{
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	})
}

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}

	parts := make([]string, 0, len(errs))
	for _, item := range errs {
		parts = append(parts, item.Error())
	}
	return strings.Join(parts, "; ")
}

func (errs ValidationErrors) OrNil() error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}
