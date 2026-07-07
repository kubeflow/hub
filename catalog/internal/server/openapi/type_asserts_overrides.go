package openapi

import (
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// AssertCatalogArtifactRequired checks if the required fields are not zero-ed
func AssertCatalogArtifactRequired(obj model.CatalogArtifact) error {
	// CatalogArtifact has no required fields but the openapi code gen
	// checks the fields from CatalogModelArtifact, which doesn't compile.
	return nil
}

// AssertCatalogArtifactConstraints checks if the values respects the defined constraints
func AssertCatalogArtifactConstraints(obj model.CatalogArtifact) error {
	return nil
}

// AssertAgentArtifactRequired checks if the required fields are not zero-ed
func AssertAgentArtifactRequired(obj model.AgentArtifact) error {
	// AgentArtifact is a oneOf union — field validation is on the concrete types.
	return nil
}

// AssertAgentArtifactConstraints checks if the values respects the defined constraints
func AssertAgentArtifactConstraints(obj model.AgentArtifact) error {
	return nil
}

// AssertPreviewCatalogSourceResponseRequired checks if the required fields are not zero-ed
func AssertPreviewCatalogSourceResponseRequired(obj model.PreviewCatalogSourceResponse) error {
	return nil
}

// AssertPreviewCatalogSourceResponseConstraints checks if the values respects the defined constraints
func AssertPreviewCatalogSourceResponseConstraints(obj model.PreviewCatalogSourceResponse) error {
	return nil
}

// AssertFilterOptionRequired checks if the required fields are not zero-ed
func AssertFilterOptionRequired(obj model.FilterOption) error {
	elements := map[string]any{
		"type": obj.Type,
	}
	for name, el := range elements {
		if isZero := IsZeroValue(el); isZero {
			return &RequiredError{Field: name}
		}
	}

	if obj.Range != nil {
		if err := AssertFilterOptionRangeRequired(*obj.Range); err != nil {
			return err
		}
	}
	return nil
}
