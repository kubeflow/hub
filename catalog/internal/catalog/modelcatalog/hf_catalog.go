package modelcatalog

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	catalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	models "github.com/kubeflow/hub/internal/platform/db/entity"
)

const (
	defaultHuggingFaceURL = "https://huggingface.co"
	defaultAPIKeyEnvVar   = "HF_API_KEY"
	apiKeyEnvVarPrefix    = "HF_API_KEY_"
	urlKey                = "url"
	apiKeyEnvVarKey       = "apiKeyEnvVar"
	maxModelsKey          = "maxModels"
	syncIntervalKey       = "syncInterval"
	allowedOrgKey         = "allowedOrganization"

	// defaultMaxModels is the default limit for models fetched PER PATTERN.
	// This limit is applied independently to each pattern in includedModels
	// (e.g., "ibm-granite/*", "meta-llama/*") to prevent overloading the
	// Hugging Face API with too many requests and to respect rate limiting.
	//
	// Example: with maxModels=100 and 3 patterns, up to 300 models total may be fetched.
	// Set to 0 to disable the limit (not recommended for large organizations).
	defaultMaxModels = 500

	// defaultSyncInterval is the default interval for periodic syncing of models.
	// This can be overridden via the syncInterval property in the source configuration.
	// For testing, a shorter interval (e.g., "1s" or "10s") can be used to speed up tests.
	defaultSyncInterval = 24 * time.Hour
)

// ModelType represents the classification of a model based on its tasks.
type ModelType string

const (
	ModelTypeGenerative ModelType = "generative"
	ModelTypePredictive ModelType = "predictive"
	ModelTypeUnknown    ModelType = "unknown"
)

// gatedString is a custom type that can unmarshal both boolean and string values from JSON
// It converts booleans to strings (false -> "false", true -> "true")
type gatedString string

// UnmarshalJSON implements json.Unmarshaler to handle both boolean and string values
func (g *gatedString) UnmarshalJSON(data []byte) error {
	// Handle null/empty
	if len(data) == 0 || string(data) == "null" {
		*g = gatedString("")
		return nil
	}

	// Try to unmarshal as boolean first (handles true/false)
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*g = gatedString(strconv.FormatBool(b))
		return nil
	}

	// If not a boolean, try as string (handles quoted strings)
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("gated field must be boolean or string, got: %s", string(data))
	}
	*g = gatedString(s)
	return nil
}

// String returns the string value
func (g gatedString) String() string {
	return string(g)
}

// hfModel implements apimodels.CatalogModel and populates it from Hugging Face API data
type hfModel struct {
	apimodels.CatalogModel
}

type hfModelProvider struct {
	client         *http.Client
	sourceId       string
	apiKey         string
	baseURL        string
	includedModels []string
	filter         *ModelFilter
	// maxModels limits how many models to fetch PER PATTERN (e.g., per "org/*").
	// This is applied independently to each pattern to respect Hugging Face API rate limits.
	// A value of 0 means no limit.
	maxModels int
	// syncInterval is the interval for periodic syncing of models.
	// This can be configured via the syncInterval property in the source configuration.
	syncInterval time.Duration
}

