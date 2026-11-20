package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/internal/db/filter"
	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"github.com/kubeflow/hub/internal/defaults"
	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"gorm.io/gorm"
)

var ErrArtifactNotFound = errors.New("artifact by id not found")

type ArtifactRepositoryImpl struct {
	db       *gorm.DB
	idToName map[int32]string
	nameToID datastore.ArtifactTypeMap
}

func NewArtifactRepository(db *gorm.DB, artifactTypes datastore.ArtifactTypeMap) models.ArtifactRepository {
	idToName := make(map[int32]string, len(artifactTypes))
	for name, id := range artifactTypes {
		idToName[id] = name
	}

	return &ArtifactRepositoryImpl{
		db:       db,
		nameToID: artifactTypes,
		idToName: idToName,
	}
}

func (r *ArtifactRepositoryImpl) GetByID(id int32) (models.Artifact, error) {
	artifact := &schema.Artifact{}
	properties := []schema.ArtifactProperty{}

	if err := r.db.Where("id = ?", id).First(artifact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Artifact{}, fmt.Errorf("%w: %v", ErrArtifactNotFound, err)
		}

		return models.Artifact{}, fmt.Errorf("error getting artifact by id: %w", err)
	}

	if err := r.db.Where("artifact_id = ?", artifact.ID).Find(&properties).Error; err != nil {
		return models.Artifact{}, fmt.Errorf("error getting properties by artifact id: %w", err)
	}

	// Use the same logic as mapDataLayerToArtifact to handle all artifact types
	mappedArtifact, err := r.mapDataLayerToArtifact(*artifact, properties)
	if err != nil {
		return models.Artifact{}, fmt.Errorf("error mapping artifact: %w", err)
	}

	return mappedArtifact, nil
}

