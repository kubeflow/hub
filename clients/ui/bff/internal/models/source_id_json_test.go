package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The v1 catalog APIs use camelCase "sourceId"; guard against regressions to "source_id".
func TestSourceIDJSONRoundTrip(t *testing.T) {
	sourceID := "sample-source"

	tests := []struct {
		name  string
		value any
		out   func() (any, func() *string)
	}{
		{
			name:  "CatalogModel",
			value: CatalogModel{Name: "model", SourceID: &sourceID},
			out: func() (any, func() *string) {
				var v CatalogModel
				return &v, func() *string { return v.SourceID }
			},
		},
		{
			name:  "Agent",
			value: Agent{ID: "1", Name: "agent", SourceID: &sourceID},
			out: func() (any, func() *string) {
				var v Agent
				return &v, func() *string { return v.SourceID }
			},
		},
		{
			name:  "McpServer",
			value: McpServer{ID: "1", Name: "mcp", SourceID: &sourceID},
			out: func() (any, func() *string) {
				var v McpServer
				return &v, func() *string { return v.SourceID }
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)

			var raw map[string]any
			require.NoError(t, json.Unmarshal(data, &raw))
			assert.Equal(t, sourceID, raw["sourceId"])
			assert.NotContains(t, raw, "source_id")

			target, get := tt.out()
			require.NoError(t, json.Unmarshal(data, target))
			require.NotNil(t, get())
			assert.Equal(t, sourceID, *get())
		})
	}
}
