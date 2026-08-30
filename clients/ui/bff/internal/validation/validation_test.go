package validation

import (
	"testing"

	"github.com/kubeflow/hub/pkg/openapi-v1"
)

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