// hfModelInfo represents the structure of Hugging Face API model information
type hfModelInfo struct {
	ID          string      `json:"id"`
	Author      string      `json:"author,omitempty"`
	Sha         string      `json:"sha,omitempty"`
	CreatedAt   string      `json:"createdAt,omitempty"`
	UpdatedAt   string      `json:"lastModified,omitempty"`
	Private     bool        `json:"private,omitempty"`
	Gated       gatedString `json:"gated,omitempty"`
	Downloads   int         `json:"downloads,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	PipelineTag string      `json:"pipeline_tag,omitempty"`
	LibraryName string      `json:"library_name,omitempty"`
	ModelID     string      `json:"modelId,omitempty"`
	Task        string      `json:"task,omitempty"`
	Siblings    []hfFile    `json:"siblings,omitempty"`
	Config      *hfConfig   `json:"config,omitempty"`
	CardData    *hfCard     `json:"cardData,omitempty"`
}

type hfFile struct {
	RFileName string `json:"rfilename"`
}

type hfConfig struct {
	Architectures []string `json:"architectures,omitempty"`
	ModelType     string   `json:"model_type,omitempty"`
}

type hfCard struct {
	Data map[string]any `json:"data,omitempty"`
}

//go:embed assets/catalog_logo.svg
var catalogLogoSVG []byte

var (
	catalogModelLogo = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(catalogLogoSVG)
)

// generativeTasks contains the set of Hugging Face task types that indicate generative models.
// These models produce novel content (text, images, audio, etc.).
var generativeTasks = map[string]bool{
	"text-generation":                true,
	"summarization":                  true,
	"translation":                    true,
	"text-to-image":                  true,
	"unconditional-image-generation": true,
	"image-to-image":                 true,
	"text-to-speech":                 true,
	"audio-to-audio":                 true,
}

// predictiveTasks contains the set of Hugging Face task types that indicate predictive models.
// These models produce classifications, scores, labels, or predictions.
var predictiveTasks = map[string]bool{
	"text-classification":         true,
	"image-classification":        true,
	"zero-shot-classification":    true,
	"audio-classification":        true,
	"question-answering":          true,
	"document-question-answering": true,
	"object-detection":            true,
	"image-segmentation":          true,
	"keypoint-detection":          true,
	"feature-extraction":          true,
	"image-feature-extraction":    true,
	"fill-mask":                   true,
}

// classifyModelTypeFromTasks classifies a model as generative, predictive, or unknown based on its task information.
func classifyModelTypeFromTasks(tasks []string) ModelType {
	if len(tasks) == 0 {
		return ModelTypeUnknown
	}

	hasPredictive := false

	// If model has both generative and predictive tasks, return generative
	for _, task := range tasks {
		task = strings.ToLower(strings.TrimSpace(task))
		if generativeTasks[task] {
			return ModelTypeGenerative
		} else if predictiveTasks[task] {
			hasPredictive = true
		}
	}

	if hasPredictive {
		return ModelTypePredictive
	}

	return ModelTypeUnknown
}

// populateFromHFInfo populates the hfModel's CatalogModel fields from Hugging Face API data
func (hfm *hfModel) populateFromHFInfo(ctx context.Context, provider *hfModelProvider, hfInfo *hfModelInfo, sourceId string, originalModelName string) {
	// Set model name
	modelName := hfInfo.ID
	if modelName == "" {
		modelName = hfInfo.ModelID
	}
	if modelName == "" {
		modelName = originalModelName
	}
	hfm.Name = modelName

	// Set ExternalId
	if hfInfo.ID != "" {
		hfm.ExternalId = &hfInfo.ID
	}

	// Set SourceId
	if sourceId != "" {
		hfm.SourceId = &sourceId
	}

	// Convert timestamps
	if hfInfo.CreatedAt != "" {
		if createTime, err := parseHFTime(hfInfo.CreatedAt); err == nil {
			createTimeStr := strconv.FormatInt(createTime, 10)
			hfm.CreateTimeSinceEpoch = &createTimeStr
		}
	}
	if hfInfo.UpdatedAt != "" {
		if updateTime, err := parseHFTime(hfInfo.UpdatedAt); err == nil {
			updateTimeStr := strconv.FormatInt(updateTime, 10)
			hfm.LastUpdateTimeSinceEpoch = &updateTimeStr
		}
	}

	// Extract license from tags
	// Skip license tags in custom properties to avoid duplication
	var filteredTags []string
	if len(hfInfo.Tags) > 0 {
		filteredTags = make([]string, 0, len(hfInfo.Tags))
		for _, tag := range hfInfo.Tags {
			if license, ok := strings.CutPrefix(tag, "license:"); ok {
				// Extract license (only first one)
				if hfm.License == nil {
					if license != "" {
						license = basecatalog.TransformLicenseToHumanReadable(license)
						hfm.License = &license
					}
				}
			} else {
				filteredTags = append(filteredTags, tag)
			}
		}
	}

	// Extract README from sibling files first (preferred source)
	// Check for common README filenames
	readmeFilenames := []string{"README.md", "readme.md", "Readme.md", "README", "readme"}

	for _, sibling := range hfInfo.Siblings {
		for _, readmeFilename := range readmeFilenames {
			if sibling.RFileName == readmeFilename {
				if readmeContent, err := provider.fetchFileContent(ctx, modelName, readmeFilename); err == nil {
					hfm.Readme = &readmeContent
					break
				} else {
					glog.V(2).Infof("Failed to fetch README from sibling file %s for model %s: %v", readmeFilename, modelName, err)
				}
			}
		}
		if hfm.Readme != nil {
			break
		}
	}

	// Extract description from cardData if available
	if hfInfo.CardData != nil && hfInfo.CardData.Data != nil {
		// Extract description from cardData if available
		if desc, ok := hfInfo.CardData.Data["description"].(string); ok && desc != "" {
			hfm.Description = &desc
		}

		// Extract language from cardData if available
		if langData, ok := hfInfo.CardData.Data["language"].([]any); ok && len(langData) > 0 {
			languages := make([]string, 0, len(langData))
			for _, lang := range langData {
				if langStr, ok := lang.(string); ok && langStr != "" {
					languages = append(languages, langStr)
				}
			}
			if len(languages) > 0 {
				hfm.Language = languages
			}
		}

		// Extract license link from cardData if available
		// Check common field names for license link/URL
		if hfm.LicenseLink == nil {
			licenseLinkFields := []string{"license_link", "licenseLink", "license_url", "licenseUrl", "license"}
			for _, field := range licenseLinkFields {
				if link, ok := hfInfo.CardData.Data[field].(string); ok && link != "" {
					if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
						hfm.LicenseLink = &link
						break
					}
				}
			}
		}

	}

	// Set provider from author
	if hfInfo.Author != "" {
		hfm.Provider = &hfInfo.Author
	}

	// Set library name
	if hfInfo.LibraryName != "" {
		hfm.LibraryName = &hfInfo.LibraryName
	}

	// Set logo
	hfm.Logo = &catalogModelLogo

	// Convert tasks
	var tasks []string
	if hfInfo.Task != "" {
		tasks = append(tasks, hfInfo.Task)
	}
	if hfInfo.PipelineTag != "" && hfInfo.PipelineTag != hfInfo.Task {
		tasks = append(tasks, hfInfo.PipelineTag)
	}
	if len(tasks) > 0 {
		hfm.Tasks = tasks
	}

	// Classify model type based on tasks (heuristic-based classification)
	modelType := classifyModelTypeFromTasks(tasks)

	// Convert tags and other metadata to custom properties
	customProps := make(map[string]apimodels.MetadataValue)

	// Add model_type classification (always set, including "unknown")
	customProps["model_type"] = apimodels.MetadataValue{
		MetadataStringValue: &apimodels.MetadataStringValue{
			StringValue: string(modelType),
		},
	}

	customProps["hf_private"] = apimodels.MetadataValue{
		MetadataStringValue: &apimodels.MetadataStringValue{
			StringValue: strconv.FormatBool(hfInfo.Private),
		},
	}

	customProps["hf_gated"] = apimodels.MetadataValue{
		MetadataStringValue: &apimodels.MetadataStringValue{
			StringValue: hfInfo.Gated.String(),
		},
	}

	if len(filteredTags) > 0 {
		if tagsJSON, err := json.Marshal(filteredTags); err == nil {
			customProps["hf_tags"] = apimodels.MetadataValue{
				MetadataStringValue: &apimodels.MetadataStringValue{
					StringValue: string(tagsJSON),
				},
			}
		}
	}

	if hfInfo.Config != nil {
		if len(hfInfo.Config.Architectures) > 0 {
			if archJSON, err := json.Marshal(hfInfo.Config.Architectures); err == nil {
				customProps["hf_architectures"] = apimodels.MetadataValue{
					MetadataStringValue: &apimodels.MetadataStringValue{
						StringValue: string(archJSON),
					},
				}
			}
		}
		if hfInfo.Config.ModelType != "" {
			customProps["hf_model_type"] = apimodels.MetadataValue{
				MetadataStringValue: &apimodels.MetadataStringValue{
					StringValue: hfInfo.Config.ModelType,
				},
			}
		}
	}

	if len(customProps) > 0 {
		hfm.SetCustomProperties(customProps)
	}
}

func (p *hfModelProvider) Models(ctx context.Context) (<-chan ModelProviderRecord, error) {
	// Read the catalog - may return partial results with an error if any models fail to be loaded
	catalog, fetchErr := p.getModelsFromHF(ctx)

	// If we got no models AND an error, return the error immediately
	if fetchErr != nil && len(catalog) == 0 {
		return nil, fetchErr
	}

	ch := make(chan ModelProviderRecord)
	go func() {
		defer close(ch)

		// Send the initial list right away, then send error status if there was a partial failure
		p.emitWithError(ctx, catalog, fetchErr, ch)

		// Set up periodic polling with configurable interval
		ticker := time.NewTicker(p.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				glog.Infof("Periodic sync: reprocessing all models for source %s", p.sourceId)
				catalog, err := p.getModelsFromHF(ctx)
				// Even if there's an error, emit successful models first, then signal the error
				if len(catalog) > 0 || err == nil {
					p.emitWithError(ctx, catalog, err, ch)
				} else {
					// No models and an error - just log it
					glog.Errorf("unable to reprocess Hugging Face models: %v", err)
				}
			}
		}
	}()

	return ch, nil
}

// expandModelNames takes a list of model identifiers (which may include wildcards)
// and returns a list of concrete model names by expanding any wildcard patterns.
// Uses the same logic as FetchModelNamesForPreview.
func (p *hfModelProvider) expandModelNames(ctx context.Context, modelIdentifiers []string) ([]string, error) {
	var allNames []string
	var failedPatterns []string
	var wildcardPatterns []string

	for _, pattern := range modelIdentifiers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		patternType, org, searchPrefix := parseModelPattern(pattern)

		switch patternType {
		case PatternInvalid:
			return nil, fmt.Errorf("wildcard pattern %q is not supported - Hugging Face requires a specific organization (e.g., 'ibm-granite/*' or 'meta-llama/Llama-2-*')", pattern)

		case PatternOrgAll, PatternOrgPrefix:
			wildcardPatterns = append(wildcardPatterns, pattern)
			glog.Infof("Expanding wildcard pattern: %s (org=%s, prefix=%s)", pattern, org, searchPrefix)
			models, err := p.listModelsByAuthor(ctx, org, searchPrefix)
			if err != nil {
				failedPatterns = append(failedPatterns, pattern)
				glog.Warningf("Failed to expand wildcard pattern %s: %v", pattern, err)
				continue
			}
			allNames = append(allNames, models...)

		case PatternExact:
			// Direct model name - no expansion needed
			allNames = append(allNames, pattern)
		}
	}

	// Check error conditions for wildcard pattern failures
	if len(wildcardPatterns) > 0 && len(allNames) == 0 {
		// All wildcard patterns failed AND no results from exact patterns - this is an error
		if len(failedPatterns) > 0 {
			return nil, fmt.Errorf("no models found: %v", failedPatterns)
		} else {
			return nil, fmt.Errorf("no models found")
		}
	} else if len(failedPatterns) > 0 {
		// Some patterns failed but we have results - log warning and continue with partial results
		glog.Warningf("Some wildcard patterns failed to expand and were skipped: %v", failedPatterns)
	}

	return allNames, nil
}

func (p *hfModelProvider) getModelsFromHF(ctx context.Context) ([]ModelProviderRecord, error) {
	// First expand any wildcard patterns to concrete model names
	expandedModels, err := p.expandModelNames(ctx, p.includedModels)
	if err != nil {
		return nil, fmt.Errorf("failed to expand model patterns: %w", err)
	}

	var records []ModelProviderRecord
	currentTime := time.Now().UnixMilli()
	lastSyncedStr := strconv.FormatInt(currentTime, 10)

	var failedModels []string

	for _, modelName := range expandedModels {
		// Skip if excluded - check before fetching to avoid unnecessary API calls
		if !p.filter.Allows(modelName) {
			glog.V(2).Infof("Skipping excluded model: %s", modelName)
			continue
		}

		modelInfo, err := p.fetchModelInfo(ctx, modelName)
		if err != nil {
			glog.Errorf("Failed to fetch model info for %s: %v", modelName, err)
			failedModels = append(failedModels, modelName)
			continue
		}

		record := p.convertHFModelToRecord(ctx, modelInfo, modelName)

		// Additional safety check: verify the final model name is not excluded
		// (in case the model name changed during conversion, e.g., from hfInfo.ID)
		if record.Model.GetAttributes() != nil && record.Model.GetAttributes().Name != nil {
			finalModelName := *record.Model.GetAttributes().Name
			if !p.filter.Allows(finalModelName) {
				glog.V(2).Infof("Skipping excluded model (after conversion): %s", finalModelName)
				continue
			}
		}

		// Add last_synced property to the model
		if record.Model != nil {
			if modelImpl, ok := record.Model.(*catalogmodels.CatalogModelImpl); ok {
				customProps := modelImpl.CustomProperties
				if customProps == nil {
					customProps = &[]models.Properties{}
				}
				// Add last_synced property
				*customProps = append(*customProps, models.NewStringProperty("last_synced", lastSyncedStr, true))
				modelImpl.CustomProperties = customProps
			}
		}

		records = append(records, record)
	}

	if len(failedModels) > 0 {
		return records, &PartiallyAvailableError{FailedModels: failedModels}
	}

	return records, nil
}

func (p *hfModelProvider) fetchModelInfo(ctx context.Context, modelName string) (*hfModelInfo, error) {
	// The HF API requires the full model identifier: org/model-name (aka repo/model-name)

	// Normalize the model name (remove any leading/trailing slashes)
	modelName = strings.Trim(modelName, "/")

	// Construct the API URL with the full model identifier
	apiURL := fmt.Sprintf("%s/api/models/%s", p.baseURL, modelName)

	glog.V(2).Infof("Fetching Hugging Face model info from: %s", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header (Hugging Face API expects this)
	req.Header.Set("User-Agent", "model-registry-catalog")

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model info for %s: %w", modelName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Hugging Face API returned status %d for model %s: %s", resp.StatusCode, modelName, string(bodyBytes))
	}

	var modelInfo hfModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, fmt.Errorf("failed to decode model info for %s: %w", modelName, err)
	}

	// Ensure ID is set from modelName if not present in API response
	if modelInfo.ID == "" {
		modelInfo.ID = modelName
	}

	return &modelInfo, nil
}

// fetchFileContent fetches the content of a file from Hugging Face repository
func (p *hfModelProvider) fetchFileContent(ctx context.Context, modelName string, filename string) (string, error) {
	// Normalize the model name (remove any leading/trailing slashes)
	modelName = strings.Trim(modelName, "/")

	// Construct the API URL for raw file content
	// Hugging Face API endpoint: {baseURL}/{model_id}/raw/main/{filename}
	apiURL := fmt.Sprintf("%s/%s/raw/main/%s", p.baseURL, modelName, filename)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", "model-registry-catalog")

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch file %s for model %s: %w", filename, modelName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Hugging Face API returned status %d for file %s in model %s: %s", resp.StatusCode, filename, modelName, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read file content for %s in model %s: %w", filename, modelName, err)
	}

	return string(bodyBytes), nil
}

func (p *hfModelProvider) convertHFModelToRecord(ctx context.Context, hfInfo *hfModelInfo, originalModelName string) ModelProviderRecord {
	// Create hfModel and populate it from HF API data
	hfm := &hfModel{}
	hfm.populateFromHFInfo(ctx, p, hfInfo, p.sourceId, originalModelName)

	// Convert to database model
	model := catalogmodels.CatalogModelImpl{}

	// Convert model attributes
	modelName := hfm.Name
	attrs := &catalogmodels.CatalogModelAttributes{
		Name:       &modelName,
		ExternalID: hfm.ExternalId,
	}

	// Convert timestamps if available
	if hfm.CreateTimeSinceEpoch != nil {
		if createTime, err := strconv.ParseInt(*hfm.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &createTime
		}
	}
	if hfm.LastUpdateTimeSinceEpoch != nil {
		if updateTime, err := strconv.ParseInt(*hfm.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &updateTime
		}
	}

	model.Attributes = attrs

	// Convert model properties
	properties, customProperties := convertHFModelProperties(&hfm.CatalogModel)
	if len(properties) > 0 {
		model.Properties = &properties
	}
	if len(customProperties) > 0 {
		model.CustomProperties = &customProperties
	}

	// Create model artifact with hf:// protocol for KServe CSI deployment
	artifacts := []sharedmodels.CatalogArtifact{}
	if hfm.ExternalId != nil && *hfm.ExternalId != "" {
		// Construct hf:// URI using the Hugging Face model ID
		hfUri := fmt.Sprintf("hf://%s", *hfm.ExternalId)
		artifactType := "model-artifact"
		artifactName := fmt.Sprintf("%s-hf-artifact", modelName)

		// Create CatalogModelArtifact
		modelArtifact := &catalogmodels.CatalogModelArtifactImpl{}
		modelArtifact.Attributes = &catalogmodels.CatalogModelArtifactAttributes{
			Name:         &artifactName,
			URI:          &hfUri,
			ArtifactType: &artifactType,
			ExternalID:   hfm.ExternalId,
		}

		// Add timestamps if available from parent model
		if attrs.CreateTimeSinceEpoch != nil {
			modelArtifact.Attributes.CreateTimeSinceEpoch = attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			modelArtifact.Attributes.LastUpdateTimeSinceEpoch = attrs.LastUpdateTimeSinceEpoch
		}

		artifacts = append(artifacts, sharedmodels.CatalogArtifact{
			CatalogModelArtifact: modelArtifact,
		})
	}

	return ModelProviderRecord{
		Model:     &model,
		Artifacts: artifacts,
	}
}

// convertHFModelProperties converts CatalogModel properties to database format
func convertHFModelProperties(catalogModel *apimodels.CatalogModel) ([]models.Properties, []models.Properties) {
	var properties []models.Properties
	var customProperties []models.Properties

	// Regular properties
	if catalogModel.Description != nil {
		properties = append(properties, models.NewStringProperty("description", *catalogModel.Description, false))
	}
	if catalogModel.Readme != nil {
		properties = append(properties, models.NewStringProperty("readme", *catalogModel.Readme, false))
	}
	if catalogModel.Provider != nil {
		properties = append(properties, models.NewStringProperty("provider", *catalogModel.Provider, false))
	}
	if catalogModel.License != nil {
		humanReadableLicense := basecatalog.TransformLicenseToHumanReadable(*catalogModel.License)
		properties = append(properties, models.NewStringProperty("license", humanReadableLicense, false))
	}
	if catalogModel.LicenseLink != nil {
		properties = append(properties, models.NewStringProperty("license_link", *catalogModel.LicenseLink, false))
	}
	if catalogModel.LibraryName != nil {
		properties = append(properties, models.NewStringProperty("library_name", *catalogModel.LibraryName, false))
	}
	if catalogModel.Logo != nil {
		properties = append(properties, models.NewStringProperty("logo", *catalogModel.Logo, false))
	}
	if catalogModel.SourceId != nil {
		properties = append(properties, models.NewStringProperty("source_id", *catalogModel.SourceId, false))
	}

	// Convert array properties
	if len(catalogModel.Tasks) > 0 {
		if tasksJSON, err := json.Marshal(catalogModel.Tasks); err == nil {
			properties = append(properties, models.NewStringProperty("tasks", string(tasksJSON), false))
		}
	}
	if len(catalogModel.Language) > 0 {
		if languageJSON, err := json.Marshal(catalogModel.Language); err == nil {
			properties = append(properties, models.NewStringProperty("language", string(languageJSON), false))
		}
	}

	// Convert custom properties from the CatalogModel
	if catalogModel.CustomProperties != nil {
		for key, value := range catalogModel.GetCustomProperties() {
			prop := convertMetadataValueToProperty(key, value)
			customProperties = append(customProperties, prop)
		}
	}

	return properties, customProperties
}

// parseHFTime parses Hugging Face timestamp format (ISO 8601)
func parseHFTime(timeStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func (p *hfModelProvider) emit(ctx context.Context, models []ModelProviderRecord, out chan<- ModelProviderRecord) {
	p.emitWithError(ctx, models, nil, out)
}

// emitWithError sends all successfully loadedmodels to the channel, then sends a final empty record.
// If err is non-nil, the final record will include the error to signal a partial failure
// (some models were loaded successfully, but others failed).
func (p *hfModelProvider) emitWithError(ctx context.Context, models []ModelProviderRecord, err error, out chan<- ModelProviderRecord) {
	done := ctx.Done()
	for _, model := range models {
		// Check if model should be excluded by name
		if model.Model.GetAttributes() != nil && model.Model.GetAttributes().Name != nil {
			modelName := *model.Model.GetAttributes().Name
			if !p.filter.Allows(modelName) {
				glog.V(2).Infof("Skipping excluded model in emit: %s", modelName)
				continue
			}
		}

		select {
		case out <- model:
		case <-done:
			return
		}
	}

	// Send an empty record to indicate that we're done with the batch.
	// Include any error to signal partial failure (models loaded, but some failed).
	select {
	case out <- ModelProviderRecord{Error: err}:
	case <-done:
	}
}

// validateCredentials checks if the Hugging Face API key credentials are valid
func (p *hfModelProvider) validateCredentials(ctx context.Context) error {
	glog.Infof("Validating Hugging Face API credentials")

	// Make a simple API call to validate credentials
	apiURL := p.baseURL + "/api/whoami-v2"
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("User-Agent", "model-registry-catalog")

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate Hugging Face credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid Hugging Face API credentials")
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Hugging Face API validation failed with status: %d: %s", resp.StatusCode, string(bodyBytes))
	}

	glog.Infof("Hugging Face credentials validated successfully")
	return nil
}

// sanitizeHFProperties validates security-sensitive properties and returns the env var name to
// use for the API key. The apiKeyEnvVar property is accepted only when it equals defaultAPIKeyEnvVar
// ("HF_API_KEY") or starts with apiKeyEnvVarPrefix ("HF_API_KEY_"); all other values are rejected
// and the default is used. Custom URLs are always rejected to prevent SSRF. In both cases the
// property is removed from the map to prevent it from leaking into downstream metadata.
func sanitizeHFProperties(props map[string]any, label string) string {
	envVarName := defaultAPIKeyEnvVar
	if envVar, ok := props[apiKeyEnvVarKey].(string); ok && envVar != "" {
		if envVar == defaultAPIKeyEnvVar || strings.HasPrefix(envVar, apiKeyEnvVarPrefix) {
			envVarName = envVar
		} else {
			glog.Warningf("%s: apiKeyEnvVar %q rejected; must be %q or start with %q — falling back to default",
				label, envVar, defaultAPIKeyEnvVar, apiKeyEnvVarPrefix)
		}
		delete(props, apiKeyEnvVarKey)
	}
	if url, ok := props[urlKey].(string); ok && url != "" {
		glog.Warningf("%s: custom URL %q ignored (SSRF prevention)", label, url)
		delete(props, urlKey)
	}
	return envVarName
}

func newHFModelProvider(ctx context.Context, source *basecatalog.ModelSource, reldir string) (<-chan ModelProviderRecord, error) {
	p := &hfModelProvider{}
	p.client = &http.Client{Timeout: 30 * time.Second}

	// Parse Source ID
	sourceId := source.GetId()
	if sourceId == "" {
		return nil, fmt.Errorf("missing source ID for Hugging Face catalog")
	}
	p.sourceId = sourceId

	// Reject custom URLs (SSRF prevention) and validate apiKeyEnvVar (must be HF_API_KEY or HF_API_KEY_*).
	apiKeyEnvVar := sanitizeHFProperties(source.Properties, "HuggingFace catalog")
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		glog.Infof("No API key configured for Hugging Face. Only public models and limited data for gated models will be available.")
	}
	p.apiKey = apiKey
	p.baseURL = defaultHuggingFaceURL

	allowedOrg, _ := source.Properties[allowedOrgKey].(string)
	restrictToOrg(allowedOrg, &source.IncludedModels, &source.ExcludedModels)

	// Parse sync interval (optional, defaults to 24 hours)
	// This can be configured as a duration string (e.g., "1s", "10s", "1m", "24h").
	// For testing, a shorter interval can be used to speed up tests.
	p.syncInterval = defaultSyncInterval
	if syncInterval, ok := source.Properties[syncIntervalKey].(string); ok && syncInterval != "" {
		if parsed, err := time.ParseDuration(syncInterval); err == nil {
			p.syncInterval = parsed
		} else {
			glog.Warningf("Invalid syncInterval duration string %q, using default: %v", syncInterval, err)
		}
	}

	if p.apiKey != "" {
		hasValidPrefix := strings.HasPrefix(p.apiKey, "hf_")
		if !hasValidPrefix {
			// API key is set but doesn't have expected prefix, warn and continue without authentication
			glog.Infof("API key does not have expected 'hf_' prefix. Only public models and limited data for gated models will be available.")
			p.apiKey = "" // Clear invalid key to prevent its use
		}
		if hasValidPrefix {
			// Validate credentials only if API key has correct format
			if err := p.validateCredentials(ctx); err != nil {
				glog.Errorf("Hugging Face catalog credential validation failed: %v", err)
				return nil, fmt.Errorf("failed to validate Hugging Face catalog credentials: %w", err)
			}
		}
	}

	// Use top-level IncludedModels from Source as the list of models to fetch
	// These can be specific model names (required for HF API) or patterns
	if len(source.IncludedModels) == 0 {
		return nil, fmt.Errorf("includedModels cannot be empty for Hugging Face catalog")
	}

	p.includedModels = source.IncludedModels

	// Create ModelFilter from source configuration (handles IncludedModels/ExcludedModels from Source)
	// Note: IncludedModels are used both for fetching and filtering
	filter, err := NewModelFilterFromSource(source, nil, nil)
	if err != nil {
		return nil, err
	}
	p.filter = filter

	return p.Models(ctx)
}

func init() {
	if err := RegisterModelProvider("hf", newHFModelProvider); err != nil {
		panic(err)
	}
}

// NewHFPreviewProvider creates an hfModelProvider for preview use.
// It initializes the provider from a PreviewConfig without starting the full model loading.
func NewHFPreviewProvider(config *PreviewConfig) (*hfModelProvider, error) {
	p := &hfModelProvider{
		client:       &http.Client{Timeout: 30 * time.Second},
		baseURL:      defaultHuggingFaceURL,
		maxModels:    defaultMaxModels,
		syncInterval: defaultSyncInterval,
	}

	// Reject custom URLs (SSRF prevention) and validate apiKeyEnvVar (must be HF_API_KEY or HF_API_KEY_*).
	apiKeyEnvVar := sanitizeHFProperties(config.Properties, "HuggingFace preview")
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		glog.Infof("No API key configured for Hugging Face preview. Only public models and limited data for gated models will be available.")
	}
	p.apiKey = apiKey

	allowedOrg, _ := config.Properties[allowedOrgKey].(string)
	restrictToOrg(allowedOrg, &config.IncludedModels, &config.ExcludedModels)

	if len(config.IncludedModels) == 0 {
		return nil, fmt.Errorf("includedModels is required for HuggingFace source preview (specifies which models to fetch from HuggingFace)")
	}

	// Parse maxModels limit (optional, defaults to 500)
	// This limit is applied PER PATTERN (e.g., each "org/*" pattern gets its own limit)
	// to prevent overloading the Hugging Face API and respect rate limiting.
	// Set to 0 to disable the limit.
	if maxModels, ok := config.Properties[maxModelsKey]; ok {
		switch v := maxModels.(type) {
		case int:
			p.maxModels = v
		case int64:
			p.maxModels = int(v)
		case float64:
			p.maxModels = int(v)
		}
	}

	return p, nil
}

// hfListResponse represents a single model in the Hugging Face list API response.
type hfListModel struct {
	ID        string `json:"id"`
	ModelID   string `json:"modelId,omitempty"`
	Author    string `json:"author,omitempty"`
	Private   bool   `json:"private,omitempty"`
	Downloads int    `json:"downloads,omitempty"`
}

// PatternType indicates how to handle an includedModels pattern.
type PatternType int

const (
	PatternExact     PatternType = iota // e.g., "org/model-name" - direct fetch
	PatternOrgAll                       // e.g., "org/*" - list all from org
	PatternOrgPrefix                    // e.g., "org/prefix*" - list from org with search
	PatternInvalid                      // e.g., "*", "*/*" - not supported
)

// parseModelPattern analyzes a model identifier to determine how to fetch it.
// Returns: patternType, org, searchPrefix
func parseModelPattern(pattern string) (PatternType, string, string) {
	pattern = strings.TrimSpace(pattern)

	// Reject unsupported wildcard patterns that would try to list all Hugging Face models
	// Hugging Face has millions of models, so we require a specific organization
	if pattern == "*" || pattern == "*/*" {
		return PatternInvalid, "", ""
	}

	// Reject patterns like "*/something" where org is a wildcard
	if strings.HasPrefix(pattern, "*/") {
		return PatternInvalid, "", ""
	}

	// Check if it's an org/* pattern
	if org, ok := strings.CutSuffix(pattern, "/*"); ok {
		// Ensure org is not empty or just whitespace
		if org == "" || strings.TrimSpace(org) == "" {
			return PatternInvalid, "", ""
		}
		return PatternOrgAll, org, ""
	}

	parts := strings.SplitN(pattern, "/", 2)

	org := parts[0]
	// Ensure org is not empty or a wildcard
	if org == "" || strings.Contains(org, "*") {
		return PatternInvalid, "", ""
	}

	var model string
	if len(parts) == 2 {
		model = parts[1]
		if model == "" {
			return PatternInvalid, "", ""
		}
	}

	// Check if it has a wildcard after org/prefix
	if prefix, ok := strings.CutSuffix(model, "*"); ok {
		if prefix != "" {
			return PatternOrgPrefix, org, prefix
		}
	}

	// Exact model name
	return PatternExact, "", ""
}

// listModelsByAuthor fetches all models from an organization using the Hugging Face list API with pagination.
// If searchPrefix is provided, it filters models that start with that prefix.
func (p *hfModelProvider) listModelsByAuthor(ctx context.Context, author string, searchPrefix string) ([]string, error) {
	var allModels []string
	limit := 100 // Max allowed by HF API
	cursor := ""

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check if we've reached the maxModels limit for this pattern
		// (maxModels is applied per-pattern to respect HF API rate limits)
		if p.maxModels > 0 && len(allModels) >= p.maxModels {
			glog.Warningf("Reached maxModels limit (%d) for pattern author=%s, stopping pagination", p.maxModels, author)
			break
		}

		// Build API URL
		apiURL := fmt.Sprintf("%s/api/models?author=%s&limit=%d", p.baseURL, author, limit)
		if searchPrefix != "" {
			apiURL += "&search=" + searchPrefix
		}
		if cursor != "" {
			apiURL += "&cursor=" + cursor
		}

		glog.V(2).Infof("Fetching Hugging Face models list: %s", apiURL)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create list request: %w", err)
		}

		req.Header.Set("User-Agent", "model-registry-catalog")
		if p.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.apiKey)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list models for author %s: %w", author, err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Hugging Face API returned status %d for author %s: %s", resp.StatusCode, author, string(bodyBytes))
		}

		var models []hfListModel
		if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode models list for author %s: %w", author, err)
		}
		resp.Body.Close()

		// Extract model IDs
		for _, m := range models {
			// Check limit before adding each model
			if p.maxModels > 0 && len(allModels) >= p.maxModels {
				break
			}

			modelID := m.ID
			if modelID == "" {
				modelID = m.ModelID
			}
			if modelID == "" {
				continue
			}

			// If we have a search prefix, double-check it matches
			// (HF search is fuzzy, so we need to verify)
			if searchPrefix != "" {
				// Extract the model name part (after org/)
				parts := strings.SplitN(modelID, "/", 2)
				if len(parts) == 2 {
					modelName := parts[1]
					if !strings.HasPrefix(strings.ToLower(modelName), strings.ToLower(searchPrefix)) {
						continue
					}
				}
			}

			allModels = append(allModels, modelID)
		}

		// Check for next page via Link header
		linkHeader := resp.Header.Get("Link")
		nextCursor := parseNextCursor(linkHeader)
		if nextCursor == "" || len(models) < limit {
			// No more pages
			break
		}
		cursor = nextCursor
	}

	glog.Infof("Listed %d models from author %s (maxModels: %d)", len(allModels), author, p.maxModels)
	return allModels, nil
}

// parseNextCursor extracts the cursor for the next page from the Link header.
// Link header format: <url>; rel="next"
func parseNextCursor(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}

	// Parse Link header for rel="next"
	for link := range strings.SplitSeq(linkHeader, ",") {
		link = strings.TrimSpace(link)
		if strings.Contains(link, `rel="next"`) {
			// Extract URL between < and >
			start := strings.Index(link, "<")
			end := strings.Index(link, ">")
			if start >= 0 && end > start {
				nextURL := link[start+1 : end]
				// Extract cursor parameter from URL
				if _, cursor, ok := strings.Cut(nextURL, "cursor="); ok {
					// Handle if there are more parameters after cursor
					if ampIdx := strings.Index(cursor, "&"); ampIdx >= 0 {
						cursor = cursor[:ampIdx]
					}
					return cursor
				}
			}
		}
	}
	return ""
}

// FetchModelNamesForPreview fetches model info from Hugging Face API for the given model identifiers
// and returns the actual model names. This is used for preview functionality.
// Supports patterns like "org/*" and "org/prefix*" which use the paginated list API.
func (p *hfModelProvider) FetchModelNamesForPreview(ctx context.Context, modelIdentifiers []string) ([]string, error) {
	if len(modelIdentifiers) == 0 {
		return nil, fmt.Errorf("includedModels is required for Hugging Face source preview")
	}

	// Validate credentials only if API key is provided
	if p.apiKey != "" {
		if err := p.validateCredentials(ctx); err != nil {
			return nil, fmt.Errorf("failed to validate HuggingFace credentials: %w", err)
		}
	}

	names := make([]string, 0)

	for _, pattern := range modelIdentifiers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		patternType, org, searchPrefix := parseModelPattern(pattern)

		switch patternType {
		case PatternInvalid:
			// Reject unsupported wildcard patterns
			return nil, fmt.Errorf("wildcard pattern %q is not supported - Hugging Face requires a specific organization (e.g., 'ibm-granite/*' or 'meta-llama/Llama-2-*')", pattern)

		case PatternOrgAll, PatternOrgPrefix:
			// Use paginated list API
			glog.Infof("Using Hugging Face list API for pattern: %s (org=%s, prefix=%s)", pattern, org, searchPrefix)
			models, err := p.listModelsByAuthor(ctx, org, searchPrefix)
			if err != nil {
				glog.Warningf("Failed to list models for pattern %s: %v", pattern, err)
				// Don't fail completely, just skip this pattern
				continue
			}
			names = append(names, models...)

		case PatternExact:
			// Direct fetch for exact model name
			modelInfo, err := p.fetchModelInfo(ctx, pattern)
			if err != nil {
				glog.Warningf("Failed to fetch model info for preview: %s: %v", pattern, err)
				names = append(names, pattern)
				continue
			}

			actualName := modelInfo.ID
			if actualName == "" {
				actualName = pattern
			}
			names = append(names, actualName)
		}
	}

	return names, nil
}

// restrictToOrg prefixes included and excluded model lists with an
// organization name for convenience and to prevent any other organization from
// being retrieved.
func restrictToOrg(org string, included *[]string, excluded *[]string) {
	if org == "" {
		// No op
		return
	}

	prefix := org + "/"

	if included == nil || len(*included) == 0 {
		*included = []string{prefix + "*"}
	} else {
		for i := range *included {
			(*included)[i] = prefix + (*included)[i]
		}
	}

	if excluded != nil {
		for i := range *excluded {
			(*excluded)[i] = prefix + (*excluded)[i]
		}
	}
}
