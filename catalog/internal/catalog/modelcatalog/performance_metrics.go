package modelcatalog

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	dbmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	models "github.com/kubeflow/hub/internal/platform/db/entity"
)

// metadataJSON represents the minimal structure needed from metadata.json files
// Only the ID field is needed to look up existing models
type metadataJSON struct {
	ID                string   `json:"id"`                  // Maps to model name for lookup
	OverallAccuracy   *float64 `json:"overall_accuracy"`    // Overall accuracy score for the model
	Size              *string  `json:"size"`                // Model parameter count (e.g., "8B params")
	TensorType        *string  `json:"tensor_type"`         // Data precision (e.g., "FP16", "INT4")
	VariantGroupID    *string  `json:"variant_group_id"`    // UUID linking model variants together
	MinVRAMGB         *float64 `json:"min_vram_gb"`         // Minimum VRAM required in GB (e.g., 466.0)
	ModelcarImageSize *float64 `json:"modelcar_image_size"` // Modelcar image size in GB (e.g., 405.19)
}

// parseMetadataJSON parses JSON data into metadataJSON struct, extracting only the ID field
func parseMetadataJSON(data []byte) (metadataJSON, error) {
	var metadata metadataJSON
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadataJSON{}, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if metadata.ID == "" {
		return metadataJSON{}, fmt.Errorf("missing required 'id' field in metadata")
	}

	return metadata, nil
}

// evaluationRecord represents a single evaluation result from evaluations.ndjson.
// Records contribute to the aggregate accuracy-metrics artifact, and complete
// records are also retained as individual evaluation-metrics artifacts.
type evaluationRecord struct {
	// Core fields needed to associate evaluation with model
	ID           string   `json:"id"`
	ModelID      string   `json:"model_id"`
	RunID        string   `json:"run_id"`
	Evaluation   string   `json:"evaluation"`
	Category     string   `json:"category"`
	Benchmark    string   `json:"benchmark"`
	Description  string   `json:"description"`
	Result       *float64 `json:"result"`
	ResultMetric string   `json:"result_metric"`
	CreatedAt    *int64   `json:"created_at"`
	UpdatedAt    *int64   `json:"updated_at"`

	// CustomProperties captures all other fields dynamically
	CustomProperties map[string]any `json:"-"`
}

var evaluationCategoryPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// UnmarshalJSON implements custom JSON unmarshaling to capture all undefined fields as CustomProperties
func (er *evaluationRecord) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map to get all fields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract the core fields
	if id, ok := raw["id"].(string); ok {
		er.ID = id
	}
	if modelID, ok := raw["model_id"].(string); ok {
		er.ModelID = modelID
	}
	if runID, ok := raw["run_id"].(string); ok {
		er.RunID = runID
	}
	if evaluation, ok := raw["evaluation"].(string); ok {
		er.Evaluation = evaluation
	}
	if category, ok := raw["category"].(string); ok {
		er.Category = category
	}
	if benchmark, ok := raw["benchmark"].(string); ok {
		er.Benchmark = benchmark
	}
	if description, ok := raw["description"].(string); ok {
		er.Description = description
	}
	if result, ok := raw["result"].(float64); ok {
		er.Result = &result
	}
	if resultMetric, ok := raw["result_metric"].(string); ok {
		er.ResultMetric = resultMetric
	}
	if createdAt, ok := timestampFromJSONValue(raw["created_at"]); ok {
		er.CreatedAt = &createdAt
	}
	if updatedAt, ok := timestampFromJSONValue(raw["updated_at"]); ok {
		er.UpdatedAt = &updatedAt
	}

	// Initialize CustomProperties if nil
	if er.CustomProperties == nil {
		er.CustomProperties = make(map[string]any)
	}

	// Copy all fields to CustomProperties, including the core ones
	maps.Copy(er.CustomProperties, raw)

	return nil
}

// timestampFromJSONValue converts an integer-valued JSON number to an epoch
// timestamp without accepting fractional or negative values.
func timestampFromJSONValue(value any) (int64, bool) {
	number, ok := value.(float64)
	if !ok || number < 0 || number != float64(int64(number)) {
		return 0, false
	}
	return int64(number), true
}

// validateDetailedEvaluation ensures a record can safely be exposed as an
// evaluation-metrics artifact. Legacy evaluation rows may still contribute to
// the aggregate accuracy artifact, but are not exposed as incomplete details.
func (er evaluationRecord) validateDetailedEvaluation(expectedModelID string) error {
	requiredStrings := []struct {
		name  string
		value string
	}{
		{name: "id", value: er.ID},
		{name: "model_id", value: er.ModelID},
		{name: "run_id", value: er.RunID},
		{name: "evaluation", value: er.Evaluation},
		{name: "category", value: er.Category},
		{name: "benchmark", value: er.Benchmark},
		{name: "description", value: er.Description},
		{name: "result_metric", value: er.ResultMetric},
	}
	for _, field := range requiredStrings {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("missing required %s", field.name)
		}
	}
	if er.ModelID != expectedModelID {
		return fmt.Errorf("model_id %q does not match metadata id %q", er.ModelID, expectedModelID)
	}
	if er.Result == nil {
		return fmt.Errorf("missing or non-numeric required result")
	}
	if !evaluationCategoryPattern.MatchString(er.Category) {
		return fmt.Errorf("category %q is not lowercase kebab-case", er.Category)
	}
	if er.CreatedAt == nil {
		return fmt.Errorf("missing or invalid required created_at")
	}
	if _, present := er.CustomProperties["updated_at"]; present && er.UpdatedAt == nil {
		return fmt.Errorf("invalid updated_at")
	}
	if er.UpdatedAt != nil && *er.UpdatedAt < *er.CreatedAt {
		return fmt.Errorf("updated_at precedes created_at")
	}
	return nil
}

