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

func (b *ModelRegistryService) UpsertServeModel(serveModel *openapi.ServeModel, inferenceServiceId *string) (*openapi.ServeModel, error) {
	if serveModel == nil {
		return nil, fmt.Errorf("invalid serve model pointer, cannot be nil: %w", api.ErrBadRequest)
	}

	if serveModel.Id != nil {
		existing, err := b.GetServeModelById(*serveModel.Id)
		if err != nil {
			return nil, err
		}

		withNotEditable, err := b.mapper.UpdateExistingServeModel(converter.NewOpenapiUpdateWrapper(existing, serveModel))
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		serveModel = &withNotEditable
	}

	// Validate HuggingFace model deployment requirements
	if err := b.validateHFModelDeployment(serveModel.ModelVersionId); err != nil {
		return nil, err
	}

	srvModel, err := b.mapper.MapFromServeModel(serveModel, inferenceServiceId)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	if inferenceServiceId == nil && srvModel.GetID() == nil {
		return nil, fmt.Errorf("missing inferenceServiceId, cannot create ServeModel without parent resource InferenceService: %w", api.ErrBadRequest)
	}

	var inferenceServiceID *int32

	if inferenceServiceId != nil {
		var err error
		inferenceServiceID, err = apiutils.ValidateIDAsInt32Ptr(inferenceServiceId, "inference service")
		if err != nil {
			return nil, err
		}

		_, err = b.inferenceServiceRepository.GetByID(*inferenceServiceID)
		if err != nil {
			return nil, fmt.Errorf("no InferenceService found for id %d: %w", *inferenceServiceID, api.ErrNotFound)
		}
	}

	savedSrvModel, err := b.serveModelRepository.Save(srvModel, inferenceServiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("serve model with name %s already exists: %w", *serveModel.Name, api.ErrConflict)
		}

		return nil, err
	}

	toReturn, err := b.mapper.MapToServeModel(savedSrvModel)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetServeModelById(id string) (*openapi.ServeModel, error) {
	glog.Infof("Getting ServeModel by id %s", id)

	convertedId, err := apiutils.ValidateIDAsInt32(id, "serve model")
	if err != nil {
		return nil, err
	}
	serveModel, err := b.serveModelRepository.GetByID(convertedId)
	if err != nil {
		return nil, fmt.Errorf("no serve model found for id %s: %w", id, api.ErrNotFound)
	}

	toReturn, err := b.mapper.MapToServeModel(serveModel)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return toReturn, nil
}

func (b *ModelRegistryService) GetServeModels(listOptions api.ListOptions, inferenceServiceId *string) (*openapi.ServeModelList, error) {
	var inferenceServiceID *int32

	if inferenceServiceId != nil {
		var err error
		inferenceServiceID, err = apiutils.ValidateIDAsInt32Ptr(inferenceServiceId, "inference service")
		if err != nil {
			return nil, err
		}
	}

	serveModels, err := b.serveModelRepository.List(models.ServeModelListOptions{
		Pagination: models.Pagination{
			PageSize:      listOptions.PageSize,
			OrderBy:       listOptions.OrderBy,
			SortOrder:     listOptions.SortOrder,
			NextPageToken: listOptions.NextPageToken,
			FilterQuery:   listOptions.FilterQuery,
		},
		InferenceServiceID: inferenceServiceID,
	})
	if err != nil {
		return nil, err
	}

	serveModelList := &openapi.ServeModelList{
		Items: []openapi.ServeModel{},
	}

	for _, serveModel := range serveModels.Items {
		serveModel, err := b.mapper.MapToServeModel(serveModel)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		serveModelList.Items = append(serveModelList.Items, *serveModel)
	}

	serveModelList.NextPageToken = serveModels.NextPageToken
	serveModelList.PageSize = serveModels.PageSize
	serveModelList.Size = int32(serveModels.Size)

	return serveModelList, nil
}

// validateHFModelDeployment checks if a HuggingFace model can be deployed.
// It prevents deployment of gated or private models until authentication support is added.
func (b *ModelRegistryService) validateHFModelDeployment(modelVersionId string) error {
	// Get model version to check custom properties
	version, err := b.GetModelVersionById(modelVersionId)
	if err != nil {
		return err
	}

	// Check if this is a HuggingFace model by looking for HF-specific custom properties
	if version.CustomProperties == nil {
		return nil // Not an HF model, no validation needed
	}

	// Check if model is gated
	if gated, ok := version.CustomProperties["hf_gated"]; ok {
		if gated.MetadataStringValue != nil {
			gatedValue := gated.MetadataStringValue.StringValue
			// HF gated field can be "false", "auto", "manual", or true
			// Only allow deployment if explicitly "false" or empty
			if gatedValue != "false" && gatedValue != "" {
				return fmt.Errorf(
					"cannot deploy gated HuggingFace model: authentication not yet supported (model requires approval or authentication): %w",
					api.ErrBadRequest)
			}
		}
	}

	// Check if model is private
	if private, ok := version.CustomProperties["hf_private"]; ok {
		if private.MetadataStringValue != nil {
			privateValue := private.MetadataStringValue.StringValue
			if privateValue == "true" {
				return fmt.Errorf(
					"cannot deploy private HuggingFace model: authentication not yet supported: %w",
					api.ErrBadRequest)
			}
		}
	}

	return nil
}
