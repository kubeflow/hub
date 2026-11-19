package validation

import (
	"errors"

	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
)

func ValidateRegisteredModel(input openapiv1.RegisteredModel) error {
	if input.Name == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}

func ValidateModelVersion(input openapiv1.ModelVersion) error {
	if input.Name == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}

func ValidateModelArtifact(input openapiv1.ModelArtifact) error {
	if input.GetName() == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}
