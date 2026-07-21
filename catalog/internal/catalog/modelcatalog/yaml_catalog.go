package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	catalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	models "github.com/kubeflow/hub/internal/platform/db/entity"
)

const (
	yamlCatalogPathKey = "yamlCatalogPath"
	excludedModelsKey  = "excludedModels"
)

// convertMetadataValueToProperty converts a MetadataValue to a Properties object
// This helper eliminates code duplication when converting custom properties
func convertMetadataValueToProperty(key string, value apimodels.MetadataValue) models.Properties {
	// Handle different MetadataValue types
	if value.MetadataStringValue != nil {
		return models.NewStringProperty(key, value.MetadataStringValue.StringValue, true)
	} else if value.MetadataIntValue != nil {
		// MetadataIntValue.IntValue is a string, need to convert to int32
		if intVal, err := strconv.ParseInt(value.MetadataIntValue.IntValue, 10, 32); err == nil {
			return models.NewIntProperty(key, int32(intVal), true)
		} else {
			// If parsing fails, store as string
			return models.NewStringProperty(key, value.MetadataIntValue.IntValue, true)
		}
	} else if value.MetadataDoubleValue != nil {
		return models.NewDoubleProperty(key, value.MetadataDoubleValue.DoubleValue, true)
	} else if value.MetadataBoolValue != nil {
		return models.NewBoolProperty(key, value.MetadataBoolValue.BoolValue, true)
	} else {
		// For complex types, serialize to JSON
		if jsonBytes, err := json.Marshal(value); err == nil {
			return models.NewStringProperty(key, string(jsonBytes), true)
		}
		// Fallback to empty string if JSON marshaling fails
		return models.NewStringProperty(key, "", true)
	}
}

// convertCustomProperties converts a map of custom properties to a slice of Properties
func convertCustomProperties(customProps *map[string]apimodels.MetadataValue) []models.Properties {
	if customProps == nil {
		return nil
	}

	var properties []models.Properties
	for key, value := range *customProps {
		properties = append(properties, convertMetadataValueToProperty(key, value))
	}
	return properties
}

func init() {
	if err := RegisterModelProvider("yaml", newYamlModelProvider); err != nil {
		panic(err)
	}
}

type yamlModel struct {
	apimodels.CatalogModel `yaml:",inline"`
	Artifacts              []*yamlArtifact `yaml:"artifacts"`
}

type yamlArtifact struct {
	apimodels.CatalogArtifact
}

