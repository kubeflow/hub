package repositories

import (
	"context"
	"errors"
	"fmt"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	ErrSkillCatalogSourceNotFound      = errors.New("skill catalog source not found")
	ErrSkillCatalogSourceAlreadyExist  = errors.New("skill catalog source already exists")
	ErrSkillCatalogSourceIdRequired    = errors.New("skill catalog source ID is required")
	ErrSkillCatalogSourceConflict      = errors.New("skill catalog source was modified by another request")
	ErrSkillCatalogCannotChangeDefault = errors.New("cannot change the default skill source")
	ErrSkillCatalogCannotDeleteDefault = errors.New("cannot delete the default skill source")
	ErrSkillCatalogValidationFailed    = errors.New("validation failed")
	ErrSkillCatalogCannotChangeType    = errors.New("cannot change skill catalog source type")
)

type SkillCatalogSettingsRepository struct{}

func NewSkillCatalogSettingsRepository() *SkillCatalogSettingsRepository {
	return &SkillCatalogSettingsRepository{}
}

func (r *SkillCatalogSettingsRepository) GetAllSkillCatalogSourceConfigs(ctx context.Context, client k8s.KubernetesClientInterface, namespace string) (*models.SkillCatalogSourceConfigList, error) {
	defaultCM, userCM, err := client.GetAllSkillCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill catalog source configmaps: %w", err)
	}

	catalogMap := make(map[string]models.SkillCatalogSourceConfig)

	if raw, ok := defaultCM.Data[k8s.SkillCatalogSourceKey]; ok {
		defaultSources, err := ParseSkillCatalogYaml(raw, true)
		if err != nil {
			return nil, fmt.Errorf("failed to parse default skill catalogs: %w", err)
		}
		for _, src := range defaultSources {
			catalogMap[src.Id] = src
		}
	}

	if raw, ok := userCM.Data[k8s.SkillCatalogSourceKey]; ok {
		userSources, err := ParseSkillCatalogYaml(raw, false)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user managed skill catalogs: %w", err)
		}
		for _, userSrc := range userSources {
			if existing, ok := catalogMap[userSrc.Id]; ok {
				catalogMap[userSrc.Id] = mergeSkillCatalogSourceConfigs(existing, userSrc)
			} else {
				catalogMap[userSrc.Id] = userSrc
			}
		}
	}

	list := &models.SkillCatalogSourceConfigList{
		Catalogs: make([]models.SkillCatalogSourceConfig, 0, len(catalogMap)),
	}
	for _, src := range catalogMap {
		list.Catalogs = append(list.Catalogs, src)
	}

	return list, nil
}

func (r *SkillCatalogSettingsRepository) GetSkillCatalogSourceConfig(ctx context.Context, client k8s.KubernetesClientInterface, namespace, sourceID string) (*models.SkillCatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllSkillCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill catalog source configmaps: %w", err)
	}

	defaultSrc := FindSkillCatalogSourceById(defaultCM.Data[k8s.SkillCatalogSourceKey], sourceID, true)
	userSrc := FindSkillCatalogSourceById(userCM.Data[k8s.SkillCatalogSourceKey], sourceID, false)

	if userSrc != nil {
		if defaultSrc != nil {
			merged := mergeSkillCatalogSourceConfigs(*defaultSrc, *userSrc)
			return &merged, nil
		}
		return userSrc, nil
	}
	if defaultSrc != nil {
		return defaultSrc, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrSkillCatalogSourceNotFound, sourceID)
}

func (r *SkillCatalogSettingsRepository) CreateSkillCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	payload models.SkillCatalogSourceConfigPayload,
) (*models.SkillCatalogSourceConfig, error) {
	if err := validateSkillCatalogSourceConfigPayload(payload); err != nil {
		return nil, err
	}

	defaultCM, userCM, err := client.GetAllSkillCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill catalog source configmaps: %w", err)
	}

	if FindSkillCatalogSourceById(defaultCM.Data[k8s.SkillCatalogSourceKey], payload.Id, true) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in default sources", ErrSkillCatalogSourceAlreadyExist, payload.Id)
	}
	if FindSkillCatalogSourceById(userCM.Data[k8s.SkillCatalogSourceKey], payload.Id, false) != nil {
		return nil, fmt.Errorf("%w: '%s' already exists in user managed sources", ErrSkillCatalogSourceAlreadyExist, payload.Id)
	}

	newEntry := ConvertSkillSourceConfigToYamlEntry(payload)
	existing := userCM.Data[k8s.SkillCatalogSourceKey]

	updated, err := AppendSkillCatalogSourceToYaml(existing, newEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to append skill catalog to yaml: %w", err)
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}
	userCM.Data[k8s.SkillCatalogSourceKey] = updated

	if err := client.UpdateSkillCatalogSourceConfig(ctx, namespace, &userCM); err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrSkillCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user skill catalog configmap: %w", err)
	}

	return r.GetSkillCatalogSourceConfig(ctx, client, namespace, payload.Id)
}

