package validation

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelVersionCreatePayload(t *testing.T) {
	raw := []byte(`{"name":"v1","registeredModelId":"42","author":"user","description":"version","customProperties":{"framework":{"metadataType":"MetadataStringValue","string_value":"pytorch"}}}`)
	var request models.ModelVersionCreate
	require.NoError(t, json.Unmarshal(raw, &request))
	var expected openapi.ModelVersion
	require.NoError(t, json.Unmarshal(raw, &expected))
	assert.Equal(t, expected, request.ModelVersion())
	encoded, err := json.Marshal(request.ModelVersion())
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(encoded))
}

func TestValidateRegisteredModel(t *testing.T) {
	specs := []testSpec[openapi.RegisteredModel]{
		{
			name:    "Empty name",
			input:   openapi.RegisteredModel{Name: ""},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapi.RegisteredModel{Name: "ValidName"},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateRegisteredModel)
}

func TestValidateModelVersion(t *testing.T) {
	specs := []testSpec[openapi.ModelVersion]{
		{
			name:    "Empty name",
			input:   openapi.ModelVersion{Name: ""},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapi.ModelVersion{Name: "ValidName"},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateModelVersion)
}

func TestValidateModel(t *testing.T) {
	specs := []testSpec[openapi.ModelArtifact]{
		{
			name:    "Empty name",
			input:   openapi.ModelArtifact{Name: openapi.PtrString("")},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapi.ModelArtifact{Name: openapi.PtrString("ValidName")},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateModelArtifact)
}
