package storage

import (
	"testing"

	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/stretchr/testify/assert"
)

func TestParseModelVersion(t *testing.T) {
	cfg := openapi.NewConfiguration()
	cfg.Host = "localhost:8080"
	client := openapi.NewAPIClient(cfg)

	provider := &ModelRegistryProvider{
		Client: client,
	}

	tests := []struct {
		name            string
		storageUri      string
		expectedModel   string
		expectedVersion *string
		expectError     bool
	}{
		{
			name:            "basic model",
			storageUri:      "model-registry://iris",
			expectedModel:   "iris",
			expectedVersion: nil,
			expectError:     false,
		},
		{
			name:            "model and version",
			storageUri:      "model-registry://iris/v1",
			expectedModel:   "iris",
			expectedVersion: stringPtr("v1"),
			expectError:     false,
		},
		{
			name:            "embedded host with model",
			storageUri:      "model-registry://localhost:8080/iris",
			expectedModel:   "iris",
			expectedVersion: nil,
			expectError:     false,
		},
		{
			name:            "embedded host with model and version",
			storageUri:      "model-registry://localhost:8080/iris/v1",
			expectedModel:   "iris",
			expectedVersion: stringPtr("v1"),
			expectError:     false,
		},
		{
			name:            "namespace query param model",
			storageUri:      "model-registry://iris?namespace=profile-alpha",
			expectedModel:   "iris",
			expectedVersion: nil,
			expectError:     false,
		},
		{
			name:            "namespace query param model and version",
			storageUri:      "model-registry://iris/v1?namespace=profile-alpha",
			expectedModel:   "iris",
			expectedVersion: stringPtr("v1"),
			expectError:     false,
		},
		{
			name:            "namespace query param with embedded host",
			storageUri:      "model-registry://localhost:8080/iris/v1?namespace=profile-alpha",
			expectedModel:   "iris",
			expectedVersion: stringPtr("v1"),
			expectError:     false,
		},
		{
			name:        "missing model name",
			storageUri:  "model-registry://",
			expectError: true,
		},
		{
			name:        "empty model name after slash",
			storageUri:  "model-registry:///iris",
			expectError: true,
		},
		{
			name:        "embedded host without model name",
			storageUri:  "model-registry://localhost:8080/",
			expectError: true,
		},
		{
			name:        "empty version name",
			storageUri:  "model-registry://iris/",
			expectError: true,
		},
		{
			name:        "embedded host with empty version name",
			storageUri:  "model-registry://localhost:8080/iris/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, version, err := provider.parseModelVersion(tt.storageUri)
			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidMRURI)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedModel, model)
				if tt.expectedVersion == nil {
					assert.Nil(t, version)
				} else {
					assert.NotNil(t, version)
					assert.Equal(t, *tt.expectedVersion, *version)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
