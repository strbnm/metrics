package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, m Metrics, err error)
	}{
		{
			name:    "valid counter",
			input:   `{"id":"c1","type":"counter","delta":10}`,
			wantErr: false,
			check: func(t *testing.T, m Metrics, err error) {
				require.NoError(t, err)
				assert.Equal(t, "c1", m.ID)
				assert.Equal(t, Counter, m.MType)
				require.NotNil(t, m.Delta)
				assert.EqualValues(t, int64(10), *m.Delta)
				assert.Nil(t, m.Value)
			},
		},
		{
			name:    "valid gauge",
			input:   `{"id":"g1","type":"gauge","value":3.14}`,
			wantErr: false,
			check: func(t *testing.T, m Metrics, err error) {
				require.NoError(t, err)
				assert.Equal(t, "g1", m.ID)
				assert.Equal(t, Gauge, m.MType)
				require.NotNil(t, m.Value)
				assert.InDelta(t, 3.14, *m.Value, 0.0001)
				assert.Nil(t, m.Delta)
			},
		},
		{
			name:    "counter with delta zero",
			input:   `{"id":"c0","type":"counter","delta":0}`,
			wantErr: false,
			check: func(t *testing.T, m Metrics, err error) {
				require.NoError(t, err)
				require.NotNil(t, m.Delta)
				assert.EqualValues(t, int64(0), *m.Delta)
			},
		},
		{
			name:    "gauge with value zero",
			input:   `{"id":"g0","type":"gauge","value":0}`,
			wantErr: false,
			check: func(t *testing.T, m Metrics, err error) {
				require.NoError(t, err)
				require.NotNil(t, m.Value)
				assert.InDelta(t, 0.0, *m.Value, 0.0001)
			},
		},
		{
			name:    "invalid MType",
			input:   `{"id":"x1","type":"unknown","delta":5}`,
			wantErr: true,
			check: func(t *testing.T, m Metrics, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `field "type": invalid value "unknown"`)
				// Структура не должна быть частично заполнена
				assert.Empty(t, m.MType)
				assert.Empty(t, m.ID)
			},
		},
		{
			name:    "empty MType",
			input:   `{"id":"e1","type":""}`,
			wantErr: true,
			check: func(t *testing.T, m Metrics, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `field "type": invalid value ""`)
			},
		},
		{
			name:    "empty ID",
			input:   `{"type":"counter","delta":10}`,
			wantErr: true,
			check: func(t *testing.T, m Metrics, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `field "id": required`)
				assert.NotContains(t, err.Error(), `field "type":`)
			},
		},
		{
			name:    "empty JSON object",
			input:   `{}`,
			wantErr: true,
			check: func(t *testing.T, m Metrics, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `field "type": invalid value ""`)
				assert.Contains(t, err.Error(), `field "id": required`)
			},
		},
		{
			name:    "malformed JSON",
			input:   `{invalid}`,
			wantErr: true,
			check: func(t *testing.T, m Metrics, err error) {
				require.Error(t, err)
				var syntaxErr *json.SyntaxError
				assert.ErrorAs(t, err, &syntaxErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Metrics
			err := json.Unmarshal([]byte(tt.input), &m)
			tt.check(t, m, err)
		})
	}
}