func (r *ArtifactRepositoryImpl) List(listOptions models.ArtifactListOptions) (*models.ListWrapper[models.Artifact], error) {
	list := models.ListWrapper[models.Artifact]{
		PageSize: listOptions.GetPageSize(),
	}

	artifacts := []models.Artifact{}
	artifactsArt := []schema.Artifact{}

	query := r.db.Model(&schema.Artifact{})

	// Exclude metric history records - they should only be returned via metric history endpoints
	if metricHistoryTypeID, ok := r.nameToID[defaults.MetricHistoryTypeName]; ok {
		query = query.Where("type_id != ?", metricHistoryTypeID)
	}

	if listOptions.Name != nil {
		// Name is not prefixed with the parent resource id to allow for filtering by name only
		// Parent resource Id is used later to filter by Attribution.context_id
		query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", escapeLike(*listOptions.Name)))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	// Filter by artifact type if specified
	if listOptions.ArtifactType != nil {
		// Handle "null" string as invalid artifact type
		if *listOptions.ArtifactType == "null" || *listOptions.ArtifactType == "" {
			return nil, fmt.Errorf("invalid artifact type: empty or null value provided: %w", api.ErrBadRequest)
		}
		typeID, err := r.getTypeIDFromArtifactType(*listOptions.ArtifactType)
		if err != nil {
			return nil, fmt.Errorf("invalid artifact type %s: %w", *listOptions.ArtifactType, api.ErrBadRequest)
		}
		query = query.Where(utils.GetTableName(r.db, &schema.Artifact{})+".type_id = ?", typeID)
	}

	query, err := applyFilterQuery(query, &listOptions, filter.DefaultEntityMappingFuncs())
	if err != nil {
		return nil, err
	}

	if listOptions.ParentResourceID != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildAttributionJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Attribution{}, "context_id")+" = ?", listOptions.ParentResourceID).
			Select(utils.GetTableName(query, &schema.Artifact{}) + ".*") // Explicitly select from Artifact table to avoid ambiguity
		// Use table-prefixed pagination to avoid column ambiguity
		query = query.Scopes(scopes.PaginateWithTablePrefix(artifacts, &listOptions.Pagination, r.db, "Artifact"))
	} else {
		query = query.Scopes(scopes.Paginate(artifacts, &listOptions.Pagination, r.db))
	}

	if err := query.Find(&artifactsArt).Error; err != nil {
		return nil, fmt.Errorf("error listing artifacts: %w", err)
	}

	hasMore := false
	pageSize := listOptions.GetPageSize()
	if pageSize > 0 {
		hasMore = len(artifactsArt) > int(pageSize)
		if hasMore {
			artifactsArt = artifactsArt[:len(artifactsArt)-1]
		}
	}

	for _, artifactArt := range artifactsArt {
		properties := []schema.ArtifactProperty{}
		if err := r.db.Where("artifact_id = ?", artifactArt.ID).Find(&properties).Error; err != nil {
			return nil, fmt.Errorf("error getting properties by artifact id: %w", err)
		}

		artifact, err := r.mapDataLayerToArtifact(artifactArt, properties)
		if err != nil {
			return nil, fmt.Errorf("error mapping artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}

	if hasMore && len(artifactsArt) > 0 {
		lastArtifact := artifactsArt[len(artifactsArt)-1]
		orderBy := listOptions.GetOrderBy()
		value := ""
		if orderBy != "" {
			switch orderBy {
			case "ID":
				value = fmt.Sprintf("%d", lastArtifact.ID)
			case "CREATE_TIME":
				value = fmt.Sprintf("%d", lastArtifact.CreateTimeSinceEpoch)
			case "LAST_UPDATE_TIME":
				value = fmt.Sprintf("%d", lastArtifact.LastUpdateTimeSinceEpoch)
			default:
				value = fmt.Sprintf("%d", lastArtifact.ID)
			}
		}
		nextToken := scopes.CreateNextPageToken(lastArtifact.ID, value)
		listOptions.NextPageToken = &nextToken
	} else {
		listOptions.NextPageToken = nil
	}

	list.Items = artifacts
	list.NextPageToken = listOptions.GetNextPageToken()
	list.PageSize = listOptions.GetPageSize()
	list.Size = int32(len(artifacts))

	return &list, nil
}

// getTypeIDFromArtifactType maps artifact type strings to their corresponding type IDs
func (r *ArtifactRepositoryImpl) getTypeIDFromArtifactType(artifactType string) (int32, error) {
	switch openapi.ArtifactTypeQueryParam(artifactType) {
	case openapi.ARTIFACTTYPEQUERYPARAM_MODEL_ARTIFACT:
		return r.nameToID[defaults.ModelArtifactTypeName], nil
	case openapi.ARTIFACTTYPEQUERYPARAM_DOC_ARTIFACT:
		return r.nameToID[defaults.DocArtifactTypeName], nil
	case openapi.ARTIFACTTYPEQUERYPARAM_DATASET_ARTIFACT:
		return r.nameToID[defaults.DataSetTypeName], nil
	case openapi.ARTIFACTTYPEQUERYPARAM_METRIC:
		return r.nameToID[defaults.MetricTypeName], nil
	case openapi.ARTIFACTTYPEQUERYPARAM_PARAMETER:
		return r.nameToID[defaults.ParameterTypeName], nil
	default:
		return 0, fmt.Errorf("unsupported artifact type: %s: %w", artifactType, api.ErrBadRequest)
	}
}

func (r *ArtifactRepositoryImpl) mapDataLayerToArtifact(artifact schema.Artifact, properties []schema.ArtifactProperty) (models.Artifact, error) {
	artToReturn := models.Artifact{}

	typeName := r.idToName[artifact.TypeID]

	switch typeName {
	case defaults.ModelArtifactTypeName:
		modelArtifact := mapDataLayerToModelArtifact(artifact, properties)
		artToReturn.ModelArtifact = &modelArtifact
	case defaults.DocArtifactTypeName:
		docArtifact := mapDataLayerToDocArtifact(artifact, properties)
		artToReturn.DocArtifact = &docArtifact
	case defaults.DataSetTypeName:
		dataSet := mapDataLayerToDataSet(artifact, properties)
		artToReturn.DataSet = &dataSet
	case defaults.MetricTypeName:
		metric := mapDataLayerToMetric(artifact, properties)
		artToReturn.Metric = &metric
	case defaults.ParameterTypeName:
		parameter := mapDataLayerToParameter(artifact, properties)
		artToReturn.Parameter = &parameter
	default:
		return models.Artifact{}, fmt.Errorf("invalid artifact type: %s=%d (expected: %v)", typeName, artifact.TypeID, r.idToName)
	}

	return artToReturn, nil
}
