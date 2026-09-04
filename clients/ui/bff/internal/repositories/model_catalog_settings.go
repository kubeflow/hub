package repositories

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ModelCatalogSettingsRepository struct {
}

func NewModelCatalogSettingsRepository() *ModelCatalogSettingsRepository {
	return &ModelCatalogSettingsRepository{}
}

var (
	ErrCatalogSourceNotFound     = errors.New("catalog source not found")
	ErrCatalogSourceAlreadyExist = errors.New("catalog source already exists")
	ErrCatalogSourceIdRequired   = errors.New("catalog source ID is required")
	ErrUnsupportedCatalogType    = errors.New("unsupported catalog type")
	ErrCannotChangeDefaultSource = errors.New("cannot change the default source")
	ErrCannotDeleteDefaultSource = errors.New("cannot delete the default source")
	ErrCatalogIDTooLong          = errors.New("catalog source ID exceeds maximum length for secret name")
	ErrCatalogIdInvalid          = errors.New("catalog source ID must contain at least one letter or digit")
	ErrCannotChangeType          = errors.New("cannot change catalog source type")
	ErrValidationFailed          = errors.New("validation failed")
	ErrCatalogSourceConflict     = errors.New("catalog source was modified by another request")
)

const (
	CatalogTypeYaml        = "yaml"
	CatalogTypeHuggingFace = "hf"
	ApiKey                 = "apiKey"
)

func (r *ModelCatalogSettingsRepository) GetAllCatalogSourceConfigs(ctx context.Context, client k8s.KubernetesClientInterface, namespace string) (*models.CatalogSourceConfigList, error) {
	defaultCM, userCM, err := client.GetAllCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog source configmaps: %w", err)
	}

	catalogMap := make(map[string]models.CatalogSourceConfig)

	if raw, ok := defaultCM.Data[k8s.CatalogSourceKey]; ok {
		defaultCatalogSources, err := ParseCatalogYaml(raw, true)
		if err != nil {
			return nil, fmt.Errorf("failed to parse default catalogs: %w", err)
		}

		for _, catalog := range defaultCatalogSources {
			catalogMap[catalog.Id] = catalog
		}
	}

	if raw, ok := userCM.Data[k8s.CatalogSourceKey]; ok {
		userManagedSources, err := ParseCatalogYaml(raw, false)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user managed catalogs: %w", err)
		}

		for _, userCatalogSource := range userManagedSources {
			if existingSource, exist := catalogMap[userCatalogSource.Id]; exist {
				mergedCatalogSources := mergeCatalogSourceConfigs(existingSource, userCatalogSource)
				catalogMap[userCatalogSource.Id] = mergedCatalogSources
			} else {
				catalogMap[userCatalogSource.Id] = userCatalogSource
			}
		}
	}

	catalogSources := &models.CatalogSourceConfigList{
		Catalogs: make([]models.CatalogSourceConfig, 0),
	}

	for _, c := range catalogMap {
		catalogSources.Catalogs = append(catalogSources.Catalogs, c)
	}

	return catalogSources, nil
}