func filterEvaluationRecordsForModel(records []evaluationRecord, expectedModelID string) []evaluationRecord {
	filtered := make([]evaluationRecord, 0, len(records))
	for _, record := range records {
		if record.ModelID != "" && record.ModelID != expectedModelID {
			glog.Warningf("Evaluation record %q has model_id %q, expected %q; skipping", record.ID, record.ModelID, expectedModelID)
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered
}

// performanceRecord represents a single performance result from performance.ndjson
// Only minimal fields needed for association are explicitly defined
type performanceRecord struct {
	// Core fields needed to associate performance data with model
	ID      string `json:"id"`
	ModelID string `json:"model_id"`

	// CustomProperties captures remaining fields dynamically
	CustomProperties map[string]any `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling to capture all undefined fields as CustomProperties
func (pr *performanceRecord) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map to get all fields
	var raw map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	// Extract the core fields
	if id, ok := raw["id"].(string); ok {
		pr.ID = id
	}
	if pr.ID == "" {
		if configID, ok := raw["config_id"].(string); ok && configID != "" {
			pr.ID = configID
			// Expose the resolved producer ID even for config-only records.
			raw["id"] = configID
		}
	}
	if modelID, ok := raw["model_id"].(string); ok {
		pr.ModelID = modelID
	}

	// Initialize CustomProperties if nil
	if pr.CustomProperties == nil {
		pr.CustomProperties = make(map[string]any)
	}

	// Copy all fields to CustomProperties, including the core ones
	maps.Copy(pr.CustomProperties, raw)

	return nil
}

// securityEvaluationRecord represents a single security evaluation result from security-evaluations.ndjson
type securityEvaluationRecord struct {
	// Core fields needed to associate security data with model
	ID      string `json:"id"`
	ModelID string `json:"model_id"`

	// CustomProperties captures remaining fields dynamically
	CustomProperties map[string]any `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling to capture all undefined fields as CustomProperties
func (sr *securityEvaluationRecord) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	if id, ok := raw["id"].(string); ok {
		sr.ID = id
	}
	if modelID, ok := raw["model_id"].(string); ok {
		sr.ModelID = modelID
	}

	if sr.CustomProperties == nil {
		sr.CustomProperties = make(map[string]any)
	}

	maps.Copy(sr.CustomProperties, raw)

	return nil
}

type PerformanceMetricsLoader struct {
	path                  []string
	modelRepo             dbmodels.CatalogModelRepository
	metricsArtifactRepo   dbmodels.CatalogMetricsArtifactRepository
	modelTypeID           int32
	metricsArtifactTypeID int32
	// Cache of lowercase model ID -> directory path for case-insensitive lookups
	modelDirCache map[string]string
}

// UpdateRepos replaces the loader's cached repository references and type IDs
// after a database reconnect. Safe to call from the OnBecomeLeader callback:
// the elector drains all previous leader callbacks before invoking a new one,
// so this always runs before NotifyLeader starts any leader-mode loading.
func (pml *PerformanceMetricsLoader) UpdateRepos(modelRepo dbmodels.CatalogModelRepository, metricsArtifactRepo dbmodels.CatalogMetricsArtifactRepository, typeMap map[string]int32) error {
	modelTypeID, exists := typeMap[service.CatalogModelTypeName]
	if !exists {
		return fmt.Errorf("CatalogModel type not found in type map")
	}
	metricsArtifactTypeID, exists := typeMap[service.CatalogMetricsArtifactTypeName]
	if !exists {
		return fmt.Errorf("CatalogMetricsArtifact type not found in type map")
	}
	pml.modelRepo = modelRepo
	pml.metricsArtifactRepo = metricsArtifactRepo
	pml.modelTypeID = modelTypeID
	pml.metricsArtifactTypeID = metricsArtifactTypeID
	return nil
}

func NewPerformanceMetricsLoader(path []string, modelRepo dbmodels.CatalogModelRepository, metricsArtifactRepo dbmodels.CatalogMetricsArtifactRepository, typeMap map[string]int32) (*PerformanceMetricsLoader, error) {
	if len(path) == 0 {
		glog.Info("No performance metrics path provided, skipping performance metrics loading")
		return nil, nil
	}

	// Check if path exists
	for _, p := range path {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			glog.Warningf("Performance metrics path %s does not exist, skipping performance metrics loading", p)
			return nil, nil
		}
	}

	glog.Infof("Loading performance metrics data from %s", path)

	// Get the TypeID for CatalogModel from the type map
	modelTypeID, exists := typeMap[service.CatalogModelTypeName]
	if !exists {
		return nil, fmt.Errorf("CatalogModel type not found in type map")
	}
	glog.V(2).Infof("Using catalog model type ID: %d", modelTypeID)

	// Get the TypeID for CatalogMetricsArtifact from the type map
	metricsArtifactTypeID, exists := typeMap[service.CatalogMetricsArtifactTypeName]
	if !exists {
		return nil, fmt.Errorf("CatalogMetricsArtifact type not found in type map")
	}
	glog.V(2).Infof("Using metrics artifact type ID: %d", metricsArtifactTypeID)

	loader := &PerformanceMetricsLoader{
		path:                  path,
		modelRepo:             modelRepo,
		metricsArtifactRepo:   metricsArtifactRepo,
		modelTypeID:           modelTypeID,
		metricsArtifactTypeID: metricsArtifactTypeID,
		modelDirCache:         make(map[string]string),
	}

	// Build the model directory cache once during initialization
	if err := loader.buildModelDirCache(); err != nil {
		return nil, fmt.Errorf("failed to build model directory cache: %v", err)
	}

	return loader, nil
}

// buildModelDirCache scans directories once and builds a cache of lowercase model ID -> directory path
func (pml *PerformanceMetricsLoader) buildModelDirCache() error {
	modelCount := 0
	for _, rootPath := range pml.path {
		err := filepath.Walk(rootPath, func(dirPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip if not a directory
			if !info.IsDir() {
				return nil
			}

			// Check if this directory contains metadata.json
			metadataPath := filepath.Join(dirPath, "metadata.json")
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				return nil // Skip directories without metadata.json
			}

			// Read and parse metadata.json to extract the model ID
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				glog.Warningf("Failed to read metadata file %s: %v", metadataPath, err)
				return nil // Continue with other directories
			}

			// Parse metadata to extract the model ID for lookup
			metadata, err := parseMetadataJSON(metadataData)
			if err != nil {
				glog.Warningf("Failed to parse metadata file %s: %v", metadataPath, err)
				return nil // Continue with other directories
			}

			// Add to cache (lowercase key for case-insensitive matching)
			cacheKey := strings.ToLower(metadata.ID)
			if existing, ok := pml.modelDirCache[cacheKey]; ok {
				glog.Warningf("Case-insensitive cache collision: %q (dir: %s) overwrites existing entry (dir: %s)", metadata.ID, dirPath, existing)
			} else {
				modelCount++
			}
			pml.modelDirCache[cacheKey] = dirPath
			glog.V(3).Infof("Cached model directory: %s -> %s", metadata.ID, dirPath)

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %v", rootPath, err)
		}
	}

	glog.Infof("Built model directory cache (%d models indexed)", modelCount)

	return nil
}

