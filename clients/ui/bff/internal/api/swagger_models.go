package api

import "github.com/kubeflow/hub/pkg/openapi"

// SwaggerMetadataValue documents the flattened JSON of openapi.MetadataValue.
// Exactly one value field is present, selected by metadataType. Swag cannot
// infer custom MarshalJSON implementations, so variant fields are explicit.
type SwaggerMetadataValue struct {
	MetadataType string   `json:"metadataType" validate:"required" enums:"MetadataBoolValue,MetadataDoubleValue,MetadataIntValue,MetadataProtoValue,MetadataStringValue,MetadataStructValue"`
	BoolValue    *bool    `json:"bool_value,omitempty"`
	DoubleValue  *float64 `json:"double_value,omitempty"`
	IntValue     *string  `json:"int_value,omitempty"`
	ProtoValue   *string  `json:"proto_value,omitempty"`
	StringValue  *string  `json:"string_value,omitempty"`
	StructValue  *string  `json:"struct_value,omitempty"`
}

// SwaggerArtifact documents the flattened JSON of openapi.Artifact.
// Fields depend on artifactType. DocArtifact's fields are a subset of
// ModelArtifact's. Metric and Parameter use different types for value.
// These documentation-only types do not change runtime serialization.
type SwaggerArtifact struct {
	openapi.ModelArtifact
	Digest        *string                `json:"digest,omitempty"`
	SourceType    *string                `json:"sourceType,omitempty"`
	Source        *string                `json:"source,omitempty"`
	Schema        *string                `json:"schema,omitempty"`
	Profile       *string                `json:"profile,omitempty"`
	Value         interface{}            `json:"value,omitempty" oneOf:"string,float64"`
	Timestamp     *string                `json:"timestamp,omitempty"`
	Step          *int64                 `json:"step,omitempty"`
	ParameterType *openapi.ParameterType `json:"parameterType,omitempty"`
}

// SwaggerLegacyModelRegistryPatch documents the registry-level alias separately
// because Swag v2 copies every path parameter to every @Router on a handler.
// This documentation-only function is never registered or called.
//
//	@Summary		Update a ModelVersion through the registry-level alias
//	@Description	This route invokes the model version updater without a model_version_id. Use the model version route instead.
//	@Tags			ModelRegistryService
//	@ID				updateModelVersionLegacy
//	@Accept			json
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace			query		string						true	"Model registry namespace"
//	@Param			model_registry_id	path		string						true	"Model registry name"
//	@Param			body				body		ModelVersionUpdateEnvelope	true	"Model version fields to update"
//	@Success		200					{object}	ModelVersionEnvelope
//	@Failure		400					{object}	ErrorEnvelope
//	@Failure		403					{object}	ErrorEnvelope
//	@Failure		404					{object}	ErrorEnvelope
//	@Failure		409					{object}	ErrorEnvelope
//	@Failure		500					{object}	ErrorEnvelope
//	@Router			/api/v1/model_registry/{model_registry_id} [patch]
func SwaggerLegacyModelRegistryPatch() {}