func (r *ModelCatalogSettingsRepository) GetCatalogSourceConfig(ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	catalogSourceId string,
) (*models.CatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog source configmaps: %w", err)
	}

	defaultSource := FindCatalogSourceById(defaultCM.Data[k8s.CatalogSourceKey], catalogSourceId, true)
	userSource := FindCatalogSourceById(userCM.Data[k8s.CatalogSourceKey], catalogSourceId, false)

	var result *models.CatalogSourceConfig

	if userSource != nil {
		if defaultSource != nil {
			merged := mergeCatalogSourceConfigs(*defaultSource, *userSource)
			result = &merged
		} else {
			result = userSource
		}
	} else if defaultSource != nil {
		result = defaultSource
	} else {
		return nil, fmt.Errorf("%w, %s", ErrCatalogSourceNotFound, catalogSourceId)
	}

	_, yamlFilePath := FindCatalogSourceProperties(userCM.Data[k8s.CatalogSourceKey], catalogSourceId)
	if yamlFilePath == "" {
		_, yamlFilePath = FindCatalogSourceProperties(defaultCM.Data[k8s.CatalogSourceKey], catalogSourceId)
	}

	if result.Type == CatalogTypeYaml {
		if yamlFilePath != "" {
			result.YamlCatalogPath = &yamlFilePath
			if yamlContent, ok := userCM.Data[yamlFilePath]; ok {
				result.Yaml = &yamlContent
			} else if yamlContent, ok := defaultCM.Data[yamlFilePath]; ok {
				result.Yaml = &yamlContent
			} else if result.IsDefault == nil || !*result.IsDefault {
				// Only warn for non-default sources
				sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)
				sessionLogger.Warn("yaml catalog content missing from configmap",
					"catalogId", catalogSourceId,
					"expectedPath", yamlFilePath,
				)
			}
		}
	}

	return result, nil
}

func (r *ModelCatalogSettingsRepository) CreateCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	payload models.CatalogSourceConfigPayload,
) (*models.CatalogSourceConfig, error) {
	if err := validateCatalogSourceConfigPayload(payload); err != nil {
		return nil, err
	}

	defaultCM, userCM, err := client.GetAllCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog source configmaps: %w", err)
	}

	if FindCatalogSourceById(defaultCM.Data[k8s.CatalogSourceKey], payload.Id, true) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in default sources", ErrCatalogSourceAlreadyExist, payload.Id)
	}

	if FindCatalogSourceById(userCM.Data[k8s.CatalogSourceKey], payload.Id, false) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in user managed sources", ErrCatalogSourceAlreadyExist, payload.Id)
	}

	var secretName string
	var yamlFileName string
	yamlContent := make(map[string]string)

	switch payload.Type {
	case CatalogTypeYaml:
		yamlFileName = fmt.Sprintf("%s.yaml", payload.Id)
		yamlContent[yamlFileName] = *payload.Yaml
	case CatalogTypeHuggingFace:
		// Only create secret if apiKey is provided
		if payload.ApiKey != nil && *payload.ApiKey != "" {
			secretName, err = createSecretForHuggingFace(ctx, client, namespace, payload.Id, *payload.ApiKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create secret for huggingface source: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCatalogType, payload.Type)
	}

	newEntry := ConvertSourceConfigToYamlEntry(payload, yamlFileName, secretName)

	existingConfigMapEntry := userCM.Data[k8s.CatalogSourceKey]

	updatedConfigMapEntry, err := AppendCatalogSourceToYaml(existingConfigMapEntry, newEntry)
	if err != nil {
		if secretName != "" {
			deleteSecretForHuggingFace(ctx, client, namespace, secretName)
		}
		return nil, fmt.Errorf("failed to append catalog to yaml: %w", err)
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}
	userCM.Data[k8s.CatalogSourceKey] = updatedConfigMapEntry

	for key, value := range yamlContent {
		userCM.Data[key] = value
	}

	err = client.UpdateCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if secretName != "" {
			deleteSecretForHuggingFace(ctx, client, namespace, secretName)
		}
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user configmap: %w", err)
	}

	return r.GetCatalogSourceConfig(ctx, client, namespace, payload.Id)
}