func (pml *PerformanceMetricsLoader) Load(ctx context.Context, record ModelProviderRecord) error {
	if pml == nil {
		return nil
	}

	attrs := record.Model.GetAttributes()
	if attrs == nil || attrs.Name == nil {
		return nil
	}

	// Namespaced name is source_id:model_name
	namespacedName := *attrs.Name
	// Resolve directory from cache: cache is keyed by lowercase metadata.ID (display name)
	displayName := strings.ToLower(DisplayNameFromStoredName(namespacedName))
	dirPath, found := pml.modelDirCache[displayName]
	if !found {
		glog.V(2).Infof("No performance metrics directory found for model %s", namespacedName)
		return nil
	}

	glog.V(2).Infof("Found cached directory for model %s: %s", namespacedName, dirPath)

	// Process this specific model directory using the cached path (use namespaced name for DB lookup)
	artifactsCreated, err := processModelDirectory(dirPath, pml.modelRepo, pml.metricsArtifactRepo, pml.modelTypeID, pml.metricsArtifactTypeID, namespacedName)
	if err != nil {
		return fmt.Errorf("failed to process metrics for model %s: %v", namespacedName, err)
	}

	if artifactsCreated > 0 {
		glog.Infof("Loaded %d performance metrics artifacts for model %s", artifactsCreated, namespacedName)
	}

	return nil
}

