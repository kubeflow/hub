package models

import (
	"github.com/kubeflow/model-registry/pkg/openapi"
)

type ToolCallingConfig struct {
	Args *string `json:"args,omitempty"`
}

type ServingConfig struct {
	ToolCalling *ToolCallingConfig `json:"toolCalling,omitempty"`
}

type CatalogModel struct {
	CreateTimeSinceEpoch     *string                           `json:"createTimeSinceEpoch,omitempty"`
	CustomProperties         *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	Description              *string                           `json:"description,omitempty"`
	Language                 []string                          `json:"language,omitempty"`
	LastUpdateTimeSinceEpoch *string                           `json:"lastUpdateTimeSinceEpoch,omitempty"`
	LibraryName              *string                           `json:"libraryName,omitempty"`
	License                  *string                           `json:"license,omitempty"`
	LicenseLink              *string                           `json:"licenseLink,omitempty"`
	Logo                     *string                           `json:"logo,omitempty"`
	Maturity                 *string                           `json:"maturity,omitempty"`
	Name                     string                            `json:"name"`
	Provider                 *string                           `json:"provider,omitempty"`
	Readme                   *string                           `json:"readme,omitempty"`
	SourceId                 *string                           `json:"source_id,omitempty"`
	Tasks                    []string                          `json:"tasks,omitempty"`
	ValidatedTasks           []string                          `json:"validatedTasks,omitempty"`
	ServingConfig            *ServingConfig                    `json:"servingConfig,omitempty"`
}

type CatalogModelList struct {
	NextPageToken string         `json:"nextPageToken"`
	PageSize      int32          `json:"pageSize"`
	Size          int32          `json:"size"`
	Items         []CatalogModel `json:"items"`
}

type FilterRange struct {
	Max *float32 `json:"max,omitempty"`
	Min *float32 `json:"min,omitempty"`
}

type FilterOption struct {
	Range  *FilterRange     `json:"range,omitempty"`
	Type   FilterOptionType `json:"type"`
	Values []interface{}    `json:"values,omitempty"`
}

type FieldFilter struct {
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type FilterOptionType string

type FilterOptionsList struct {
	Filters      *map[string]FilterOption           `json:"filters,omitempty"`
	NamedQueries *map[string]map[string]FieldFilter `json:"namedQueries,omitempty"`
}
