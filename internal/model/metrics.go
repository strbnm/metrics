package models

import (
	"encoding/json"
	"fmt"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// Кастомная логика десериализации с валидацией MType
func (m *Metrics) UnmarshalJSON(data []byte) error {
	// Аliasing, чтобы избежать рекурсии при вызове json.Unmarshal.
	type MetricAlias Metrics

	// Парсим во временный объект
	alias := new(MetricAlias)
	if err := json.Unmarshal(data, alias); err != nil {
		return fmt.Errorf("JSON parse error: %w", err)
	}

	var vErr ValidationError

	// Валидация ID.
	if alias.ID == "" {
		vErr.Errors = append(vErr.Errors, FieldError{
			Field:   "id",
			Message: "required",
		})
	}

	// Валидация MType.
	switch alias.MType {
	case Counter, Gauge:
		// ок
	default:
		vErr.Errors = append(vErr.Errors, FieldError{
			Field:   "type",
			Message: fmt.Sprintf("invalid value %q, must be %q or %q", alias.MType, Counter, Gauge),
		})
	}

	if len(vErr.Errors) > 0 {
		return &vErr
	}

	// Только после успешной валидации копируем данные в m.
	*m = Metrics(*alias)
	return nil
}