// processModelDirectory processes a single model directory containing metadata.json and metric files
// Only processes metrics for models that already exist in the database.
// namespacedModelName is the stored model name (sourceId:modelName) used for GetByName lookup.
func processModelDirectory(dirPath string, modelRepo dbmodels.CatalogModelRepository, metricsArtifactRepo dbmodels.CatalogMetricsArtifactRepository, modelTypeID int32, metricsArtifactTypeID int32, namespacedModelName string) (int, error) {
	// Read and parse metadata.json to extract the model ID
	metadataPath := filepath.Join(dirPath, "metadata.json")
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read metadata file %s: %v", metadataPath, err)
	}

	// Parse metadata to extract the model ID for lookup
	metadata, err := parseMetadataJSON(metadataData)
	if err != nil {
		return 0, fmt.Errorf("failed to parse metadata file %s: %v", metadataPath, err)
	}

	// Check if the model already exists - only process metrics for existing models (look up by namespaced name)
	existingModel, err := modelRepo.GetByName(namespacedModelName)
	if err != nil {
		return 0, fmt.Errorf("failed to check for existing model: %v", err)
	}

	// Skip processing if model doesn't exist
	if existingModel == nil {
		glog.V(2).Infof("Model %s does not exist in database, skipping metrics processing", namespacedModelName)
		return 0, nil
	}

	// Enrich the model with metadata before processing metrics artifacts
	if err := enrichCatalogModelFromMetadata(existingModel, metadata, modelRepo); err != nil {
		glog.Warningf("Failed to enrich model %s with metadata: %v", namespacedModelName, err)
		// Continue processing - don't fail the whole operation
	}

	modelID := *existingModel.GetID()
	glog.V(2).Infof("Found existing model %s with ID %d, processing metrics", namespacedModelName, modelID)

	// Use batch processing for all artifacts
	return processModelArtifactsBatch(dirPath, modelID, metadata.ID, metadata.OverallAccuracy, metricsArtifactRepo, metricsArtifactTypeID)
}

