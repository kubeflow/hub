package core

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/internal/converter"
	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"gorm.io/gorm"
)

func (b *ModelRegistryService) UpsertServingEnvironment(servingEnvironment *openapi.ServingEnvironment) (*openapi.ServingEnvironment, error) {
	if servingEnvironment == nil {
		return nil, fmt.Errorf("invalid serving environment pointer, cannot be nil: %w", api.ErrBadRequest)
	}

	if servingEnvironment.Id != nil {
		existing, err := b.GetServingEnvironmentById(*servingEnvironment.Id)
		if err != nil {
			return nil, err
		}

		withNotEditable, err := b.mapper.UpdateExistingServingEnvironment(converter.NewOpenapiUpdateWrapper(existing, servingEnvironment))
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		servingEnvironment = &withNotEditable
	}

	servEnv, err := b.mapper.MapFromServingEnvironment(servingEnvironment)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	savedServEnv, err := b.servingEnvironmentRepository.Save(servEnv)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("serving environment with name %s already exists: %w", servingEnvironment.Name, api.ErrConflict)
		}

		return nil, err
	}

	toReturn, err := b.mapper.MapToServingEnvironment(savedServEnv)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetServingEnvironmentById(id string) (*openapi.ServingEnvironment, error) {
	convertedId, err := apiutils.ValidateIDAsInt32(id, "serving environment")
	if err != nil {
		return nil, err
	}

	model, err := b.servingEnvironmentRepository.GetByID(convertedId)
	if err != nil {
		return nil, fmt.Errorf("no serving environment found for id %s: %w", id, api.ErrNotFound)
	}

	toReturn, err := b.mapper.MapToServingEnvironment(model)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetServingEnvironmentByParams(name *string, externalId *string) (*openapi.ServingEnvironment, error) {
	if name == nil && externalId == nil {
		return nil, fmt.Errorf("invalid parameters call, supply either name or externalId: %w", api.ErrBadRequest)
	}

	servEnvsList, err := b.servingEnvironmentRepository.List(models.ServingEnvironmentListOptions{
		Name:       name,
		ExternalID: externalId,
	})
	if err != nil {
		return nil, err
	}

	if len(servEnvsList.Items) == 0 {
		return nil, fmt.Errorf("no serving environment found for name=%v, externalId=%v: %w", apiutils.ZeroIfNil(name), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	if len(servEnvsList.Items) > 1 {
		return nil, fmt.Errorf("multiple serving environments found for name=%v, externalId=%v: %w", apiutils.ZeroIfNil(name), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	toReturn, err := b.mapper.MapToServingEnvironment(servEnvsList.Items[0])
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetServingEnvironments(listOptions api.ListOptions) (*openapi.ServingEnvironmentList, error) {
	servEnvsList, err := b.servingEnvironmentRepository.List(models.ServingEnvironmentListOptions{
		Pagination: models.Pagination{
			PageSize:      listOptions.PageSize,
			OrderBy:       listOptions.OrderBy,
			SortOrder:     listOptions.SortOrder,
			NextPageToken: listOptions.NextPageToken,
			FilterQuery:   listOptions.FilterQuery,
		},
	})
	if err != nil {
		return nil, err
	}

	servingEnvironmentList := &openapi.ServingEnvironmentList{
		Items: []openapi.ServingEnvironment{},
	}

	for _, servEnv := range servEnvsList.Items {
		servingEnvironment, err := b.mapper.MapToServingEnvironment(servEnv)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		servingEnvironmentList.Items = append(servingEnvironmentList.Items, *servingEnvironment)
	}

	servingEnvironmentList.NextPageToken = servEnvsList.NextPageToken
	servingEnvironmentList.PageSize = servEnvsList.PageSize
	servingEnvironmentList.Size = int32(servEnvsList.Size)

	return servingEnvironmentList, nil
}
