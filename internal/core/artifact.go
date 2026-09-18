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

// ensureArtifactName ensures that an artifact has a name during creation.
// If the artifact has no ID (creation) and no name, it generates a UUID.
func ensureArtifactName(artifact *openapi.Artifact) {
	if artifact == nil {
		return
	}

	// Helper function to generate name if needed
	generateNameIfNeeded := func(id *string, name **string) {
		if id == nil && *name == nil {
			// This is a create operation with no name, generate UUID
			*name = converter.GenerateNewName()
		}
	}

	if artifact.ModelArtifact != nil {
		generateNameIfNeeded(artifact.ModelArtifact.Id, &artifact.ModelArtifact.Name)
	} else if artifact.DocArtifact != nil {
		generateNameIfNeeded(artifact.DocArtifact.Id, &artifact.DocArtifact.Name)
	} else if artifact.DataSet != nil {
		generateNameIfNeeded(artifact.DataSet.Id, &artifact.DataSet.Name)
	} else if artifact.Metric != nil {
		generateNameIfNeeded(artifact.Metric.Id, &artifact.Metric.Name)
	} else if artifact.Parameter != nil {
		generateNameIfNeeded(artifact.Parameter.Id, &artifact.Parameter.Name)
	}
}

func (b *ModelRegistryService) upsertArtifact(artifact *openapi.Artifact, parentResourceId *string) (*openapi.Artifact, error) {
	var parentResourceIDPtr *int32

	artToReturn := &openapi.Artifact{}

	if artifact == nil {
		return nil, fmt.Errorf("invalid artifact pointer, cannot be nil: %w", api.ErrBadRequest)
	}

	// Ensure artifact has a name if it's being created
	ensureArtifactName(artifact)

	// Only convert parentResourceId to int32 if it's provided
	if parentResourceId != nil {
		var err error
		parentResourceIDPtr, err = apiutils.ValidateIDAsInt32Ptr(parentResourceId, "parent resource")
		if err != nil {
			return nil, err
		}
	}

	// Set experiment properties if the artifact is being linked to an experiment run
	if parentResourceId != nil {
		experimentRun, err := b.GetExperimentRunById(*parentResourceId)
		if err == nil {
			b.setExperimentPropertiesOnArtifact(artifact, experimentRun.ExperimentId, *parentResourceId)
		}
	}

	if ma := artifact.ModelArtifact; ma != nil {
		if ma.Id != nil {
			existing, err := b.getArtifact(*ma.Id)
			if err != nil {
				return nil, fmt.Errorf("mismatched types, artifact with id %s is not a model artifact: %w", *ma.Id, api.ErrBadRequest)
			}

			if existing.ModelArtifact == nil {
				return nil, fmt.Errorf("artifact with id %s is not a model artifact: %w", *ma.Id, api.ErrBadRequest)
			}

			withNotEditable, err := b.mapper.UpdateExistingModelArtifact(converter.NewOpenapiUpdateWrapper(existing.ModelArtifact, ma))
			if err != nil {
				return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
			}

			// Handle CustomProperties preservation for partial updates
			// If the update didn't specify CustomProperties (nil), preserve existing ones
			if ma.CustomProperties == nil && existing.ModelArtifact.CustomProperties != nil {
				withNotEditable.CustomProperties = existing.ModelArtifact.CustomProperties
			}

			ma = &withNotEditable
		}

		modelArtifact, err := b.mapper.MapFromModelArtifact(ma, parentResourceId)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		modelArtifact, err = b.modelArtifactRepository.Save(modelArtifact, parentResourceIDPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, fmt.Errorf("model artifact with name %s already exists: %w", *ma.Name, api.ErrConflict)
			}
			return nil, err
		}

		toReturn, err := b.mapper.MapToModelArtifact(modelArtifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		artToReturn.ModelArtifact = toReturn
		return artToReturn, nil
	} else if da := artifact.DocArtifact; da != nil {
		if da.Id != nil {
			existing, err := b.getArtifact(*da.Id)
			if err != nil {
				return nil, fmt.Errorf("mismatched types, artifact with id %s is not a doc artifact: %w", *da.Id, api.ErrBadRequest)
			}

			if existing.DocArtifact == nil {
				return nil, fmt.Errorf("artifact with id %s is not a doc artifact: %w", *da.Id, api.ErrBadRequest)
			}

			withNotEditable, err := b.mapper.UpdateExistingDocArtifact(converter.NewOpenapiUpdateWrapper(existing.DocArtifact, da))
			if err != nil {
				return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
			}

			// Handle CustomProperties preservation for partial updates
			// If the update didn't specify CustomProperties (nil), preserve existing ones
			if da.CustomProperties == nil && existing.DocArtifact.CustomProperties != nil {
				withNotEditable.CustomProperties = existing.DocArtifact.CustomProperties
			}

			da = &withNotEditable
		}

		docArtifact, err := b.mapper.MapFromDocArtifact(da, parentResourceId)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		docArtifact, err = b.docArtifactRepository.Save(docArtifact, parentResourceIDPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, fmt.Errorf("doc artifact with name %s already exists: %w", *da.Name, api.ErrConflict)
			}
			return nil, err
		}

		toReturn, err := b.mapper.MapToDocArtifact(docArtifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		artToReturn.DocArtifact = toReturn
		return artToReturn, nil
	} else if ds := artifact.DataSet; ds != nil {
		// Handle DataSet artifacts using embedmd converters
		if ds.Id != nil {
			existing, err := b.getArtifact(*ds.Id)
			if err != nil {
				return nil, fmt.Errorf("mismatched types, artifact with id %s is not a dataset artifact: %w", *ds.Id, api.ErrBadRequest)
			}

			if existing.DataSet == nil {
				return nil, fmt.Errorf("artifact with id %s is not a dataset artifact: %w", *ds.Id, api.ErrBadRequest)
			}

			withNotEditable, err := b.mapper.UpdateExistingDataSet(converter.NewOpenapiUpdateWrapper(existing.DataSet, ds))
			if err != nil {
				return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
			}

			// Handle CustomProperties preservation for partial updates
			// If the update didn't specify CustomProperties (nil), preserve existing ones
			if ds.CustomProperties == nil && existing.DataSet.CustomProperties != nil {
				withNotEditable.CustomProperties = existing.DataSet.CustomProperties
			}

			ds = &withNotEditable
		}

		dataSetEntity, err := b.mapper.MapFromDataSet(ds)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		dataSetEntity, err = b.dataSetRepository.Save(dataSetEntity, parentResourceIDPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, fmt.Errorf("dataset artifact with name %s already exists: %w", *ds.Name, api.ErrConflict)
			}
			return nil, err
		}

		toReturn, err := b.mapper.MapToDataSet(dataSetEntity)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		artToReturn.DataSet = toReturn
		return artToReturn, nil
	} else if me := artifact.Metric; me != nil {
		// Handle metric artifacts using embedmd converters
		// Validate that metric has a value
		if me.Value == nil {
			return nil, fmt.Errorf("metric value is required: %w", api.ErrBadRequest)
		}

		if me.Id != nil {
			existing, err := b.getArtifact(*me.Id)
			if err != nil {
				return nil, fmt.Errorf("mismatched types, artifact with id %s is not a metric artifact: %w", *me.Id, api.ErrBadRequest)
			}

			if existing.Metric == nil {
				return nil, fmt.Errorf("artifact with id %s is not a metric artifact: %w", *me.Id, api.ErrBadRequest)
			}

			withNotEditable, err := b.mapper.UpdateExistingMetric(converter.NewOpenapiUpdateWrapper(existing.Metric, me))
			if err != nil {
				return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
			}

			// Handle CustomProperties preservation for partial updates
			// If the update didn't specify CustomProperties (nil), preserve existing ones
			if me.CustomProperties == nil && existing.Metric.CustomProperties != nil {
				withNotEditable.CustomProperties = existing.Metric.CustomProperties
			}

			me = &withNotEditable
		} else {
			// For new metrics (no ID), check if a metric with the same name already exists
			// in the same parent context - similar to MLMD behavior
			if parentResourceId != nil && me.Name != nil {
				existing, err := b.getArtifactByParams(me.Name, parentResourceId, nil, string(openapi.ARTIFACTTYPEQUERYPARAM_METRIC))
				if err == nil && existing != nil && existing.Metric != nil {
					// Metric with same name exists, update it instead of creating new one
					withNotEditable, err := b.mapper.UpdateExistingMetric(converter.NewOpenapiUpdateWrapper(existing.Metric, me))
					if err != nil {
						return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
					}

					// Handle CustomProperties preservation for partial updates
					// If the update didn't specify CustomProperties (nil), preserve existing ones
					if me.CustomProperties == nil && existing.Metric.CustomProperties != nil {
						withNotEditable.CustomProperties = existing.Metric.CustomProperties
					}

					me = &withNotEditable
				} else if api.IgnoreNotFound(err) != nil {
					// Only return error if it's not a "not found" error
					return nil, fmt.Errorf("error checking for existing metric: %w", err)
				}
				// If not found, continue with creating new metric
			}
		}

		metricEntity, err := b.mapper.MapFromMetric(me, parentResourceId)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		metricEntity, err = b.metricRepository.Save(metricEntity, parentResourceIDPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, fmt.Errorf("metric artifact with name %s already exists: %w", *me.Name, api.ErrConflict)
			}
			return nil, err
		}

		toReturn, err := b.mapper.MapToMetricFromMetric(metricEntity)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		artToReturn.Metric = toReturn
		return artToReturn, nil
	} else if pa := artifact.Parameter; pa != nil {
		// ADD PARAMETER SUPPORT using embedmd converters
		if pa.Id != nil {
			existing, err := b.getArtifact(*pa.Id)
			if err != nil {
				return nil, fmt.Errorf("mismatched types, artifact with id %s is not a parameter artifact: %w", *pa.Id, api.ErrBadRequest)
			}

			if existing.Parameter == nil {
				return nil, fmt.Errorf("artifact with id %s is not a parameter artifact: %w", *pa.Id, api.ErrBadRequest)
			}

			withNotEditable, err := b.mapper.UpdateExistingParameter(converter.NewOpenapiUpdateWrapper(existing.Parameter, pa))
			if err != nil {
				return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
			}

			// Handle CustomProperties preservation for partial updates
			// If the update didn't specify CustomProperties (nil), preserve existing ones
			if pa.CustomProperties == nil && existing.Parameter.CustomProperties != nil {
				withNotEditable.CustomProperties = existing.Parameter.CustomProperties
			}

			pa = &withNotEditable
		} else {
			// For new parameters (no ID), check if a parameter with the same name already exists
			// in the same parent context - similar to metric behavior
			if parentResourceId != nil && pa.Name != nil {
				existing, err := b.getArtifactByParams(pa.Name, parentResourceId, nil, "parameter")
				if err == nil && existing != nil && existing.Parameter != nil {
					// Parameter with same name exists, update it instead of creating new one
					withNotEditable, err := b.mapper.UpdateExistingParameter(converter.NewOpenapiUpdateWrapper(existing.Parameter, pa))
					if err != nil {
						return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
					}

					// Handle CustomProperties preservation for partial updates
					// If the update didn't specify CustomProperties (nil), preserve existing ones
					if pa.CustomProperties == nil && existing.Parameter.CustomProperties != nil {
						withNotEditable.CustomProperties = existing.Parameter.CustomProperties
					}

					pa = &withNotEditable
				} else if api.IgnoreNotFound(err) != nil {
					// Only return error if it's not a "not found" error
					return nil, fmt.Errorf("error checking for existing parameter: %w", err)
				}
				// If not found, continue with creating new parameter
			}
		}

		parameterEntity, err := b.mapper.MapFromParameter(pa, parentResourceId)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		parameterEntity, err = b.parameterRepository.Save(parameterEntity, parentResourceIDPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, fmt.Errorf("parameter artifact with name %s already exists: %w", *pa.Name, api.ErrConflict)
			}
			return nil, err
		}

		toReturn, err := b.mapper.MapToParameter(parameterEntity)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}

		artToReturn.Parameter = toReturn
		return artToReturn, nil
	}

	return nil, fmt.Errorf("invalid artifact type, must be either ModelArtifact, DocArtifact, DataSet, Metric, or Parameter: %w", api.ErrBadRequest)
}