func (r *ModelCatalogSettingsRepository) UpdateCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	sourceId string,
	payload models.CatalogSourceConfigPayload,
) (*models.CatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog source configmaps: %w", err)
	}

	existingUserSource := FindCatalogSourceById(userCM.Data[k8s.CatalogSourceKey], sourceId, false)
	existingDefaultSource := FindCatalogSourceById(defaultCM.Data[k8s.CatalogSourceKey], sourceId, true)

	var existingCatalog *models.CatalogSourceConfig
	var isOverridingDefault bool

	if existingUserSource != nil {
		existingCatalog = existingUserSource
		isOverridingDefault = existingDefaultSource != nil
	} else if existingDefaultSource != nil {
		existingCatalog = existingDefaultSource
		isOverridingDefault = true
	} else {
		return nil, fmt.Errorf("%w: '%s'", ErrCatalogSourceNotFound, sourceId)
	}

	if payload.Type != "" && payload.Type != existingCatalog.Type {
		return nil, fmt.Errorf(
			"%w: cannot change from '%s' to '%s'",
			ErrCannotChangeType,
			existingCatalog.Type,
			payload.Type,
		)
	}

	catalogType := existingCatalog.Type

	if isOverridingDefault {
		if err := validateUpdatePayloadForDefaultOverride(payload); err != nil {
			return nil, err
		}
	}

	var secretName, yamlFilePath string
	if !isOverridingDefault || existingUserSource != nil {
		secretName, yamlFilePath = FindCatalogSourceProperties(userCM.Data[k8s.CatalogSourceKey], sourceId)
		if secretName == "" || yamlFilePath == "" {
			defaultSecretName, defaultYamlPath := FindCatalogSourceProperties(defaultCM.Data[k8s.CatalogSourceKey], sourceId)
			if secretName == "" {
				secretName = defaultSecretName
			}
			if yamlFilePath == "" {
				yamlFilePath = defaultYamlPath
			}
		}
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}

	if !isOverridingDefault || existingUserSource != nil {
		switch catalogType {
		case CatalogTypeYaml:
			if payload.Yaml != nil && *payload.Yaml != "" {
				if yamlFilePath == "" {
					yamlFilePath = fmt.Sprintf("%s.yaml", sourceId)
				}
				userCM.Data[yamlFilePath] = *payload.Yaml
			}

		case CatalogTypeHuggingFace:
			if payload.ApiKey != nil && *payload.ApiKey != "" {
				if secretName != "" {
					err := client.PatchSecret(ctx, namespace, secretName, map[string]string{
						ApiKey: *payload.ApiKey,
					})
					if err != nil {
						if apierrors.IsNotFound(err) {
							secretName, err = createSecretForHuggingFace(ctx, client, namespace, sourceId, *payload.ApiKey)
							if err != nil {
								return nil, fmt.Errorf("failed to create replacement secret: %w", err)
							}
						} else {
							return nil, fmt.Errorf("failed to patch secret '%s': %w", secretName, err)
						}
					}
				} else {
					secretName, err = createSecretForHuggingFace(ctx, client, namespace, sourceId, *payload.ApiKey)
					if err != nil {
						return nil, fmt.Errorf("failed to create secret: %w", err)
					}
				}
			}
		}
	}

	if existingUserSource != nil {
		updatedYAML, err := UpdateCatalogSourceInYAML(
			userCM.Data[k8s.CatalogSourceKey],
			sourceId,
			payload,
			secretName,
			yamlFilePath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update catalog in yaml: %w", err)
		}
		userCM.Data[k8s.CatalogSourceKey] = updatedYAML
	} else {
		overrideEntry := BuildOverrideEntryForDefaultSource(sourceId, payload)
		updatedYAML, err := AppendCatalogSourceToYaml(userCM.Data[k8s.CatalogSourceKey], overrideEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to append override entry: %w", err)
		}
		userCM.Data[k8s.CatalogSourceKey] = updatedYAML
	}

	err = client.UpdateCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user configmap: %w", err)
	}

	return r.GetCatalogSourceConfig(ctx, client, namespace, sourceId)

}

