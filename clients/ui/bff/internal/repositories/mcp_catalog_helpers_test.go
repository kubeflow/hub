package repositories

import (
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestShouldClearMcpCatalogSourceStatusOnUpdate(t *testing.T) {
	content := "servers: []"
	path := "mcp_source.yaml"
	base := &models.McpCatalogSourceConfig{
		Type:            McpCatalogTypeYaml,
		Yaml:            &content,
		YamlCatalogPath: &path,
		Enabled:         boolPointer(true),
		Labels:          []string{"community"},
		IncludedServers: []string{"server-a", "server-b"},
		ExcludedServers: []string{"server-c"},
	}

	tests := []struct {
		name     string
		mutate   func(*models.McpCatalogSourceConfig)
		expected bool
	}{
		{
			name: "yaml content changed",
			mutate: func(config *models.McpCatalogSourceConfig) {
				changed := "servers:\n  - name: server-a"
				config.Yaml = &changed
			},
			expected: true,
		},
		{
			name: "yaml catalog path changed",
			mutate: func(config *models.McpCatalogSourceConfig) {
				changed := "new_path.yaml"
				config.YamlCatalogPath = &changed
			},
			expected: true,
		},
		{
			name: "enabled true to false",
			mutate: func(config *models.McpCatalogSourceConfig) {
				config.Enabled = boolPointer(false)
			},
			expected: false,
		},
		{
			name: "server filters changed",
			mutate: func(config *models.McpCatalogSourceConfig) {
				config.IncludedServers = []string{"server-a", "server-new"}
			},
			expected: false,
		},
		{
			name: "filter order only",
			mutate: func(config *models.McpCatalogSourceConfig) {
				config.IncludedServers = []string{"server-b", "server-a"}
			},
			expected: false,
		},
		{
			name: "labels only",
			mutate: func(config *models.McpCatalogSourceConfig) {
				config.Labels = []string{"updated"}
			},
			expected: false,
		},
		{
			name: "yaml unchanged",
			mutate: func(config *models.McpCatalogSourceConfig) {
				config.Name = "New Name"
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after := *base
			tt.mutate(&after)
			assert.Equal(t, tt.expected, ShouldClearMcpCatalogSourceStatusOnUpdate(base, &after))
		})
	}
}

func TestShouldClearMcpCatalogSourceStatusOnUpdate_EnabledFalseToTrue(t *testing.T) {
	content := "servers: []"
	path := "mcp_source.yaml"

	before := &models.McpCatalogSourceConfig{
		Type:            McpCatalogTypeYaml,
		Yaml:            &content,
		YamlCatalogPath: &path,
		Enabled:         boolPointer(false),
	}

	after := &models.McpCatalogSourceConfig{
		Type:            McpCatalogTypeYaml,
		Yaml:            &content,
		YamlCatalogPath: &path,
		Enabled:         boolPointer(true),
	}

	assert.True(t, ShouldClearMcpCatalogSourceStatusOnUpdate(before, after))
}

func TestShouldClearMcpCatalogSourceStatusOnUpdate_NilHandling(t *testing.T) {
	content := "servers: []"
	config := &models.McpCatalogSourceConfig{
		Type: McpCatalogTypeYaml,
		Yaml: &content,
	}

	assert.False(t, ShouldClearMcpCatalogSourceStatusOnUpdate(nil, config))
	assert.False(t, ShouldClearMcpCatalogSourceStatusOnUpdate(config, nil))
	assert.False(t, ShouldClearMcpCatalogSourceStatusOnUpdate(nil, nil))
}

func boolPointer(value bool) *bool {
	return &value
}
