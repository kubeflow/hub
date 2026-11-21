package models

import "github.com/kubeflow/hub/pkg/openapi"

// RegisteredModelCreate is the creation payload, separate from the response and PATCH types.
// Its fields match openapi.RegisteredModel so conversion preserves the existing wire format.
type RegisteredModelCreate struct {
	CustomProperties         map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	Description              *string                          `json:"description,omitempty"`
	ExternalId               *string                          `json:"externalId,omitempty"`
	Name                     string                           `json:"name" validate:"required" minLength:"1"`
	Id                       *string                          `json:"id,omitempty"`
	CreateTimeSinceEpoch     *string                          `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                          `json:"lastUpdateTimeSinceEpoch,omitempty"`
	Readme                   *string                          `json:"readme,omitempty"`
	Maturity                 *string                          `json:"maturity,omitempty"`
	Language                 []string                         `json:"language,omitempty"`
	Tasks                    []string                         `json:"tasks,omitempty"`
	Provider                 *string                          `json:"provider,omitempty"`
	Logo                     *string                          `json:"logo,omitempty"`
	License                  *string                          `json:"license,omitempty"`
	LicenseLink              *string                          `json:"licenseLink,omitempty"`
	LibraryName              *string                          `json:"libraryName,omitempty"`
	Owner                    *string                          `json:"owner,omitempty"`
	State                    *openapi.RegisteredModelState    `json:"state,omitempty"`
}

// ModelVersionCreateForRegisteredModel gets its parent from the route, not the body.
// The optional body parent is retained for compatibility with existing callers.
type ModelVersionCreateForRegisteredModel struct {
	CustomProperties         map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	Description              *string                          `json:"description,omitempty"`
	ExternalId               *string                          `json:"externalId,omitempty"`
	Name                     string                           `json:"name" validate:"required" minLength:"1"`
	State                    *openapi.ModelVersionState       `json:"state,omitempty"`
	Author                   *string                          `json:"author,omitempty"`
	RegisteredModelId        string                           `json:"registeredModelId"`
	Id                       *string                          `json:"id,omitempty"`
	CreateTimeSinceEpoch     *string                          `json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                          `json:"lastUpdateTimeSinceEpoch,omitempty"`
}

// ModelVersionCreate requires the parent in the body when it is absent from the route.
type ModelVersionCreate struct {
	ModelVersionCreateForRegisteredModel
	RegisteredModelId string `json:"registeredModelId" validate:"required" minLength:"1"`
}

func (request ModelVersionCreate) ModelVersion() openapi.ModelVersion {
	version := openapi.ModelVersion(request.ModelVersionCreateForRegisteredModel)
	version.RegisteredModelId = request.RegisteredModelId
	return version
}
