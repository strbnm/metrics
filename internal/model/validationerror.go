package models

import (
	"fmt"
	"strings"
)

// FieldError — ошибка валидации конкретного поля.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e FieldError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// ValidationError — агрегатная ошибка, содержащая все ошибки валидации.
type ValidationError struct {
	Errors []FieldError
}

func (ve *ValidationError) Error() string {
	msgs := make([]string, len(ve.Errors))
	for i, e := range ve.Errors {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}
