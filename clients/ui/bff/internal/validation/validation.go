package validation

import (
	"errors"

	"github.com/kubeflow/hub/pkg/openapi-v1"
)

func ValidateRegisteredModel(input openapi.RegisteredModel) error {
	if input.Name == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}

func ValidateModelVersion(input openapi.ModelVersion) error {
	if input.Name == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}

func ValidateModelArtifact(input openapi.ModelArtifact) error {
	if input.GetName() == "" {
		return errors.New("name cannot be empty")
	}
	// Add more field validations as required
	return nil
}
