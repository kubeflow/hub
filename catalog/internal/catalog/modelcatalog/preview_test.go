package modelcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// previewFilesFor builds the PreviewFiles the server would wire for a sources
// file living in dir.
func previewFilesFor(dir string) *basecatalog.PreviewFiles {
	return basecatalog.NewPreviewFiles(func() []string {
		return []string{filepath.Join(dir, "sources.yaml")}
	}, nil)
}

func TestParsePreviewConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, config *PreviewConfig)
	}{
		{
			name: "valid config with all fields",
			configYAML: `
type: yaml
includedModels:
  - "Granite/*"
  - "Llama/*"
excludedModels:
  - "*-draft"
  - "*-experimental"
properties:
  yamlCatalogPath: "/path/to/catalog.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *PreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Equal(t, []string{"Granite/*", "Llama/*"}, config.IncludedModels)
				assert.Equal(t, []string{"*-draft", "*-experimental"}, config.ExcludedModels)
				assert.Equal(t, "/path/to/catalog.yaml", config.Properties["yamlCatalogPath"])
			},
		},
		{
			name: "valid config with only type",
			configYAML: `
type: yaml
properties:
  yamlCatalogPath: "/path/to/catalog.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *PreviewConfig) {
				assert.Equal(t, "yaml", config.Type)
				assert.Empty(t, config.IncludedModels)
				assert.Empty(t, config.ExcludedModels)
			},
		},
		{
			name: "valid huggingface config",
			configYAML: `
type: hf
includedModels:
  - "microsoft/*"
properties:
  apiKey: "test-key"
  modelLimit: 100
`,
			expectError: false,
			validate: func(t *testing.T, config *PreviewConfig) {
				assert.Equal(t, "hf", config.Type)
				assert.Equal(t, []string{"microsoft/*"}, config.IncludedModels)
				assert.Equal(t, "test-key", config.Properties["apiKey"])
			},
		},
		{
			name:        "missing type field",
			configYAML:  `includedModels: ["Granite/*"]`,
			expectError: true,
			errorMsg:    "missing required field: type",
		},
		{
			name: "extra fields from full source config are ignored",
			configYAML: `
name: "Community and Custom Models"
id: community_custom_models
type: yaml
enabled: true
includedModels:
  - "Granite/*"
properties:
  yamlCatalogPath: "/path/to/catalog.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, config *PreviewConfig) {
				// Extra fields (name, id, enabled) should be ignored
				assert.Equal(t, "yaml", config.Type)
				assert.Equal(t, []string{"Granite/*"}, config.IncludedModels)
				assert.Equal(t, "/path/to/catalog.yaml", config.Properties["yamlCatalogPath"])
			},
		},
		{
			name: "empty type field",
			configYAML: `
type: ""
includedModels:
  - "Granite/*"
`,
			expectError: true,
			errorMsg:    "missing required field: type",
		},
		{
			name: "invalid YAML syntax",
			configYAML: `
type: yaml
includedModels: [
  - broken
`,
			expectError: true,
			errorMsg:    "failed to parse config",
		},
		{
			name: "empty pattern in includedModels",
			configYAML: `
type: yaml
includedModels:
  - "Granite/*"
  - ""
`,
			expectError: true,
			errorMsg:    "pattern cannot be empty",
		},
		{
			name: "whitespace-only pattern",
			configYAML: `
type: yaml
includedModels:
  - "   "
`,
			expectError: true,
			errorMsg:    "pattern cannot be empty",
		},
		{
			name: "conflicting pattern in both include and exclude",
			configYAML: `
type: yaml
includedModels:
  - "Granite/model-2"
excludedModels:
  - "Granite/model-2"
`,
			expectError: false,
		},
		{
			name: "conflicting pattern in both include and exclude, with wildcard",
			configYAML: `
type: yaml
includedModels:
  - "Granite/*"
excludedModels:
  - "Granite/*"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParsePreviewConfig([]byte(tt.configYAML))

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestPreviewSourceModels(t *testing.T) {
	// Create a temporary catalog file for testing
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test-catalog.yaml")

	catalogContent := `
source: Test Source
models:
  - name: Granite/3b-instruct
    description: Granite model
  - name: Granite/8b-code
    description: Another Granite model
  - name: Llama/7b-chat
    description: Llama model
  - name: Llama/13b-chat-draft
    description: Draft Llama model
  - name: Mistral/7b-instruct
    description: Mistral model
  - name: DeepSeek/coder-v2
    description: DeepSeek model
`
	err := os.WriteFile(catalogPath, []byte(catalogContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name          string
		config        *PreviewConfig
		expectError   bool
		errorMsg      string
		expectedTotal int
		expectedIncl  int
		expectedExcl  int
		validateNames func(t *testing.T, results []string)
	}{
		{
			name: "no filters - all models included",
			config: &PreviewConfig{
				Type:       "yaml",
				Properties: map[string]any{"yamlCatalogPath": catalogPath},
				Files:      previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  6,
			expectedExcl:  0,
		},
		{
			name: "include only Granite models",
			config: &PreviewConfig{
				Type:           "yaml",
				IncludedModels: []string{"Granite/*"},
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  2,
			expectedExcl:  4,
			validateNames: func(t *testing.T, included []string) {
				assert.Contains(t, included, "Granite/3b-instruct")
				assert.Contains(t, included, "Granite/8b-code")
			},
		},
		{
			name: "include Granite and Llama",
			config: &PreviewConfig{
				Type:           "yaml",
				IncludedModels: []string{"Granite/*", "Llama/*"},
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  4,
			expectedExcl:  2,
		},
		{
			name: "exclude draft models",
			config: &PreviewConfig{
				Type:           "yaml",
				ExcludedModels: []string{"*-draft"},
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  5,
			expectedExcl:  1,
			validateNames: func(t *testing.T, included []string) {
				assert.NotContains(t, included, "Llama/13b-chat-draft")
			},
		},
		{
			name: "include Llama but exclude drafts",
			config: &PreviewConfig{
				Type:           "yaml",
				IncludedModels: []string{"Llama/*"},
				ExcludedModels: []string{"*-draft"},
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  1, // Only Llama/7b-chat
			expectedExcl:  5,
			validateNames: func(t *testing.T, included []string) {
				assert.Equal(t, []string{"Llama/7b-chat"}, included)
			},
		},
		{
			name: "case insensitive matching",
			config: &PreviewConfig{
				Type:           "yaml",
				IncludedModels: []string{"granite/*"}, // lowercase
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  2, // Should match Granite/*
			expectedExcl:  4,
		},
		{
			name: "wildcard in middle of pattern",
			config: &PreviewConfig{
				Type:           "yaml",
				IncludedModels: []string{"*/7b-*"},
				Properties:     map[string]any{"yamlCatalogPath": catalogPath},
				Files:          previewFilesFor(tmpDir),
			},
			expectedTotal: 6,
			expectedIncl:  2, // Llama/7b-chat and Mistral/7b-instruct
			expectedExcl:  4,
		},
		{
			name: "unsupported source type",
			config: &PreviewConfig{
				Type:       "unknown",
				Properties: map[string]any{},
			},
			expectError: true,
			errorMsg:    "unsupported source type",
		},
		{
			name: "huggingface requires includedModels",
			config: &PreviewConfig{
				Type:       "hf",
				Properties: map[string]any{},
			},
			expectError: true,
			errorMsg:    "includedModels is required for HuggingFace source preview",
		},
		{
			name: "missing yamlCatalogPath property",
			config: &PreviewConfig{
				Type:       "yaml",
				Properties: map[string]any{},
			},
			expectError: true,
			errorMsg:    "missing required property",
		},
		{
			name: "catalog path outside the allowed directories",
			config: &PreviewConfig{
				Type:       "yaml",
				Properties: map[string]any{"yamlCatalogPath": "/nonexistent/path.yaml"},
				Files:      previewFilesFor(t.TempDir()),
			},
			expectError: true,
			errorMsg:    "failed to read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := PreviewSourceModels(ctx, tt.config, nil) // nil = path-based mode

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, results)

			// Count included and excluded
			var includedCount, excludedCount int
			var includedNames []string
			for _, r := range results {
				if r.Included {
					includedCount++
					includedNames = append(includedNames, r.Name)
				} else {
					excludedCount++
				}
			}

			assert.Equal(t, tt.expectedTotal, len(results), "total models mismatch")
			assert.Equal(t, tt.expectedIncl, includedCount, "included count mismatch")
			assert.Equal(t, tt.expectedExcl, excludedCount, "excluded count mismatch")

			if tt.validateNames != nil {
				tt.validateNames(t, includedNames)
			}
		})
	}
}

func TestLoadYamlModelNames(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		catalogContent string
		setupConfig    func(path string) *PreviewConfig
		expectError    bool
		errorMsg       string
		expectedNames  []string
	}{
		{
			name: "valid catalog with multiple models",
			catalogContent: `
source: Test
models:
  - name: Model/A
  - name: Model/B
  - name: Model/C
`,
			setupConfig: func(path string) *PreviewConfig {
				return &PreviewConfig{
					Type:       "yaml",
					Properties: map[string]any{"yamlCatalogPath": path},
				}
			},
			expectedNames: []string{"Model/A", "Model/B", "Model/C"},
		},
		{
			name: "empty models list",
			catalogContent: `
source: Empty
models: []
`,
			setupConfig: func(path string) *PreviewConfig {
				return &PreviewConfig{
					Type:       "yaml",
					Properties: map[string]any{"yamlCatalogPath": path},
				}
			},
			expectedNames: []string{},
		},
		{
			name: "catalog with artifacts",
			catalogContent: `
source: With Artifacts
models:
  - name: Model/WithArtifacts
    description: Has artifacts
    artifacts:
      - uri: oci://test/artifact:v1
        customProperties:
          hardware_type:
            metadataType: MetadataStringValue
            string_value: GPU
`,
			setupConfig: func(path string) *PreviewConfig {
				return &PreviewConfig{
					Type:       "yaml",
					Properties: map[string]any{"yamlCatalogPath": path},
				}
			},
			expectedNames: []string{"Model/WithArtifacts"},
		},
		{
			name:           "invalid YAML content",
			catalogContent: `not: valid: yaml: [`,
			setupConfig: func(path string) *PreviewConfig {
				return &PreviewConfig{
					Type:       "yaml",
					Properties: map[string]any{"yamlCatalogPath": path},
				}
			},
			expectError: true,
			errorMsg:    "failed to parse catalog file",
		},
		{
			name: "relative path resolves against the config directory",
			catalogContent: `source: Relative
models:
  - name: Relative/Model
`,
			setupConfig: func(path string) *PreviewConfig {
				// Use just the filename (relative path)
				return &PreviewConfig{
					Type:       "yaml",
					Properties: map[string]any{"yamlCatalogPath": filepath.Base(path)},
				}
			},
			expectedNames: []string{"Relative/Model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write catalog file
			catalogPath := filepath.Join(tmpDir, tt.name+".yaml")
			err := os.WriteFile(catalogPath, []byte(tt.catalogContent), 0644)
			require.NoError(t, err)

			config := tt.setupConfig(catalogPath)
			config.Files = previewFilesFor(tmpDir)
			ctx := context.Background()

			names, err := loadYamlModelNames(ctx, config, nil) // nil = path-based mode

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedNames, names)
		})
	}
}

func TestPreviewSourceModels_StatelessMode(t *testing.T) {
	// Test stateless mode where catalog data is passed directly
	catalogData := []byte(`
source: Stateless Test
models:
  - name: Granite/3b-instruct
    description: Granite model
  - name: Llama/7b-chat
    description: Llama model
  - name: Mistral/7b-draft
    description: Draft model
`)

	t.Run("stateless mode with catalog data", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "yaml",
			IncludedModels: []string{"Granite/*", "Llama/*"},
			ExcludedModels: []string{"*-draft"},
			// No yamlCatalogPath needed in stateless mode
		}

		results, err := PreviewSourceModels(context.Background(), config, catalogData)
		require.NoError(t, err)
		require.Len(t, results, 3)

		var included []string
		for _, r := range results {
			if r.Included {
				included = append(included, r.Name)
			}
		}

		assert.Len(t, included, 2)
		assert.Contains(t, included, "Granite/3b-instruct")
		assert.Contains(t, included, "Llama/7b-chat")
	})

	t.Run("stateless mode takes precedence over path", func(t *testing.T) {
		// Even with a yamlCatalogPath, catalog data should be used
		config := &PreviewConfig{
			Type:       "yaml",
			Properties: map[string]any{"yamlCatalogPath": "/nonexistent/path.yaml"},
		}

		results, err := PreviewSourceModels(context.Background(), config, catalogData)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("stateless mode with empty catalog data falls back to path", func(t *testing.T) {
		config := &PreviewConfig{
			Type:       "yaml",
			Properties: map[string]any{}, // No path either
		}

		_, err := PreviewSourceModels(context.Background(), config, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required property")
	})

	t.Run("stateless mode with invalid catalog data", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "yaml",
		}

		invalidData := []byte("not: valid: yaml: [")
		_, err := PreviewSourceModels(context.Background(), config, invalidData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse catalog file")
	})
}

// Note: HuggingFace preview tests that require API calls are in hf_preview_test.go
// with proper mock HTTP servers. The tests below only test error conditions that
// don't require API calls.

func TestPreviewSourceModels_HuggingFace_Errors(t *testing.T) {
	t.Run("hf preview with empty includedModels returns error", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "hf",
			IncludedModels: []string{},
		}

		_, err := PreviewSourceModels(context.Background(), config, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "includedModels is required")
	})

	t.Run("hf preview with invalid API key prefix returns error", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "hf",
			IncludedModels: []string{"some-org/some-model"},
			Properties:     map[string]any{"apiKey": "bad-prefix-key"},
		}

		_, err := PreviewSourceModels(context.Background(), config, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with 'hf_' prefix")
	})

	t.Run("hf preview without API key works for public models", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "hf",
			IncludedModels: []string{
				"openai-community/gpt2", // public model that doesn't require auth
			},
		}

		// Should not return an error - API key is optional
		result, err := PreviewSourceModels(context.Background(), config, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestPreviewSourceModels_FilterBehavior(t *testing.T) {
	// Create a temporary catalog file
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "filter-test.yaml")

	catalogContent := `
source: Filter Test
models:
  - name: ibm-granite/code-base-8b
  - name: ibm-granite/code-instruct-8b
  - name: ibm-granite/lab-model-experimental
  - name: meta-llama/Llama-2-7b
  - name: meta-llama/Llama-2-7b-draft
  - name: mistralai/Mistral-7B-Instruct-v0.1
`
	err := os.WriteFile(catalogPath, []byte(catalogContent), 0644)
	require.NoError(t, err)

	t.Run("exclusions take precedence over inclusions", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "yaml",
			IncludedModels: []string{"ibm-granite/*"},
			ExcludedModels: []string{"*-experimental"},
			Properties:     map[string]any{"yamlCatalogPath": catalogPath},
			Files:          previewFilesFor(tmpDir),
		}

		results, err := PreviewSourceModels(context.Background(), config, nil)
		require.NoError(t, err)

		// Should include ibm-granite models except experimental
		var included []string
		for _, r := range results {
			if r.Included {
				included = append(included, r.Name)
			}
		}

		assert.Len(t, included, 2)
		assert.Contains(t, included, "ibm-granite/code-base-8b")
		assert.Contains(t, included, "ibm-granite/code-instruct-8b")
		assert.NotContains(t, included, "ibm-granite/lab-model-experimental")
	})

	t.Run("multiple include patterns work as OR", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "yaml",
			IncludedModels: []string{"ibm-granite/*", "meta-llama/*"},
			Properties:     map[string]any{"yamlCatalogPath": catalogPath},
			Files:          previewFilesFor(tmpDir),
		}

		results, err := PreviewSourceModels(context.Background(), config, nil)
		require.NoError(t, err)

		var includedCount int
		for _, r := range results {
			if r.Included {
				includedCount++
			}
		}

		// 3 ibm-granite + 2 meta-llama = 5
		assert.Equal(t, 5, includedCount)
	})

	t.Run("multiple exclude patterns work as OR", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "yaml",
			ExcludedModels: []string{"*-experimental", "*-draft"},
			Properties:     map[string]any{"yamlCatalogPath": catalogPath},
			Files:          previewFilesFor(tmpDir),
		}

		results, err := PreviewSourceModels(context.Background(), config, nil)
		require.NoError(t, err)

		var excluded []string
		for _, r := range results {
			if !r.Included {
				excluded = append(excluded, r.Name)
			}
		}

		assert.Len(t, excluded, 2)
		assert.Contains(t, excluded, "ibm-granite/lab-model-experimental")
		assert.Contains(t, excluded, "meta-llama/Llama-2-7b-draft")
	})
}
