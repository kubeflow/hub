package validation

import (
	"testing"

	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
)

func TestValidateRegisteredModel(t *testing.T) {
	specs := []testSpec[openapiv1.RegisteredModel]{
		{
			name:    "Empty name",
			input:   openapiv1.RegisteredModel{Name: ""},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapiv1.RegisteredModel{Name: "ValidName"},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateRegisteredModel)
}

func TestValidateModelVersion(t *testing.T) {
	specs := []testSpec[openapiv1.ModelVersion]{
		{
			name:    "Empty name",
			input:   openapiv1.ModelVersion{Name: ""},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapiv1.ModelVersion{Name: "ValidName"},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateModelVersion)
}

func TestValidateModel(t *testing.T) {
	specs := []testSpec[openapiv1.ModelArtifact]{
		{
			name:    "Empty name",
			input:   openapiv1.ModelArtifact{Name: openapiv1.PtrString("")},
			wantErr: true,
		},
		{
			name:    "Valid name",
			input:   openapiv1.ModelArtifact{Name: openapiv1.PtrString("ValidName")},
			wantErr: false,
		},
	}

	validateTestSpecs(t, specs, ValidateModelArtifact)
}
