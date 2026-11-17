package models

import "encoding/json"

// CatalogSourceCreateFields shares source fields without imposing creation rules on PATCH.
// Its fields match CatalogSourceConfigPayload so conversion retains all accepted fields.
type CatalogSourceCreateFields struct {
	Id                  string   `json:"id" validate:"required" minLength:"1" example:"community_models"`
	Name                string   `json:"name" validate:"required" minLength:"1"`
	Type                string   `json:"type"`
	Enabled             *bool    `json:"enabled,omitempty"`
	Labels              []string `json:"labels"`
	ApiKey              *string  `json:"apiKey,omitempty"`
	HasConfiguredApiKey *bool    `json:"hasConfiguredApiKey,omitempty"`
	AllowedOrganization *string  `json:"allowedOrganization,omitempty"`
	IncludedModels      []string `json:"includedModels,omitempty"`
	ExcludedModels      []string `json:"excludedModels,omitempty"`
	IsDefault           *bool    `json:"isDefault,omitempty"`
	Yaml                *string  `json:"yaml,omitempty"`
	YamlCatalogPath     *string  `json:"yamlCatalogPath,omitempty"`
}

// YamlCatalogSourceCreate requires inline catalog content.
type YamlCatalogSourceCreate struct {
	CatalogSourceCreateFields
	Type string  `json:"type" validate:"required" enums:"yaml"`
	Yaml *string `json:"yaml" validate:"required" minLength:"1"`
}

// HFCatalogSourceCreate requires a Hugging Face organization.
type HFCatalogSourceCreate struct {
	CatalogSourceCreateFields
	Type                string  `json:"type" validate:"required" enums:"hf"`
	AllowedOrganization *string `json:"allowedOrganization" validate:"required" minLength:"1"`
}

// DecodeCatalogSourceCreate selects the same provider-specific types documented by Swag.
// Existing repository validation remains responsible for unsupported provider types.
func DecodeCatalogSourceCreate(data json.RawMessage) (CatalogSourceConfigPayload, error) {
	var provider struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &provider); err != nil {
		return CatalogSourceConfigPayload{}, err
	}
	switch provider.Type {
	case "yaml":
		var request YamlCatalogSourceCreate
		if err := json.Unmarshal(data, &request); err != nil {
			return CatalogSourceConfigPayload{}, err
		}
		payload := CatalogSourceConfigPayload(request.CatalogSourceCreateFields)
		payload.Type, payload.Yaml = request.Type, request.Yaml
		return payload, nil
	case "hf":
		var request HFCatalogSourceCreate
		if err := json.Unmarshal(data, &request); err != nil {
			return CatalogSourceConfigPayload{}, err
		}
		payload := CatalogSourceConfigPayload(request.CatalogSourceCreateFields)
		payload.Type, payload.AllowedOrganization = request.Type, request.AllowedOrganization
		return payload, nil
	default:
		var payload CatalogSourceConfigPayload
		err := json.Unmarshal(data, &payload)
		return payload, err
	}
}

// McpCatalogSourceCreate has creation requirements that do not apply to PATCH.
type McpCatalogSourceCreate struct {
	Id              string   `json:"id" validate:"required" minLength:"1" example:"community_mcp"`
	Name            string   `json:"name" validate:"required" minLength:"1"`
	Type            string   `json:"type" validate:"required" enums:"yaml"`
	Enabled         *bool    `json:"enabled,omitempty"`
	Labels          []string `json:"labels"`
	IncludedServers []string `json:"includedServers,omitempty"`
	ExcludedServers []string `json:"excludedServers,omitempty"`
	IsDefault       *bool    `json:"isDefault,omitempty"`
	Yaml            *string  `json:"yaml" validate:"required" minLength:"1"`
	YamlCatalogPath *string  `json:"yamlCatalogPath,omitempty"`
}