// processModelArtifactsBatch processes all metric artifacts for a model in batch
// This reduces DB overhead by parsing, checking, and inserting in optimized phases
func processModelArtifactsBatch(dirPath string, modelID int32, modelName string, overallAccuracy *float64, metricsArtifactRepo dbmodels.CatalogMetricsArtifactRepository, metricsArtifactTypeID int32) (int, error) {
	// Parse all metrics files
	var evaluationRecords []evaluationRecord
	var performanceRecords []performanceRecord
	var securityRecords []securityEvaluationRecord

	// Parse evaluation metrics if file exists
	evaluationsPath := filepath.Join(dirPath, "evaluations.ndjson")
	if _, err := os.Stat(evaluationsPath); err == nil {
		records, err := parseEvaluationFile(evaluationsPath)
		if err != nil {
			glog.Errorf("Failed to parse evaluations file for %s: %v", modelName, err)
		} else {
			// Legacy records without model_id are accepted for the aggregate
			// accuracy artifact. Explicitly mismatched records are rejected so
			// data from one model can never be attributed to another model.
			evaluationRecords = filterEvaluationRecordsForModel(records, modelName)
		}
	}

	// Parse performance metrics if file exists
	performancePath := filepath.Join(dirPath, "performance.ndjson")
	if _, err := os.Stat(performancePath); err == nil {
		records, err := parsePerformanceFile(performancePath)
		if err != nil {
			glog.Errorf("Failed to parse performance file for %s: %v", modelName, err)
		} else {
			performanceRecords = records
		}
	}

	// Parse security evaluation metrics if file exists
	securityPath := filepath.Join(dirPath, "security-evaluations.ndjson")
	if _, err := os.Stat(securityPath); err == nil {
		records, err := parseSecurityEvaluationFile(securityPath)
		if err != nil {
			glog.Errorf("Failed to parse security evaluations file for %s: %v", modelName, err)
		} else {
			securityRecords = records
		}
	}

	totalRecords := len(evaluationRecords) + len(performanceRecords) + len(securityRecords)
	if totalRecords == 0 {
		return 0, nil
	}

	// Bulk load all existing artifacts for this model and check in-memory
	// Single DB query to get ALL existing artifacts for this model
	existingArtifactsList, err := metricsArtifactRepo.List(dbmodels.CatalogMetricsArtifactListOptions{
		ParentResourceID: &modelID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to load existing artifacts for model: %v", err)
	}

	// Include the metrics type so a legacy raw ID never suppresses another kind.
	type artifactKey struct {
		metricsType dbmodels.MetricsType
		externalID  string
	}
	existingArtifactsMap := make(map[artifactKey]bool, existingArtifactsList.Size)
	legacyArtifactsMap := make(map[artifactKey]bool)
	for _, artifact := range existingArtifactsList.Items {
		if attrs := artifact.GetAttributes(); attrs != nil && attrs.ExternalID != nil {
			key := artifactKey{attrs.MetricsType, *attrs.ExternalID}
			// Scoped records use the same name and external ID. Legacy record
			// names have a kind prefix, unlike their raw external IDs. Never
			// mistake a scoped identity for a different record's producer ID.
			if attrs.Name == nil || *attrs.Name != *attrs.ExternalID {
				legacyArtifactsMap[key] = true
			} else {
				existingArtifactsMap[key] = true
			}
		}
	}

	// The same identity is used for within-file deduplication and DB lookups.
	// Legacy rows are recognized only within this model and metrics type until
	// the normal model reload replaces them with scoped identities.
	seenIDs := make(map[string]bool, totalRecords)
	shouldInsert := func(metricsType dbmodels.MetricsType, producerID string) bool {
		externalID := metricsArtifactIdentity(modelID, metricsType, producerID)
		if seenIDs[externalID] {
			glog.Warningf("Duplicate %s artifact ID %s in file, skipping", metricsType, producerID)
			return false
		}
		seenIDs[externalID] = true
		if existingArtifactsMap[artifactKey{metricsType, externalID}] || legacyArtifactsMap[artifactKey{metricsType, producerID}] {
			glog.V(2).Infof("%s artifact %s already exists, skipping", metricsType, producerID)
			return false
		}
		return true
	}

	// Check which artifacts need to be created using the in-memory map
	artifactsToInsert := make([]*dbmodels.CatalogMetricsArtifactImpl, 0, totalRecords+len(evaluationRecords))

	// Check evaluation artifacts
	if len(evaluationRecords) > 0 {
		externalID := fmt.Sprintf("accuracy-metrics-model-%d", modelID)
		if !existingArtifactsMap[artifactKey{dbmodels.MetricsTypeAccuracy, externalID}] {
			artifact := createAccuracyMetricsArtifact(evaluationRecords, modelID, metricsArtifactTypeID, overallAccuracy, nil, nil)
			artifactsToInsert = append(artifactsToInsert, artifact)
		} else {
			glog.V(2).Infof("Accuracy metrics artifact already exists, skipping")
		}
	}

	// Preserve complete evaluation rows as individual artifacts for the
	// Evaluation Insights table and run history. The aggregate accuracy artifact
	// above remains unchanged for existing consumers.
	for _, evalRecord := range evaluationRecords {
		if err := evalRecord.validateDetailedEvaluation(modelName); err != nil {
			glog.Warningf("Evaluation record %q is not eligible for detailed serving: %v", evalRecord.ID, err)
			continue
		}
		if shouldInsert(dbmodels.MetricsTypeEvaluation, evalRecord.ID) {
			artifact := createEvaluationArtifact(evalRecord, modelID, metricsArtifactTypeID)
			artifactsToInsert = append(artifactsToInsert, artifact)
		}
	}

	// Check performance artifacts
	for _, perfRecord := range performanceRecords {
		if shouldInsert(dbmodels.MetricsTypePerformance, perfRecord.ID) {
			artifact := createPerformanceArtifact(perfRecord, modelID, metricsArtifactTypeID, nil, nil)
			artifactsToInsert = append(artifactsToInsert, artifact)
		}
	}

	// Check security evaluation artifacts
	for _, secRecord := range securityRecords {
		if shouldInsert(dbmodels.MetricsTypeSecurityMetrics, secRecord.ID) {
			artifact := createSecurityArtifact(secRecord, modelID, metricsArtifactTypeID, nil, nil)
			artifactsToInsert = append(artifactsToInsert, artifact)
		}
	}

	if len(artifactsToInsert) == 0 {
		glog.V(2).Infof("All artifacts already exist for model %s, nothing to insert", modelName)
		return 0, nil
	}

	// Batch insert all new artifacts using BatchSave
	// Convert to slice of interface type for BatchSave
	artifactsToSave := make([]dbmodels.CatalogMetricsArtifact, len(artifactsToInsert))
	for i, artifact := range artifactsToInsert {
		artifactsToSave[i] = artifact
	}

	savedArtifacts, err := metricsArtifactRepo.BatchSave(artifactsToSave, &modelID)
	if err != nil {
		return 0, fmt.Errorf("failed to batch save artifacts: %v", err)
	}

	return len(savedArtifacts), nil
}

// parseEvaluationFile reads and parses an evaluations.ndjson file
func parseEvaluationFile(filePath string) ([]evaluationRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open evaluation file %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	evaluationRecords := []evaluationRecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var evalRecord evaluationRecord
		if err := json.Unmarshal([]byte(line), &evalRecord); err != nil {
			glog.Errorf("Failed to parse evaluation record: %v", err)
			continue
		}

		evaluationRecords = append(evaluationRecords, evalRecord)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading evaluation file: %v", err)
	}

	return evaluationRecords, nil
}

// parsePerformanceFile reads and parses a performance.ndjson file
func parsePerformanceFile(filePath string) ([]performanceRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open performance file %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	performanceRecords := []performanceRecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var perfRecord performanceRecord
		if err := json.Unmarshal([]byte(line), &perfRecord); err != nil {
			glog.Errorf("Failed to parse performance record: %v", err)
			continue
		}

		performanceRecords = append(performanceRecords, perfRecord)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading performance file: %v", err)
	}

	return performanceRecords, nil
}

