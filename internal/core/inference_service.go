package core

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/internal/converter"
	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"gorm.io/gorm"
)

func (b *ModelRegistryService) UpsertInferenceService(inferenceService *openapi.InferenceService) (*openapi.InferenceService, error) {
	if inferenceService == nil {
		return nil, fmt.Errorf("invalid inference service pointer, cannot be nil: %w", api.ErrBadRequest)
	}

	if inferenceService.Id != nil {
		existing, err := b.GetInferenceServiceById(*inferenceService.Id)
		if err != nil {
			return nil, err
		}

		withNotEditable, err := b.mapper.UpdateExistingInferenceService(converter.NewOpenapiUpdateWrapper(existing, inferenceService))
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		inferenceService = &withNotEditable
	}

	_, err := b.GetServingEnvironmentById(inferenceService.ServingEnvironmentId)
	if err != nil {
		return nil, fmt.Errorf("no serving environment found for id %s: %w", inferenceService.ServingEnvironmentId, api.ErrNotFound)
	}

	infSvc, err := b.mapper.MapFromInferenceService(inferenceService, inferenceService.ServingEnvironmentId)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	savedInfSvc, err := b.inferenceServiceRepository.Save(infSvc)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("inference service with name %s already exists: %w", *infSvc.GetAttributes().Name, api.ErrConflict)
		}

		return nil, err
	}

	toReturn, err := b.mapper.MapToInferenceService(savedInfSvc)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetInferenceServiceById(id string) (*openapi.InferenceService, error) {
	glog.Infof("Getting InferenceService by id %s", id)

	convertedId, err := apiutils.ValidateIDAsInt32(id, "inference service")
	if err != nil {
		return nil, err
	}

	model, err := b.inferenceServiceRepository.GetByID(convertedId)
	if err != nil {
		return nil, fmt.Errorf("no InferenceService found for id %s: %w", id, api.ErrNotFound)
	}

	toReturn, err := b.mapper.MapToInferenceService(model)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetInferenceServiceByParams(name *string, parentResourceId *string, externalId *string) (*openapi.InferenceService, error) {
	// Caller MUST provide either name and parentResourceId or externalId
	if (name == nil || parentResourceId == nil) && externalId == nil {
		return nil, fmt.Errorf("invalid parameters call, supply either (name and parentResourceId), or externalId: %w", api.ErrBadRequest)
	}

	var parentResourceID *int32
	if parentResourceId != nil {
		var err error
		parentResourceID, err = apiutils.ValidateIDAsInt32Ptr(parentResourceId, "parent resource")
		if err != nil {
			return nil, err
		}
	}

	infServicesList, err := b.inferenceServiceRepository.List(models.InferenceServiceListOptions{
		Name:             name,
		ExternalID:       externalId,
		ParentResourceID: parentResourceID,
	})
	if err != nil {
		return nil, err
	}

	if len(infServicesList.Items) > 1 {
		return nil, fmt.Errorf("multiple inference services found for name=%v, parentResourceId=%v, externalId=%v: %w", apiutils.ZeroIfNil(name), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	if len(infServicesList.Items) == 0 {
		return nil, fmt.Errorf("no inference service found for name=%v, parentResourceId=%v, externalId=%v: %w", apiutils.ZeroIfNil(name), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	glog.Infof("Found InferenceService - with name=%v, parentResourceId=%v, externalId=%v", apiutils.ZeroIfNil(name), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId))

	toReturn, err := b.mapper.MapToInferenceService(infServicesList.Items[0])
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetInferenceServices(listOptions api.ListOptions, servingEnvironmentId *string, runtime *string) (*openapi.InferenceServiceList, error) {
	var parentResourceID *int32

	if servingEnvironmentId != nil {
		var err error
		parentResourceID, err = apiutils.ValidateIDAsInt32Ptr(servingEnvironmentId, "serving environment")
		if err != nil {
			return nil, err
		}
	}

	infServicesList, err := b.inferenceServiceRepository.List(models.InferenceServiceListOptions{
		Pagination: models.Pagination{
			PageSize:      listOptions.PageSize,
			OrderBy:       listOptions.OrderBy,
			SortOrder:     listOptions.SortOrder,
			NextPageToken: listOptions.NextPageToken,
			FilterQuery:   listOptions.FilterQuery,
		},
		Runtime:          runtime,
		ParentResourceID: parentResourceID,
	})
	if err != nil {
		return nil, err
	}

	inferenceServiceList := &openapi.InferenceServiceList{
		Items: []openapi.InferenceService{},
	}

	for _, infSvc := range infServicesList.Items {
		inferenceService, err := b.mapper.MapToInferenceService(infSvc)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		inferenceServiceList.Items = append(inferenceServiceList.Items, *inferenceService)
	}

	inferenceServiceList.NextPageToken = infServicesList.NextPageToken
	inferenceServiceList.PageSize = infServicesList.PageSize
	inferenceServiceList.Size = int32(infServicesList.Size)

	return inferenceServiceList, nil
}