func (r *SkillCatalogSettingsRepository) UpdateSkillCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace, sourceID string,
	payload models.SkillCatalogSourceConfigPayload,
) (*models.SkillCatalogSourceConfig, error) {
	if err := validateSkillCatalogUpdatePayload(payload); err != nil {
		return nil, err
	}

	defaultCM, userCM, err := client.GetAllSkillCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill catalog source configmaps: %w", err)
	}

	existingUser := FindSkillCatalogSourceById(userCM.Data[k8s.SkillCatalogSourceKey], sourceID, false)
	existingDefault := FindSkillCatalogSourceById(defaultCM.Data[k8s.SkillCatalogSourceKey], sourceID, true)

	if existingUser == nil && existingDefault == nil {
		return nil, fmt.Errorf("%w: '%s'", ErrSkillCatalogSourceNotFound, sourceID)
	}

	isOverridingDefault := existingDefault != nil

	if payload.Type != "" {
		existing := existingUser
		if existing == nil {
			existing = existingDefault
		}
		if payload.Type != existing.Type {
			return nil, fmt.Errorf("%w: cannot change from '%s' to '%s'", ErrSkillCatalogCannotChangeType, existing.Type, payload.Type)
		}
	}

	// Applies whenever a default with this id exists, not only on the first override:
	// once an enabled-toggle has written a user entry, later updates take the branch
	// below, which would otherwise let name/repositories through unchecked. This mirrors
	// where the MCP catalog places the same check.
	if isOverridingDefault {
		if err := validateSkillUpdatePayloadForDefaultOverride(payload); err != nil {
			return nil, err
		}
	}

	if userCM.Data == nil {
		userCM.Data = make(map[string]string)
	}

	if existingUser != nil {
		updatedYAML, err := UpdateSkillCatalogSourceInYAML(userCM.Data[k8s.SkillCatalogSourceKey], sourceID, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to update skill catalog in yaml: %w", err)
		}
		userCM.Data[k8s.SkillCatalogSourceKey] = updatedYAML
	} else if isOverridingDefault {
		overrideEntry := map[string]interface{}{"id": sourceID}
		if payload.Enabled != nil {
			overrideEntry["enabled"] = *payload.Enabled
		}
		updatedYAML, err := AppendSkillCatalogSourceToYaml(userCM.Data[k8s.SkillCatalogSourceKey], overrideEntry)
		if err != nil {
			return nil, fmt.Errorf("failed to append override entry: %w", err)
		}
		userCM.Data[k8s.SkillCatalogSourceKey] = updatedYAML
	}

	if err := client.UpdateSkillCatalogSourceConfig(ctx, namespace, &userCM); err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrSkillCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update user skill catalog configmap: %w", err)
	}

	return r.GetSkillCatalogSourceConfig(ctx, client, namespace, sourceID)
}

func (r *SkillCatalogSettingsRepository) DeleteSkillCatalogSourceConfig(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace, sourceID string,
) (*models.SkillCatalogSourceConfig, error) {
	defaultCM, userCM, err := client.GetAllSkillCatalogSourceConfigs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill catalog source configmaps: %w", err)
	}

	if FindSkillCatalogSourceById(defaultCM.Data[k8s.SkillCatalogSourceKey], sourceID, true) != nil {
		return nil, fmt.Errorf("%w: '%s' is a default source", ErrSkillCatalogCannotDeleteDefault, sourceID)
	}

	toDelete := FindSkillCatalogSourceById(userCM.Data[k8s.SkillCatalogSourceKey], sourceID, false)
	if toDelete == nil {
		return nil, fmt.Errorf("%w: '%s' not found in user sources", ErrSkillCatalogSourceNotFound, sourceID)
	}

	updatedYAML, err := RemoveSkillCatalogSourceFromYAML(userCM.Data[k8s.SkillCatalogSourceKey], sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove skill catalog from sources.yaml: %w", err)
	}
	userCM.Data[k8s.SkillCatalogSourceKey] = updatedYAML

	if err := client.UpdateSkillCatalogSourceConfig(ctx, namespace, &userCM); err != nil {
		if apierrors.IsConflict(err) {
			return nil, fmt.Errorf("%w: %v", ErrSkillCatalogSourceConflict, err)
		}
		return nil, fmt.Errorf("failed to update configmap after deletion: %w", err)
	}

	return toDelete, nil
}