// createAccuracyMetricsArtifact creates a single metrics artifact from all evaluation records
func createAccuracyMetricsArtifact(evalRecords []evaluationRecord, modelID int32, typeID int32, overallAccuracy *float64, existingID *int32, existingCreateTime *int64) *dbmodels.CatalogMetricsArtifactImpl {
	artifactName := fmt.Sprintf("accuracy-metrics-model-%d", modelID)
	externalID := fmt.Sprintf("accuracy-metrics-model-%d", modelID)

	// Use existing create time if provided, otherwise find from evaluation records
	createTime := existingCreateTime
	var updateTime *int64

	for _, evalRecord := range evalRecords {
		if existingCreateTime == nil {
			if createdAtFloat, ok := evalRecord.CustomProperties["created_at"].(float64); ok {
				createdAt := int64(createdAtFloat)
				if createTime == nil || createdAt < *createTime {
					createTime = &createdAt
				}
			}
		}
		if updatedAtFloat, ok := evalRecord.CustomProperties["updated_at"].(float64); ok {
			updatedAt := int64(updatedAtFloat)
			if updateTime == nil || updatedAt > *updateTime {
				updateTime = &updatedAt
			}
		}
		delete(evalRecord.CustomProperties, "updated_at")
		delete(evalRecord.CustomProperties, "created_at")
	}

	// Properties can be empty or contain general metadata
	properties := []models.Properties{}

	// Create custom properties - simple mapping of benchmark_name to score_value.
	// Deduplicate by benchmark name: if multiple evaluation records share the same
	// benchmark, keep the last score encountered. This prevents DB constraint violations
	// on the (artifact_id, name, is_custom_property) composite primary key.
	benchmarkScores := make(map[string]float64, len(evalRecords))
	for _, evalRecord := range evalRecords {
		score, ok := evalRecord.CustomProperties["score"].(float64)
		if !ok && evalRecord.Result != nil {
			score = *evalRecord.Result
			ok = true
		}
		if ok {
			if _, duplicate := benchmarkScores[evalRecord.Benchmark]; duplicate {
				glog.Warningf("Duplicate benchmark %q for model %d, using latest score", evalRecord.Benchmark, modelID)
			}
			benchmarkScores[evalRecord.Benchmark] = score
		}
	}

	customProperties := make([]models.Properties, 0, len(benchmarkScores)+1)
	for benchmark, score := range benchmarkScores {
		customProperties = append(customProperties, models.Properties{
			Name:        benchmark,
			DoubleValue: &score,
		})
	}

	// Add overall_average custom property from metadata.json overall_accuracy field
	if overallAccuracy != nil {
		customProperties = append(customProperties, models.Properties{
			Name:        "overall_average",
			DoubleValue: overallAccuracy,
		})
	}

	// Create the metrics artifact with metricsType set to accuracy-metrics
	metricsArtifact := &dbmodels.CatalogMetricsArtifactImpl{
		ID:     existingID, // Use existing ID if updating
		TypeID: &typeID,
		Attributes: &dbmodels.CatalogMetricsArtifactAttributes{
			Name:                     &artifactName,
			ExternalID:               &externalID,
			CreateTimeSinceEpoch:     createTime,
			LastUpdateTimeSinceEpoch: updateTime,
			MetricsType:              dbmodels.MetricsTypeAccuracy,
		},
		Properties:       &properties,
		CustomProperties: &customProperties,
	}

	return metricsArtifact
}

// metricsArtifactIdentity scopes a producer result to its catalog model and
// metrics type. Hashing the original UTF-8 bytes bounds the database key length
// and preserves distinctions that case-insensitive database collations lose.
func metricsArtifactIdentity(modelID int32, metricsType dbmodels.MetricsType, producerID string) string {
	return fmt.Sprintf("catalog-metrics:v1:%d:%s:%x", modelID, metricsType, sha256.Sum256([]byte(producerID)))
}

// createEvaluationArtifact creates one evaluation-metrics artifact per complete
// evaluations.ndjson record so callers can filter, paginate, and order run
// history without losing record-level metadata.
func createEvaluationArtifact(evalRecord evaluationRecord, modelID int32, typeID int32) *dbmodels.CatalogMetricsArtifactImpl {
	identity := metricsArtifactIdentity(modelID, dbmodels.MetricsTypeEvaluation, evalRecord.ID)
	properties := []models.Properties{}
	customProperties := make([]models.Properties, 0, len(evalRecord.CustomProperties))

	for key, value := range evalRecord.CustomProperties {
		if key == "created_at" || key == "updated_at" || value == nil {
			continue
		}

		property := models.Properties{Name: key}
		switch typedValue := value.(type) {
		case string:
			property.StringValue = &typedValue
		case float64:
			property.DoubleValue = &typedValue
		case bool:
			property.BoolValue = &typedValue
		case json.Number:
			if intValue, err := typedValue.Int64(); err == nil {
				property.SetInt64Value(intValue)
			} else if doubleValue, err := typedValue.Float64(); err == nil {
				property.DoubleValue = &doubleValue
			} else {
				stringValue := typedValue.String()
				property.StringValue = &stringValue
			}
		default:
			encoded, err := json.Marshal(typedValue)
			if err != nil {
				stringValue := fmt.Sprintf("%v", typedValue)
				property.StringValue = &stringValue
			} else {
				stringValue := string(encoded)
				property.StringValue = &stringValue
			}
		}
		customProperties = append(customProperties, property)
	}

	return &dbmodels.CatalogMetricsArtifactImpl{
		TypeID: &typeID,
		Attributes: &dbmodels.CatalogMetricsArtifactAttributes{
			Name:                     &identity,
			ExternalID:               &identity,
			CreateTimeSinceEpoch:     evalRecord.CreatedAt,
			LastUpdateTimeSinceEpoch: evalRecord.UpdatedAt,
			MetricsType:              dbmodels.MetricsTypeEvaluation,
		},
		Properties:       &properties,
		CustomProperties: &customProperties,
	}
}