func (r *ModelCatalogSettingsRepository) DeleteCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	catalogSourceId string,
) (*models.CatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog source configmaps: %w", err)
	}

	if FindCatalogSourceById(defaultCM.Data[k8s.CatalogSourceKey], catalogSourceId, true) != nil {
		return nil, fmt.Errorf("%w: '%s' is a default source", ErrCannotDeleteDefaultSource, catalogSourceId)
	}

	catalogSourceToDelete := FindCatalogSourceById(userCM.Data[k8s.CatalogSourceKey], catalogSourceId, false)
	if catalogSourceToDelete == nil {
		return nil, fmt.Errorf("%w: '%s' not found in user sources", ErrCatalogSourceNotFound, catalogSourceId)
	}

	secretName, yamlFilePath := FindCatalogSourceProperties(userCM.Data[k8s.CatalogSourceKey], catalogSourceId)

	if catalogSourceToDelete.Type == CatalogTypeHuggingFace && secretName != "" {
		err := client.DeleteSecret(ctx, namespace, secretName)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf(
					"failed to delete secret '%s' for catalog '%s': %w. Source was not deleted",
					secretName,
					catalogSourceId,
					err,
				)
			}
		}
	}

	if catalogSourceToDelete.Type == CatalogTypeYaml && yamlFilePath != "" {
		delete(userCM.Data, yamlFilePath)
	}

	updatedYAML, err := RemoveCatalogSourceFromYAML(userCM.Data[k8s.CatalogSourceKey], catalogSourceId)
	if err != nil {
		return nil, fmt.Errorf("failed to remove catalog from sources.yaml: %w", err)
	}
	userCM.Data[k8s.CatalogSourceKey] = updatedYAML

	err = client.UpdateCatalogSourceConfig(ctx, namespace, &userCM)
	if err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update configmap after deletion: %w", err)
	}

	return catalogSourceToDelete, nil
}

func mergeCatalogSourceConfigs(defaultCatalog models.CatalogSourceConfig, userCatalog models.CatalogSourceConfig) models.CatalogSourceConfig {
	mergedSource := defaultCatalog

	if userCatalog.Name != "" {
		mergedSource.Name = userCatalog.Name
	}

	if userCatalog.Type != "" {
		mergedSource.Type = userCatalog.Type
	}

	if userCatalog.Enabled != nil {
		mergedSource.Enabled = userCatalog.Enabled
	}

	if userCatalog.AllowedOrganization != nil {
		mergedSource.AllowedOrganization = userCatalog.AllowedOrganization
	}

	if userCatalog.ApiKey != nil {
		mergedSource.ApiKey = userCatalog.ApiKey
	}

	if userCatalog.IncludedModels != nil {
		mergedSource.IncludedModels = userCatalog.IncludedModels
	}

	if userCatalog.ExcludedModels != nil {
		mergedSource.ExcludedModels = userCatalog.ExcludedModels
	}

	if userCatalog.Yaml != nil {
		mergedSource.Yaml = userCatalog.Yaml
	}

	return mergedSource
}

func validateCatalogSourceConfigPayload(payload models.CatalogSourceConfigPayload) error {
	if payload.Id == "" {
		return fmt.Errorf("%w", ErrCatalogSourceIdRequired)
	}

	if err := validateCatalogId(payload.Id); err != nil {
		return err
	}

	if payload.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidationFailed)
	}

	if payload.Type == "" {
		return fmt.Errorf("%w: type is required", ErrValidationFailed)
	}

	switch payload.Type {
	case CatalogTypeYaml:
		if payload.Yaml == nil || *payload.Yaml == "" {
			return fmt.Errorf("%w: yaml field is required for yaml-type sources", ErrValidationFailed)
		}
	case CatalogTypeHuggingFace:
		if payload.AllowedOrganization == nil || *payload.AllowedOrganization == "" {
			return fmt.Errorf("%w: allowedOrganization is required for huggingface-type sources", ErrValidationFailed)
		}
		// apiKey is optional for HuggingFace sources
	default:
		return fmt.Errorf("%w: unsupported catalog type: %s (supported: yaml, huggingface)", ErrValidationFailed, payload.Type)
	}
	return nil
}