func (b *ModelRegistryService) UpsertModelVersionArtifact(artifact *openapi.Artifact, parentResourceId string) (*openapi.Artifact, error) {
	// Validate that the ModelVersion exists before creating the artifact
	_, err := b.GetModelVersionById(parentResourceId)
	if err != nil {
		return nil, err
	}

	return b.upsertArtifact(artifact, &parentResourceId)
}

func (b *ModelRegistryService) UpsertArtifact(artifact *openapi.Artifact) (*openapi.Artifact, error) {
	return b.upsertArtifact(artifact, nil)
}

func (b *ModelRegistryService) getArtifact(id string) (*openapi.Artifact, error) {
	artToReturn := &openapi.Artifact{}
	convertedId, err := apiutils.ValidateIDAsInt32(id, "artifact")
	if err != nil {
		return nil, err
	}

	artifact, err := b.artifactRepository.GetByID(convertedId)
	if err != nil {
		return nil, fmt.Errorf("no artifact found for id %s: %w", id, api.ErrNotFound)
	}

	if artifact.ModelArtifact != nil {
		toReturn, err := b.mapper.MapToModelArtifact(*artifact.ModelArtifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artToReturn.ModelArtifact = toReturn
	} else if artifact.DocArtifact != nil {
		toReturn, err := b.mapper.MapToDocArtifact(*artifact.DocArtifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artToReturn.DocArtifact = toReturn
	} else if artifact.DataSet != nil {
		toReturn, err := b.mapper.MapToDataSet(*artifact.DataSet)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artToReturn.DataSet = toReturn
	} else if artifact.Metric != nil {
		toReturn, err := b.mapper.MapToMetricFromMetric(*artifact.Metric)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artToReturn.Metric = toReturn
	} else if artifact.Parameter != nil {
		toReturn, err := b.mapper.MapToParameter(*artifact.Parameter)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artToReturn.Parameter = toReturn
	}

	return artToReturn, nil
}

func (b *ModelRegistryService) GetArtifactById(id string) (*openapi.Artifact, error) {
	return b.getArtifact(id)
}

func (b *ModelRegistryService) getArtifactByParams(artifactName *string, parentResourceId *string, externalId *string, artifactType string) (*openapi.Artifact, error) {
	if (artifactName == nil || parentResourceId == nil) && externalId == nil {
		return nil, fmt.Errorf("invalid parameters call, supply either (artifactName and parentResourceId), or externalId: %w", api.ErrBadRequest)
	}

	var parentResourceID *int32
	if parentResourceId != nil {
		var err error
		parentResourceID, err = apiutils.ValidateIDAsInt32Ptr(parentResourceId, "parent resource")
		if err != nil {
			return nil, err
		}
	}

	artifacts, err := b.artifactRepository.List(models.ArtifactListOptions{
		Name:             artifactName,
		ExternalID:       externalId,
		ParentResourceID: parentResourceID,
	})
	if err != nil {
		return nil, err
	}

	if artifactType != "" {
		artifactType += " "
	}

	if len(artifacts.Items) == 0 {
		return nil, fmt.Errorf("no %sartifacts found for name=%v, parentResourceId=%v, externalId=%v: %w", artifactType, apiutils.ZeroIfNil(artifactName), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	if len(artifacts.Items) > 1 {
		return nil, fmt.Errorf("multiple %sartifacts found for name=%v, parentResourceId=%v, externalId=%v: %w", artifactType, apiutils.ZeroIfNil(artifactName), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	mappedArtifact, err := b.mapper.MapToArtifact(artifacts.Items[0])
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
	}

	return mappedArtifact, nil
}

func (b *ModelRegistryService) GetArtifactByParams(artifactName *string, parentResourceId *string, externalId *string) (*openapi.Artifact, error) {
	return b.getArtifactByParams(artifactName, parentResourceId, externalId, "")
}

func (b *ModelRegistryService) GetArtifacts(artifactType openapi.ArtifactTypeQueryParam, listOptions api.ListOptions, parentResourceId *string) (*openapi.ArtifactList, error) {
	var parentResourceIDPtr *int32

	if parentResourceId != nil {
		var err error
		parentResourceIDPtr, err = apiutils.ValidateIDAsInt32Ptr(parentResourceId, "parent resource")
		if err != nil {
			return nil, err
		}
	}

	// Convert artifactType parameter to string if provided
	var artifactTypeStr *string
	if artifactType != "" {
		artifactTypeStr = (*string)(&artifactType)
	}

	artifacts, err := b.artifactRepository.List(models.ArtifactListOptions{
		Pagination: models.Pagination{
			PageSize:      listOptions.PageSize,
			OrderBy:       listOptions.OrderBy,
			SortOrder:     listOptions.SortOrder,
			NextPageToken: listOptions.NextPageToken,
			FilterQuery:   listOptions.FilterQuery,
		},
		ParentResourceID: parentResourceIDPtr,
		ArtifactType:     artifactTypeStr,
	})
	if err != nil {
		return nil, err
	}

	artifactsList := &openapi.ArtifactList{
		Items: []openapi.Artifact{},
	}

	for _, artifact := range artifacts.Items {
		mappedArtifact, err := b.mapper.MapToArtifact(artifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		artifactsList.Items = append(artifactsList.Items, *mappedArtifact)
	}

	artifactsList.NextPageToken = artifacts.NextPageToken
	artifactsList.PageSize = artifacts.PageSize
	artifactsList.Size = int32(artifacts.Size)

	return artifactsList, nil
}

func (b *ModelRegistryService) UpsertModelArtifact(modelArtifact *openapi.ModelArtifact) (*openapi.ModelArtifact, error) {
	if modelArtifact == nil {
		return nil, fmt.Errorf("invalid model artifact pointer, can't upsert nil: %w", api.ErrBadRequest)
	}

	art, err := b.UpsertArtifact(&openapi.Artifact{
		ModelArtifact: modelArtifact,
	})
	if err != nil {
		return nil, err
	}

	return art.ModelArtifact, nil
}

func (b *ModelRegistryService) GetModelArtifactById(id string) (*openapi.ModelArtifact, error) {
	art, err := b.GetArtifactById(id)
	if err != nil {
		return nil, err
	}

	if art.ModelArtifact == nil {
		return nil, fmt.Errorf("artifact with id %s is not a model artifact: %w", id, api.ErrNotFound)
	}

	return art.ModelArtifact, nil
}

func (b *ModelRegistryService) GetModelArtifactByInferenceService(inferenceServiceId string) (*openapi.ModelArtifact, error) {
	mv, err := b.GetModelVersionByInferenceService(inferenceServiceId)
	if err != nil {
		return nil, err
	}

	artifactList, err := b.GetArtifacts(openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT, api.ListOptions{}, mv.Id)
	if err != nil {
		return nil, err
	}

	if len(artifactList.Items) == 0 {
		return nil, fmt.Errorf("no artifacts found for model version %s: %w", *mv.Id, api.ErrNotFound)
	}

	if artifactList.Items[0].ModelArtifact == nil {
		return nil, fmt.Errorf("no artifacts found for model version %s: %w", *mv.Id, api.ErrNotFound)
	}

	return artifactList.Items[0].ModelArtifact, nil
}

func (b *ModelRegistryService) GetModelArtifactByParams(artifactName *string, parentResourceId *string, externalId *string) (*openapi.ModelArtifact, error) {
	art, err := b.getArtifactByParams(artifactName, parentResourceId, externalId, "model")
	if err != nil {
		return nil, err
	}

	if art.ModelArtifact == nil {
		return nil, fmt.Errorf("artifact with name=%v, parentResourceId=%v, externalId=%v is not a model artifact: %w", apiutils.ZeroIfNil(artifactName), apiutils.ZeroIfNil(parentResourceId), apiutils.ZeroIfNil(externalId), api.ErrNotFound)
	}

	return art.ModelArtifact, nil
}

func (b *ModelRegistryService) GetModelArtifacts(listOptions api.ListOptions, parentResourceId *string) (*openapi.ModelArtifactList, error) {
	var parentResourceIDPtr *int32

	if parentResourceId != nil {
		var err error
		parentResourceIDPtr, err = apiutils.ValidateIDAsInt32Ptr(parentResourceId, "parent resource")
		if err != nil {
			return nil, err
		}
	}

	modelArtifacts, err := b.modelArtifactRepository.List(models.ModelArtifactListOptions{
		Pagination: models.Pagination{
			PageSize:      listOptions.PageSize,
			OrderBy:       listOptions.OrderBy,
			SortOrder:     listOptions.SortOrder,
			NextPageToken: listOptions.NextPageToken,
			FilterQuery:   listOptions.FilterQuery,
		},
		ParentResourceID: parentResourceIDPtr,
	})
	if err != nil {
		return nil, err
	}

	modelArtifactList := &openapi.ModelArtifactList{
		Items: []openapi.ModelArtifact{},
	}

	for _, modelArtifact := range modelArtifacts.Items {
		modelArtifact, err := b.mapper.MapToModelArtifact(modelArtifact)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", err, api.ErrBadRequest)
		}
		modelArtifactList.Items = append(modelArtifactList.Items, *modelArtifact)
	}

	modelArtifactList.NextPageToken = modelArtifacts.NextPageToken
	modelArtifactList.PageSize = modelArtifacts.PageSize
	modelArtifactList.Size = int32(modelArtifacts.Size)

	return modelArtifactList, nil
}

func setExperimentPropertiesOnCustomProperties(customProps *map[string]openapi.MetadataValue, experimentId, experimentRunId string) {
	if *customProps == nil {
		*customProps = map[string]openapi.MetadataValue{}
	}

	(*customProps)["experiment_id"] = openapi.MetadataValue{
		MetadataIntValue: &openapi.MetadataIntValue{
			IntValue:     experimentId,
			MetadataType: "MetadataIntValue",
		},
	}
	(*customProps)["experiment_run_id"] = openapi.MetadataValue{
		MetadataIntValue: &openapi.MetadataIntValue{
			IntValue:     experimentRunId,
			MetadataType: "MetadataIntValue",
		},
	}
}

// setExperimentPropertiesOnArtifact is a helper function that sets experiment_id and experiment_run_id
// both as direct fields (for API response) and as custom properties (for database storage and filterQuery)
func (b *ModelRegistryService) setExperimentPropertiesOnArtifact(artifact *openapi.Artifact, experimentId, experimentRunId string) {
	// Define a helper function to reduce repetition
	setProperties := func(experimentIdPtr, experimentRunIdPtr **string, customProps *map[string]openapi.MetadataValue) {
		*experimentIdPtr = &experimentId
		*experimentRunIdPtr = &experimentRunId
		setExperimentPropertiesOnCustomProperties(customProps, experimentId, experimentRunId)
	}

	if artifact.ModelArtifact != nil {
		setProperties(&artifact.ModelArtifact.ExperimentId, &artifact.ModelArtifact.ExperimentRunId, &artifact.ModelArtifact.CustomProperties)
	}
	if artifact.DocArtifact != nil {
		setProperties(&artifact.DocArtifact.ExperimentId, &artifact.DocArtifact.ExperimentRunId, &artifact.DocArtifact.CustomProperties)
	}
	if artifact.DataSet != nil {
		setProperties(&artifact.DataSet.ExperimentId, &artifact.DataSet.ExperimentRunId, &artifact.DataSet.CustomProperties)
	}
	if artifact.Metric != nil {
		setProperties(&artifact.Metric.ExperimentId, &artifact.Metric.ExperimentRunId, &artifact.Metric.CustomProperties)
	}
	if artifact.Parameter != nil {
		setProperties(&artifact.Parameter.ExperimentId, &artifact.Parameter.ExperimentRunId, &artifact.Parameter.CustomProperties)
	}
}