// createPerformanceArtifact creates a metrics artifact from performance record
func createPerformanceArtifact(perfRecord performanceRecord, modelID int32, typeID int32, existingID *int32, existingCreateTime *int64) *dbmodels.CatalogMetricsArtifactImpl {
	artifactName := metricsArtifactIdentity(modelID, dbmodels.MetricsTypePerformance, perfRecord.ID)

	// Use existing create time if provided, otherwise extract from custom properties
	createTime := existingCreateTime
	var updateTime *int64

	if existingCreateTime == nil {
		if createdAtNum, ok := perfRecord.CustomProperties["created_at"].(json.Number); ok {
			createdAt, err := createdAtNum.Int64()
			if err == nil {
				createTime = &createdAt
			} else {
				glog.Warningf("%s: invalid created_at value: %v", artifactName, err)
			}
		}
	}
	if createTime == nil {
		createTime = new(time.Now().UnixMilli())
	}

	if updatedAtNum, ok := perfRecord.CustomProperties["updated_at"].(json.Number); ok {
		updatedAt, err := updatedAtNum.Int64()
		if err == nil {
			updateTime = &updatedAt
		} else {
			glog.Warningf("%s: invalid updated_at value: %v", artifactName, err)
		}
	}
	if updateTime == nil {
		updateTime = new(time.Now().UnixMilli())
	}
	delete(perfRecord.CustomProperties, "updated_at")
	delete(perfRecord.CustomProperties, "created_at")

	// Properties can be empty - all data goes in custom properties
	properties := []models.Properties{}

	// Create custom properties - simple mapping of all performance data
	customProperties := []models.Properties{}

	// Add all fields from the performance record as custom properties
	for key, value := range perfRecord.CustomProperties {
		prop := models.Properties{Name: key}

		// Handle different value types
		switch v := value.(type) {
		case string:
			prop.StringValue = &v
		case float64:
			prop.DoubleValue = &v
		case int64:
			prop.SetInt64Value(v)
		case int:
			intVal := int32(v)
			prop.IntValue = &intVal
		case bool:
			prop.BoolValue = &v
		case json.Number:
			if n, err := v.Int64(); err == nil {
				prop.SetInt64Value(n)
			} else if f, err := v.Float64(); err == nil {
				prop.DoubleValue = &f
			} else {
				// This shouldn't happen, but convert it to a string if it does.
				strVal := v.String()
				prop.StringValue = &strVal
			}
		default:
			// Convert other types to string representation
			strVal := fmt.Sprintf("%v", v)
			prop.StringValue = &strVal
		}

		customProperties = append(customProperties, prop)
	}

	// Create the metrics artifact with metricsType set to performance-metrics
	metricsArtifact := &dbmodels.CatalogMetricsArtifactImpl{
		ID:     existingID, // Use existing ID if updating
		TypeID: &typeID,
		Attributes: &dbmodels.CatalogMetricsArtifactAttributes{
			Name:                     &artifactName,
			ExternalID:               &artifactName,
			CreateTimeSinceEpoch:     createTime,
			LastUpdateTimeSinceEpoch: updateTime,
			MetricsType:              dbmodels.MetricsTypePerformance,
		},
		Properties:       &properties,
		CustomProperties: &customProperties,
	}

	return metricsArtifact
}

// parseSecurityEvaluationFile reads and parses a security-evaluations.ndjson file
func parseSecurityEvaluationFile(filePath string) ([]securityEvaluationRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open security evaluation file %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	securityRecords := []securityEvaluationRecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var secRecord securityEvaluationRecord
		if err := json.Unmarshal([]byte(line), &secRecord); err != nil {
			glog.Errorf("Failed to parse security evaluation record: %v", err)
			continue
		}

		securityRecords = append(securityRecords, secRecord)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading security evaluation file: %v", err)
	}

	return securityRecords, nil
}