// convertModelAttributes converts basic model attributes and timestamps
func (ym *yamlModel) convertModelAttributes() *catalogmodels.CatalogModelAttributes {
	attrs := &catalogmodels.CatalogModelAttributes{
		Name: &ym.Name,
	}

	// Convert timestamps
	if ym.CreateTimeSinceEpoch != nil {
		if createTime, err := strconv.ParseInt(*ym.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &createTime
		}
	}

	if ym.LastUpdateTimeSinceEpoch != nil {
		if updateTime, err := strconv.ParseInt(*ym.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	return attrs
}

// convertModelProperties converts model properties to regular and custom properties
func (ym *yamlModel) convertModelProperties() ([]models.Properties, []models.Properties) {
	var properties []models.Properties
	var customProperties []models.Properties

	// Regular properties
	if ym.Description != nil {
		properties = append(properties, models.NewStringProperty("description", *ym.Description, false))
	}
	if ym.Readme != nil {
		properties = append(properties, models.NewStringProperty("readme", *ym.Readme, false))
	}
	if ym.Maturity != nil {
		properties = append(properties, models.NewStringProperty("maturity", *ym.Maturity, false))
	}
	if ym.Provider != nil {
		properties = append(properties, models.NewStringProperty("provider", *ym.Provider, false))
	}
	if ym.Logo != nil {
		properties = append(properties, models.NewStringProperty("logo", *ym.Logo, false))
	}
	if ym.License != nil {
		humanReadableLicense := basecatalog.TransformLicenseToHumanReadable(*ym.License)
		properties = append(properties, models.NewStringProperty("license", humanReadableLicense, false))
	}
	if ym.LicenseLink != nil {
		properties = append(properties, models.NewStringProperty("license_link", *ym.LicenseLink, false))
	}
	if ym.LibraryName != nil {
		properties = append(properties, models.NewStringProperty("library_name", *ym.LibraryName, false))
	}
	if ym.SourceId != nil {
		properties = append(properties, models.NewStringProperty("source_id", *ym.SourceId, false))
	}

	// Convert array properties as struct properties
	if ym.Language == nil {
		ym.Language = []string{}
	}
	if languageJSON, err := json.Marshal(ym.Language); err == nil {
		properties = append(properties, models.NewStringProperty("language", string(languageJSON), false))
	}

	if ym.Tasks == nil {
		ym.Tasks = []string{}
	}
	if tasksJSON, err := json.Marshal(ym.Tasks); err == nil {
		properties = append(properties, models.NewStringProperty("tasks", string(tasksJSON), false))
	}

	if ym.ValidatedTasks == nil {
		ym.ValidatedTasks = []string{}
	}
	if validatedTasksJSON, err := json.Marshal(ym.ValidatedTasks); err == nil {
		properties = append(properties, models.NewStringProperty("validated_tasks", string(validatedTasksJSON), false))
	}

	if ym.ServingConfig != nil {
		if servingConfigJSON, err := json.Marshal(ym.ServingConfig); err == nil {
			properties = append(properties, models.NewStringProperty("serving_config", string(servingConfigJSON), false))
		}
	}

	// Drop hardware_tag if its value is empty
	if val, ok := ym.CustomProperties["hardware_tag"]; ok &&
		val.MetadataStringValue != nil && val.MetadataStringValue.StringValue == "" {
		delete(ym.CustomProperties, "hardware_tag")
	}

	// Convert custom properties from the YAML model
	if customProps := convertCustomProperties(&ym.CustomProperties); customProps != nil {
		customProperties = append(customProperties, customProps...)
	}

	return properties, customProperties
}

// convertModelArtifact converts a CatalogModelArtifact to database format
func convertModelArtifact(artifact *apimodels.CatalogModelArtifact) *sharedmodels.CatalogArtifact {
	modelArtifact := &catalogmodels.CatalogModelArtifactImpl{}

	// Set basic attributes
	attrs := &catalogmodels.CatalogModelArtifactAttributes{
		URI: &artifact.Uri,
	}

	// Convert timestamps
	if artifact.CreateTimeSinceEpoch != nil {
		if createTime, err := strconv.ParseInt(*artifact.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &createTime
		}
	}
	if artifact.LastUpdateTimeSinceEpoch != nil {
		if updateTime, err := strconv.ParseInt(*artifact.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	modelArtifact.Attributes = attrs

	var artifactProperties []models.Properties
	artifactProperties = append(artifactProperties, models.NewStringProperty("uri", artifact.Uri, false))

	// Convert custom properties using helper function
	if customProps := convertCustomProperties(&artifact.CustomProperties); customProps != nil {
		modelArtifact.CustomProperties = &customProps
	}

	modelArtifact.Properties = &artifactProperties

	return &sharedmodels.CatalogArtifact{
		CatalogModelArtifact: modelArtifact,
	}
}

// convertMetricsArtifact converts a CatalogMetricsArtifact to database format
func convertMetricsArtifact(artifact *apimodels.CatalogMetricsArtifact) *sharedmodels.CatalogArtifact {
	metricsArtifact := &catalogmodels.CatalogMetricsArtifactImpl{}

	// Set basic attributes
	attrs := &catalogmodels.CatalogMetricsArtifactAttributes{
		MetricsType: catalogmodels.MetricsType(artifact.MetricsType),
	}

	// Convert timestamps
	if artifact.CreateTimeSinceEpoch != nil {
		if createTime, err := strconv.ParseInt(*artifact.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &createTime
		}
	}
	if artifact.LastUpdateTimeSinceEpoch != nil {
		if updateTime, err := strconv.ParseInt(*artifact.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	metricsArtifact.Attributes = attrs

	// Handle properties
	var artifactProperties []models.Properties
	artifactProperties = append(artifactProperties, models.NewStringProperty("metricsType", artifact.MetricsType, false))

	// Convert custom properties using helper function
	if customProps := convertCustomProperties(&artifact.CustomProperties); customProps != nil {
		metricsArtifact.CustomProperties = &customProps
	}

	metricsArtifact.Properties = &artifactProperties

	return &sharedmodels.CatalogArtifact{
		CatalogMetricsArtifact: metricsArtifact,
	}
}

func (ym *yamlModel) ToModelProviderRecord() ModelProviderRecord {
	model := catalogmodels.CatalogModelImpl{}
	artifacts := make([]sharedmodels.CatalogArtifact, len(ym.Artifacts))

	// Convert model attributes
	model.Attributes = ym.convertModelAttributes()

	// Convert model properties
	properties, customProperties := ym.convertModelProperties()
	if len(properties) > 0 {
		model.Properties = &properties
	}
	if len(customProperties) > 0 {
		model.CustomProperties = &customProperties
	}

	// Convert artifacts with URI validation
	for j := range ym.Artifacts {
		if ym.Artifacts[j].CatalogModelArtifact != nil {
			// Validate artifact URI before processing
			if err := basecatalog.ValidateArtifactURI(ym.Artifacts[j].CatalogModelArtifact.Uri); err != nil {
				return ModelProviderRecord{
					Error: fmt.Errorf("model %q artifact %d: %w", ym.Name, j, err),
				}
			}
			artifacts[j] = *convertModelArtifact(ym.Artifacts[j].CatalogModelArtifact)
		} else if ym.Artifacts[j].CatalogMetricsArtifact != nil {
			artifacts[j] = *convertMetricsArtifact(ym.Artifacts[j].CatalogMetricsArtifact)
		}
	}

	return ModelProviderRecord{
		Model:     &model,
		Artifacts: artifacts,
	}
}

func (a *yamlArtifact) UnmarshalJSON(buf []byte) error {
	// This is very similar to generated code to unmarshal a
	// CatalogArtifact, but this version properly handles artifacts without
	// an artifactType, which is important for backwards compatibility.
	var yat struct {
		ArtifactType string `json:"artifactType"`
	}

	err := json.Unmarshal(buf, &yat)
	if err != nil {
		return err
	}

	switch yat.ArtifactType {
	case "model-artifact", "":
		err = json.Unmarshal(buf, &a.CatalogArtifact.CatalogModelArtifact)
		if a.CatalogArtifact.CatalogModelArtifact != nil {
			// Ensure artifactType is set even if it wasn't initially.
			a.CatalogArtifact.CatalogModelArtifact.ArtifactType = "model-artifact"
		}
	case "metrics-artifact":
		err = json.Unmarshal(buf, &a.CatalogArtifact.CatalogMetricsArtifact)
	default:
		return fmt.Errorf("unknown artifactType: %s", yat.ArtifactType)
	}

	return err
}

type yamlCatalog struct {
	Source string      `yaml:"source"`
	Models []yamlModel `yaml:"models"`
}

// SourceReadError wraps a file-level read or parse failure so the consumer can
// distinguish it from per-model validation errors.
type SourceReadError struct {
	err error
}

func (e *SourceReadError) Error() string { return e.err.Error() }
func (e *SourceReadError) Unwrap() error { return e.err }

type yamlModelProvider struct {
	path   string
	filter *ModelFilter
}

func (p *yamlModelProvider) Models(ctx context.Context) (<-chan ModelProviderRecord, error) {
	ch := make(chan ModelProviderRecord)
	go func() {
		defer close(ch)

		// Set up the watcher before the initial read so that changes arriving
		// during or just after the read are not missed.
		changes, watchErr := basecatalog.GetMonitor().Path(ctx, p.path)
		if watchErr != nil {
			glog.Errorf("unable to watch YAML catalog file: %v", watchErr)
		}

		catalog, err := p.read()
		if err != nil {
			p.emitError(ctx, err, ch)
		} else {
			p.emit(ctx, catalog, ch)
		}

		if changes == nil {
			// Watcher setup failed; no live updates possible.
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-changes:
				if !ok {
					return
				}
				glog.Infof("Reloading YAML catalog %s", p.path)
				catalog, err = p.read()
				if err != nil {
					glog.Errorf("unable to load YAML catalog: %v", err)
					p.emitError(ctx, err, ch)
					continue
				}
				p.emit(ctx, catalog, ch)
			}
		}
	}()

	return ch, nil
}

// emitError wraps readErr as a SourceReadError and sends it followed by a
// sentinel so the consumer can update the source status to "error" while
// keeping the channel alive for future reload attempts.
func (p *yamlModelProvider) emitError(ctx context.Context, readErr error, out chan<- ModelProviderRecord) {
	done := ctx.Done()
	select {
	case out <- ModelProviderRecord{Error: &SourceReadError{err: readErr}}:
	case <-done:
		return
	}
	select {
	case out <- ModelProviderRecord{}:
	case <-done:
	}
}

func (p *yamlModelProvider) read() (*yamlCatalog, error) {
	buf, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s file: %v", yamlCatalogPathKey, err)
	}

	var catalog yamlCatalog
	if err = yaml.UnmarshalStrict(buf, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse %s file: %v", yamlCatalogPathKey, err)
	}

	return &catalog, nil
}

func (p *yamlModelProvider) emit(ctx context.Context, catalog *yamlCatalog, out chan<- ModelProviderRecord) {
	done := ctx.Done()
	for _, model := range catalog.Models {
		if !p.filter.Allows(model.Name) {
			continue
		}

		select {
		case out <- model.ToModelProviderRecord():
		case <-done:
			return
		}
	}

	// Send an empty record to indicate that we're done with the batch.
	select {
	case out <- ModelProviderRecord{}:
	case <-done:
	}
}

func newYamlModelProvider(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
	p := &yamlModelProvider{}

	path, exists := source.Properties[yamlCatalogPathKey].(string)
	if !exists || path == "" {
		return nil, fmt.Errorf("missing %s string property", yamlCatalogPathKey)
	}

	if filepath.IsAbs(path) {
		p.path = path
	} else {
		p.path = filepath.Join(reldir, path)
	}

	var legacyExcluded []string
	if raw, exists := source.Properties[excludedModelsKey]; exists {
		values, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("%q property should be a list", excludedModelsKey)
		}

		legacyExcluded = make([]string, len(values))
		for i, value := range values {
			nameStr, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s: invalid list: index %d: wanted string, got %T", excludedModelsKey, i, value)
			}
			legacyExcluded[i] = nameStr
		}
	}

	filter, err := NewModelFilterFromSource(source, nil, legacyExcluded)
	if err != nil {
		return nil, err
	}
	p.filter = filter

	return p.Models(ctx)
}