func createSecretForHuggingFace(ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	catalogId string,
	apiKey string) (string, error) {
	modifiedSecretName := strings.ReplaceAll(catalogId, "_", "-")
	secretName := fmt.Sprintf("catalog-%s-apikey", modifiedSecretName)

	// limit for secretName is 253. so catalogId length should not exceed 238
	if len(secretName) > 253 {
		return "", fmt.Errorf("%w: '%s' (max 238 characters for ID)", ErrCatalogIDTooLong, catalogId)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/component": "model-catalog",
				"hub.kubeflow.org/hf-source":  hfSourceLabelValue(catalogId),
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			ApiKey: apiKey,
		},
	}
	secretCreated, err := client.CreateSecret(ctx, namespace, secret)
	if err != nil {
		return "", fmt.Errorf("failed to create secret '%s': %w", secretName, err)
	}
	return secretCreated.Name, nil

}

func deleteSecretForHuggingFace(ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	secretName string) {
	err := client.DeleteSecret(ctx, namespace, secretName)
	if err != nil && !apierrors.IsNotFound(err) {
		sessionLogger := ctx.Value(constants.TraceLoggerKey).(*slog.Logger)
		sessionLogger.Warn("failed to cleanup secret during rollback",
			"secretName", secretName,
			"namespace", namespace,
			"error", err,
		)
	}
}

// hfSourceLabelValueMaxLen bounds hfSourceLabelValue's output to 63 characters,
// the Kubernetes limit on label values.
const hfSourceLabelValueMaxLen = 63

// hfSourceLabelValue converts a catalog source ID into a value usable both as
// the "hub.kubeflow.org/hf-source" label on the HuggingFace API key secret and
// as the suffix of the HF_API_KEY_<SUFFIX> environment variable the catalog
// service resolves the key from (see envVarSuffix in
// catalog/internal/catalog/modelcatalog/hf_catalog.go). The two must stay in
// sync: whatever normalization changes here should be mirrored there, and
// vice versa.
//
// It uppercases the ID, replaces every character that is not A-Z, 0-9, or '_'
// with '_', trims leading/trailing '_' (a Kubernetes label value must begin
// and end with an alphanumeric), and truncates to hfSourceLabelValueMaxLen
// characters (re-trimming any trailing '_' left by the truncation).
// validateCatalogId already restricts source IDs to [a-z0-9_]+, so the only
// normalization this performs in practice is uppercasing, trimming leading or
// trailing underscores, and truncating IDs longer than 63 characters.
func hfSourceLabelValue(id string) string {
	var b strings.Builder
	b.Grow(len(id))
	for _, r := range strings.ToUpper(id) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	raw := b.String()

	trimmed := strings.Trim(raw, "_")
	if len(trimmed) > hfSourceLabelValueMaxLen {
		trimmed = strings.TrimRight(trimmed[:hfSourceLabelValueMaxLen], "_")
	}

	// Degenerate case: the ID normalized to all underscores (e.g. "___").
	// Fall back to the untrimmed form rather than returning "".
	if trimmed == "" {
		return raw
	}
	return trimmed
}

var validCatalogIdRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

func validateCatalogId(id string) error {
	if id == "" {
		return ErrCatalogSourceIdRequired
	}

	if !validCatalogIdRegex.MatchString(id) {
		return fmt.Errorf("invalid catalog ID: must contain only lowercase letters, numbers, and underscores")
	}

	if strings.Trim(id, "_") == "" {
		return ErrCatalogIdInvalid
	}

	if len(id) > 238 {
		return fmt.Errorf("%w: '%s' (max 238 characters)", ErrCatalogIDTooLong, id)
	}

	return nil
}

func validateUpdatePayloadForDefaultOverride(payload models.CatalogSourceConfigPayload) error {
	if payload.Name != "" {
		return fmt.Errorf("%w: cannot change 'name'", ErrCannotChangeDefaultSource)
	}

	if len(payload.Labels) > 0 {
		return fmt.Errorf("%w: cannot change 'labels'", ErrCannotChangeDefaultSource)
	}

	if payload.Yaml != nil && *payload.Yaml != "" {
		return fmt.Errorf("%w: cannot change 'yaml' content", ErrCannotChangeDefaultSource)
	}

	return nil
}