// createSecurityArtifact creates a metrics artifact from a security evaluation record
func createSecurityArtifact(secRecord securityEvaluationRecord, modelID int32, typeID int32, existingID *int32, existingCreateTime *int64) *dbmodels.CatalogMetricsArtifactImpl {
	artifactName := metricsArtifactIdentity(modelID, dbmodels.MetricsTypeSecurityMetrics, secRecord.ID)

	createTime := existingCreateTime
	var updateTime *int64

	if existingCreateTime == nil {
		if createdAtNum, ok := secRecord.CustomProperties["created_at"].(json.Number); ok {
			createdAt, err := createdAtNum.Int64()
			if err == nil {
				createTime = &createdAt
			} else {
				glog.Warningf("%s: invalid created_at value: %v", artifactName, err)
			}
		}
	}
	if createTime == nil {
		createTime = new(time.Now().UnixMilli())
	}

	if updatedAtNum, ok := secRecord.CustomProperties["updated_at"].(json.Number); ok {
		updatedAt, err := updatedAtNum.Int64()
		if err == nil {
			updateTime = &updatedAt
		} else {
			glog.Warningf("%s: invalid updated_at value: %v", artifactName, err)
		}
	}
	if updateTime == nil {
		updateTime = new(time.Now().UnixMilli())
	}
	delete(secRecord.CustomProperties, "updated_at")
	delete(secRecord.CustomProperties, "created_at")

	properties := []models.Properties{}
	customProperties := []models.Properties{}

	for key, value := range secRecord.CustomProperties {
		prop := models.Properties{Name: key}

		switch v := value.(type) {
		case string:
			prop.StringValue = &v
		case float64:
			prop.DoubleValue = &v
		case int64:
			prop.SetInt64Value(v)
		case int:
			intVal := int32(v)
			prop.IntValue = &intVal
		case bool:
			prop.BoolValue = &v
		case json.Number:
			if n, err := v.Int64(); err == nil {
				prop.SetInt64Value(n)
			} else if f, err := v.Float64(); err == nil {
				prop.DoubleValue = &f
			} else {
				strVal := v.String()
				prop.StringValue = &strVal
			}
		default:
			strVal := fmt.Sprintf("%v", v)
			prop.StringValue = &strVal
		}

		customProperties = append(customProperties, prop)
	}

	metricsArtifact := &dbmodels.CatalogMetricsArtifactImpl{
		ID:     existingID,
		TypeID: &typeID,
		Attributes: &dbmodels.CatalogMetricsArtifactAttributes{
			Name:                     &artifactName,
			ExternalID:               &artifactName,
			CreateTimeSinceEpoch:     createTime,
			LastUpdateTimeSinceEpoch: updateTime,
			MetricsType:              dbmodels.MetricsTypeSecurityMetrics,
		},
		Properties:       &properties,
		CustomProperties: &customProperties,
	}

	return metricsArtifact
}

// enrichCatalogModelFromMetadata updates CatalogModel with additional fields from metadata.json
func enrichCatalogModelFromMetadata(existingModel dbmodels.CatalogModel, metadata metadataJSON, modelRepo dbmodels.CatalogModelRepository) error {
	// Build custom properties to add/update
	var customProperties []models.Properties

	if metadata.Size != nil && *metadata.Size != "" {
		customProperties = append(customProperties, models.Properties{
			Name:             "size",
			StringValue:      metadata.Size,
			IsCustomProperty: true,
		})
	}

	if metadata.TensorType != nil && *metadata.TensorType != "" {
		customProperties = append(customProperties, models.Properties{
			Name:             "tensor_type",
			StringValue:      metadata.TensorType,
			IsCustomProperty: true,
		})
	}

	if metadata.VariantGroupID != nil && *metadata.VariantGroupID != "" {
		customProperties = append(customProperties, models.Properties{
			Name:             "variant_group_id",
			StringValue:      metadata.VariantGroupID,
			IsCustomProperty: true,
		})
	}

	if metadata.MinVRAMGB != nil {
		customProperties = append(customProperties, models.Properties{
			Name:             "min_vram_gb",
			DoubleValue:      metadata.MinVRAMGB,
			IsCustomProperty: true,
		})
	}

	if metadata.ModelcarImageSize != nil {
		customProperties = append(customProperties, models.Properties{
			Name:             "modelcar_image_size",
			DoubleValue:      metadata.ModelcarImageSize,
			IsCustomProperty: true,
		})
	}

	if len(customProperties) == 0 {
		return nil // Nothing to update
	}

	// Merge new custom properties into the model, new values overwrite existing ones with same name.
	newNames := make(map[string]struct{}, len(customProperties))
	for _, p := range customProperties {
		newNames[p.Name] = struct{}{}
	}
	impl, ok := existingModel.(*dbmodels.CatalogModelImpl)
	if !ok {
		return fmt.Errorf("unexpected model type %T", existingModel)
	}
	if impl.CustomProperties == nil {
		impl.CustomProperties = &customProperties
	} else {
		filtered := (*impl.CustomProperties)[:0]
		for _, p := range *impl.CustomProperties {
			if _, overwritten := newNames[p.Name]; !overwritten {
				filtered = append(filtered, p)
			}
		}
		merged := append(filtered, customProperties...)
		impl.CustomProperties = &merged
	}

	// Save the updated model
	_, err := modelRepo.Save(existingModel)
	if err != nil {
		return fmt.Errorf("failed to save enriched model: %v", err)
	}

	glog.V(2).Infof("Enriched model %s with %d custom properties", *existingModel.GetAttributes().Name, len(customProperties))
	return nil
}
