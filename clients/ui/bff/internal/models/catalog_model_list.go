package models

import (
	"github.com/kubeflow/hub/pkg/openapi"
)

// ToolCallingConfig configures tool calling for a serving runtime.
// @Description Tool-calling configuration for the model's serving runtime.
type ToolCallingConfig struct {
	// Tool-call parser identifier used by the serving runtime.
	ToolCallParser *string `json:"toolCallParser,omitempty" example:"granite"`
	// Path to the chat template packaged with the model.
	ChatTemplate *string `json:"chatTemplate,omitempty" example:"/templates/tool_chat.jinja"`
	// Whether the serving runtime should select tools automatically.
	EnableAutoToolChoice *bool `json:"enableAutoToolChoice,omitempty" example:"true"`
	// Additional serving-runtime command-line arguments required for tool calling.
	RequiredArgs []string `json:"requiredArgs,omitempty" example:"--enable-auto-tool-choice"`
}

// ServingConfig contains validated serving settings.
// @Description Validated serving configuration for a catalog model.
type ServingConfig struct {
	// Settings for serving the model with tool-calling support.
	ToolCalling *ToolCallingConfig `json:"toolCalling,omitempty"`
}

// CatalogModel is a model discovered from a catalog source.
// @Description Catalog model metadata, documentation, licensing, and validated serving configuration.
type CatalogModel struct {
	// Creation time in milliseconds since the Unix epoch, encoded as a string.
	CreateTimeSinceEpoch *string `json:"createTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Custom properties keyed by name; each value is selected by metadataType.
	CustomProperties *map[string]openapi.MetadataValue `json:"customProperties,omitempty"`
	// Human-readable summary of the model.
	Description *string `json:"description,omitempty" example:"An instruction-tuned language model."`
	// Supported language codes, using ISO 639 identifiers where applicable.
	Language []string `json:"language,omitempty" example:"en,es"`
	// Last update time in milliseconds since the Unix epoch, encoded as a string.
	LastUpdateTimeSinceEpoch *string `json:"lastUpdateTimeSinceEpoch,omitempty" format:"int64" example:"1700000000000"`
	// Library used to load or run the model.
	LibraryName *string `json:"libraryName,omitempty" example:"transformers"`
	// Short name or identifier of the model's license.
	License *string `json:"license,omitempty" example:"apache-2.0"`
	// URL of the full license text.
	LicenseLink *string `json:"licenseLink,omitempty" format:"uri" example:"https://www.apache.org/licenses/LICENSE-2.0"`
	// URL of the model logo; data URLs are supported.
	Logo *string `json:"logo,omitempty" example:"https://example.com/model-logo.png"`
	// Publication or support maturity level of the model.
	Maturity *string `json:"maturity,omitempty" example:"Generally Available"`
	// Model identifier within its source; it may include slashes.
	Name string `json:"name" example:"ibm-granite/granite-3.1-8b-base"`
	// Organization or entity that provides the model.
	Provider *string `json:"provider,omitempty" example:"IBM"`
	// Full model documentation in Markdown.
	Readme *string `json:"readme,omitempty" example:"# Model documentation"`
	// Identifier of the catalog source providing the model.
	SourceId *string `json:"source_id,omitempty" example:"community"`
	// Tasks the model is designed to perform.
	Tasks []string `json:"tasks,omitempty" example:"text-generation"`
	// Tasks validated end-to-end for this model, using the same names as tasks.
	ValidatedTasks []string `json:"validatedTasks,omitempty" example:"tool-calling"`
	// Validated runtime settings for deploying and serving this model.
	ServingConfig *ServingConfig `json:"servingConfig,omitempty"`
}

// CatalogModelList is a page of catalog models.
// @Description A page of catalog models with pagination information.
type CatalogModelList struct {
	// Opaque token for retrieving the next page; empty when no further page exists.
	NextPageToken string `json:"nextPageToken"`
	// Maximum number of models requested per page.
	PageSize int32 `json:"pageSize" example:"10"`
	// Number of models in this page.
	Size int32 `json:"size" example:"1"`
	// Models returned in this page.
	Items []CatalogModel `json:"items"`
}

type FilterRange struct {
	// Largest supported numeric filter value.
	Max *float32 `json:"max,omitempty" example:"100"`
	// Smallest supported numeric filter value.
	Min *float32 `json:"min,omitempty" example:"0"`
}

type FilterOption struct {
	// Numeric bounds, when the filter supports a range.
	Range *FilterRange `json:"range,omitempty"`
	// Type of values accepted by this filter.
	Type FilterOptionType `json:"type" example:"string"`
	// Allowed values for categorical filters.
	Values []interface{} `json:"values,omitempty"`
}

type FieldFilter struct {
	// Comparison operator used by the named query.
	Operator string `json:"operator" example:"="`
	// Value compared against the selected field.
	Value interface{} `json:"value"`
}

type FilterOptionType string

type FilterOptionsList struct {
	// Filterable field names and their allowed values or numeric bounds.
	Filters *map[string]FilterOption `json:"filters,omitempty"`
	// Reusable query templates indexed by query name and field name.
	NamedQueries *map[string]map[string]FieldFilter `json:"namedQueries,omitempty"`
}
