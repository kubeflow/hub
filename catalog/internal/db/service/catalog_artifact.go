package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/db/models"
	catpagination "github.com/kubeflow/hub/catalog/internal/db/pagination"
	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"github.com/kubeflow/hub/pkg/api"
	"gorm.io/gorm"
)

var ErrCatalogArtifactNotFound = errors.New("catalog artifact by id not found")

// isValidPropertyName validates basic property name constraints
func isValidPropertyName(name string) bool {
	// Empty names are not valid
	if name == "" {
		return false
	}
	// Check length (reasonable limit to prevent abuse)
	if len(name) > 255 {
		return false
	}

	return true
}

type CatalogArtifactRepositoryImpl struct {
	db       *gorm.DB
	idToName map[int32]string
	nameToID datastore.ArtifactTypeMap
}

func NewCatalogArtifactRepository(db *gorm.DB, artifactTypes datastore.ArtifactTypeMap) models.CatalogArtifactRepository {
	idToName := make(map[int32]string, len(artifactTypes))
	for name, id := range artifactTypes {
		idToName[id] = name
	}

	return &CatalogArtifactRepositoryImpl{
		db:       db,
		nameToID: artifactTypes,
		idToName: idToName,
	}
}

func (r *CatalogArtifactRepositoryImpl) GetByID(id int32) (models.CatalogArtifact, error) {
	artifact := &schema.Artifact{}
	properties := []schema.ArtifactProperty{}

	if err := r.db.Where("id = ?", id).First(artifact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CatalogArtifact{}, fmt.Errorf("%w: %v", ErrCatalogArtifactNotFound, err)
		}
		return models.CatalogArtifact{}, fmt.Errorf("error getting catalog artifact by id: %w", err)
	}

	if err := r.db.Where("artifact_id = ?", artifact.ID).Find(&properties).Error; err != nil {
		return models.CatalogArtifact{}, fmt.Errorf("error getting properties by artifact id: %w", err)
	}

	// Use the same logic as mapDataLayerToCatalogArtifact to handle artifact types
	mappedArtifact, err := r.mapDataLayerToCatalogArtifact(*artifact, properties)
	if err != nil {
		return models.CatalogArtifact{}, fmt.Errorf("error mapping catalog artifact: %w", err)
	}

	return mappedArtifact, nil
}

// List retrieves catalog artifacts with support for filtering, pagination, and custom property ordering.
//
// The method handles ordering in the following priority:
// 1. NAME - Special catalog-specific ordering
// 2. Standard columns (ID, CREATE_TIME, LAST_UPDATE_TIME) - Uses allowed column map
// 3. Custom properties (e.g., accuracy.double_value) - Dynamic property-based ordering
// 4. Fallback to ID ordering for invalid or unrecognized inputs
func (r *CatalogArtifactRepositoryImpl) List(listOptions models.CatalogArtifactListOptions) (*dbmodels.ListWrapper[models.CatalogArtifact], error) {
	list := dbmodels.ListWrapper[models.CatalogArtifact]{
		PageSize: listOptions.GetPageSize(),
	}

	artifacts := []models.CatalogArtifact{}
	artifactsArt := []schema.Artifact{}

	query := r.db.Model(&schema.Artifact{})

	// Apply filters similar to the internal artifact service
	if listOptions.Name != nil {
		// Name is not prefixed with the parent resource id to allow for filtering by name only
		// Parent resource Id is used later to filter by Attribution.context_id
		query = query.Where("name LIKE ?", fmt.Sprintf("%%:%s", *listOptions.Name))
	} else if listOptions.ExternalID != nil {
		query = query.Where("external_id = ?", listOptions.ExternalID)
	}

	// Filter by artifact type(s) if specified
	if len(listOptions.ArtifactTypesFilter) > 0 {
		// Handle multiple artifact types
		typeIDs := []int32{}
		for _, artifactType := range listOptions.ArtifactTypesFilter {
			// Handle "null" string as invalid artifact type
			if artifactType == "null" || artifactType == "" {
				return nil, fmt.Errorf("invalid artifact type: empty or null value provided: %w", api.ErrBadRequest)
			}
			typeID, err := r.getTypeIDFromArtifactType(artifactType)
			if err != nil {
				return nil, fmt.Errorf("invalid catalog artifact type %s: %w", artifactType, err)
			}
			typeIDs = append(typeIDs, typeID)
		}
		query = query.Where("type_id IN ?", typeIDs)
	} else if listOptions.ArtifactType != nil {
		// Handle single artifact type for backward compatibility
		// Handle "null" string as invalid artifact type
		if *listOptions.ArtifactType == "null" || *listOptions.ArtifactType == "" {
			return nil, fmt.Errorf("invalid artifact type: empty or null value provided: %w", api.ErrBadRequest)
		}
		typeID, err := r.getTypeIDFromArtifactType(*listOptions.ArtifactType)
		if err != nil {
			return nil, fmt.Errorf("invalid catalog artifact type %s: %w", *listOptions.ArtifactType, err)
		}
		query = query.Where("type_id = ?", typeID)
	} else {
		// Only include catalog artifact types
		catalogTypeIDs := []int32{}
		for _, typeID := range r.nameToID {
			catalogTypeIDs = append(catalogTypeIDs, typeID)
		}
		query = query.Where("type_id IN ?", catalogTypeIDs)
	}

	// Apply parent resource filtering if specified
	if listOptions.ParentResourceID != nil {
		// Proper GORM JOIN: Use helper that respects naming strategy
		query = query.Joins(utils.BuildAttributionJoin(query)).
			Where(utils.GetColumnRef(query, &schema.Attribution{}, "context_id")+" = ?", listOptions.ParentResourceID).
			Select(utils.GetTableName(query, &schema.Artifact{}) + ".*") // Explicitly select from Artifact table to avoid ambiguity
	}

	// Apply advanced filter query if supported
	var err error
	query, err = service.ApplyFilterQuery(query, &listOptions, NewCatalogArtifactEntityMappings())
	if err != nil {
		return nil, err
	}

	orderBy := listOptions.GetOrderBy()
	sortOrder := listOptions.GetSortOrder()
	nextPageToken := listOptions.GetNextPageToken()
	pageSize := listOptions.GetPageSize()

	// Handle NAME ordering specially (catalog-specific) to avoid string-to-integer cast issues
	if orderBy == "NAME" {
		artifactTable := utils.GetTableName(query, &schema.Artifact{})
		query = catpagination.ApplyNameOrdering(query, artifactTable, sortOrder, nextPageToken, pageSize, false)
	} else if _, isAllowedColumn := catpagination.CatalogOrderByColumns[orderBy]; isAllowedColumn {
		// Handle standard allowed columns (ID, CREATE_TIME, LAST_UPDATE_TIME)
		pagination := &dbmodels.Pagination{
			PageSize:      &pageSize,
			OrderBy:       &orderBy,
			SortOrder:     &sortOrder,
			NextPageToken: &nextPageToken,
		}

		// Use catalog-specific allowed columns
		query = query.Scopes(scopes.PaginateWithOptions(artifactsArt, pagination, r.db, "Artifact", catpagination.CatalogOrderByColumns))
	} else {
		// Assume it's a custom property ordering (e.g., accuracy.double_value, timestamp.string_value)
		query, err = r.applyCustomOrdering(query, &listOptions)
		if err != nil {
			return nil, err
		}
	}

	if err := query.Find(&artifactsArt).Error; err != nil {
		// Sanitize database errors to avoid exposing internal details to users
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error listing catalog artifacts: %w", err)
	}

	hasMore := false
	if pageSize > 0 {
		hasMore = len(artifactsArt) > int(pageSize)
		if hasMore {
			artifactsArt = artifactsArt[:len(artifactsArt)-1] // Remove the extra item used for hasMore detection
		}
	}

	// Map each artifact with its properties
	for _, artifactArt := range artifactsArt {
		properties := []schema.ArtifactProperty{}
		if err := r.db.Where("artifact_id = ?", artifactArt.ID).Find(&properties).Error; err != nil {
			return nil, fmt.Errorf("error getting properties by artifact id: %w", err)
		}

		artifact, err := r.mapDataLayerToCatalogArtifact(artifactArt, properties)
		if err != nil {
			return nil, fmt.Errorf("error mapping catalog artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}

	// Handle pagination token - generate token when there are more pages
	if hasMore && len(artifactsArt) > 0 {
		// Use the last artifact to generate pagination token
		lastArtifact := artifactsArt[len(artifactsArt)-1]
		nextToken := r.createPaginationToken(lastArtifact, listOptions)
		listOptions.NextPageToken = &nextToken
	} else {
		listOptions.NextPageToken = nil
	}

	list.Items = artifacts
	list.NextPageToken = listOptions.GetNextPageToken()
	list.Size = int32(len(artifacts))

	return &list, nil
}

// getTypeIDFromArtifactType maps catalog artifact type strings to their corresponding type IDs
func (r *CatalogArtifactRepositoryImpl) getTypeIDFromArtifactType(artifactType string) (int32, error) {
	switch artifactType {
	case "model-artifact":
		return r.nameToID[CatalogModelArtifactTypeName], nil
	case "metrics-artifact":
		return r.nameToID[CatalogMetricsArtifactTypeName], nil
	default:
		return 0, fmt.Errorf("unsupported catalog artifact type: %s", artifactType)
	}
}

func (r *CatalogArtifactRepositoryImpl) mapDataLayerToCatalogArtifact(artifact schema.Artifact, properties []schema.ArtifactProperty) (models.CatalogArtifact, error) {
	artToReturn := models.CatalogArtifact{}

	typeName := r.idToName[artifact.TypeID]

	mapper, exists := models.GetArtifactMapper(typeName)
	if !exists {
		return models.CatalogArtifact{}, fmt.Errorf("no mapper registered for artifact type: %s", typeName)
	}

	mapped := mapper(artifact, properties)

	switch typeName {
	case CatalogModelArtifactTypeName:
		if entity, ok := mapped.(models.CatalogArtifactEntity); ok {
			artToReturn.CatalogModelArtifact = entity
		} else {
			return models.CatalogArtifact{}, fmt.Errorf("mapper for %s returned incompatible type", typeName)
		}
	case CatalogMetricsArtifactTypeName:
		if entity, ok := mapped.(models.CatalogArtifactEntity); ok {
			artToReturn.CatalogMetricsArtifact = entity
		} else {
			return models.CatalogArtifact{}, fmt.Errorf("mapper for %s returned incompatible type", typeName)
		}
	default:
		return models.CatalogArtifact{}, fmt.Errorf("invalid catalog artifact type: %s=%d (expected: %v)", typeName, artifact.TypeID, r.idToName)
	}

	return artToReturn, nil
}

// createPaginationToken generates a pagination token based on the last artifact and ordering
func (r *CatalogArtifactRepositoryImpl) createPaginationToken(artifact schema.Artifact, listOptions models.CatalogArtifactListOptions) string {
	orderBy := listOptions.GetOrderBy()

	// Handle NAME ordering (catalog-specific)
	if orderBy == "NAME" {
		return catpagination.CreateNamePaginationToken(artifact.ID, artifact.Name)
	}

	// Handle custom property ordering
	sortValueQuery, column, err := r.sortValueQuery(&listOptions)
	if err != nil {
		// If there's an error in the sort value query (e.g., invalid value type),
		// fall back to ID ordering for the token
		// Note: This shouldn't normally happen as the error would be caught earlier in List()
		glog.Warningf("Error in sortValueQuery during pagination token creation: %v", err)
	} else if sortValueQuery != nil {
		artifactTable := utils.GetTableName(r.db, &schema.Artifact{})
		sortValueQuery = sortValueQuery.Where(artifactTable+".id=?", artifact.ID)

		var result struct {
			IntValue    *int64   `gorm:"int_value"`
			DoubleValue *float64 `gorm:"double_value"`
			StringValue *string  `gorm:"string_value"`
		}
		err := sortValueQuery.Scan(&result).Error
		if err != nil {
			// Log warning and fall back to default
			glog.Warningf("Failed to get sort value for pagination token: %v", err)
		} else {
			switch column {
			case "int_value":
				return scopes.CreateNextPageToken(artifact.ID, result.IntValue)
			case "double_value":
				return scopes.CreateNextPageToken(artifact.ID, result.DoubleValue)
			case "string_value":
				return scopes.CreateNextPageToken(artifact.ID, result.StringValue)
			}
		}
	}

	// Standard ordering fields
	value := ""
	switch orderBy {
	case "ID":
		value = fmt.Sprintf("%d", artifact.ID)
	case "CREATE_TIME":
		value = fmt.Sprintf("%d", artifact.CreateTimeSinceEpoch)
	case "LAST_UPDATE_TIME":
		value = fmt.Sprintf("%d", artifact.LastUpdateTimeSinceEpoch)
	default:
		// Default to ID ordering
		value = fmt.Sprintf("%d", artifact.ID)
	}

	return scopes.CreateNextPageToken(artifact.ID, value)
}

// sortValueQuery returns a query that will produce the value to sort on for
// the List response. The returned string is the column name.
//
// If the sort does not require a subquery, sortValueQuery returns nil, "".
// If the format is correct but the value type is invalid, returns nil, "" and an error.
func (r *CatalogArtifactRepositoryImpl) sortValueQuery(listOptions *models.CatalogArtifactListOptions, extraColumns ...any) (*gorm.DB, string, error) {
	db := r.db
	artifactTable := utils.GetTableName(db, &schema.Artifact{})

	query := db.Table(artifactTable)

	orderBy := strings.Split(listOptions.GetOrderBy(), ".")

	var valueColumn string

	// Handle <property>.<value_column> e.g. accuracy.double_value, timestamp.string_value
	if len(orderBy) == 2 {
		propertyName := orderBy[0]
		valueColumn = orderBy[1]

		switch valueColumn {
		case "int_value", "double_value", "string_value":
			// OK - valid value type
		default:
			// Invalid value type - return error immediately
			return nil, "", fmt.Errorf("invalid custom property value type '%s': must be one of 'int_value', 'double_value', or 'string_value': %w", valueColumn, api.ErrBadRequest)
		}

		if !isValidPropertyName(propertyName) {
			return nil, "", fmt.Errorf("invalid custom property name '%s': %w", propertyName, api.ErrBadRequest)
		}

		propertyTable := utils.GetTableName(db, &schema.ArtifactProperty{})
		query = query.
			Select(fmt.Sprintf("max(%s.%s) AS %s", propertyTable, valueColumn, valueColumn), extraColumns...).
			Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id=%s.artifact_id AND %s.name=?", propertyTable, artifactTable, propertyTable, propertyTable), propertyName)

		return query, valueColumn, nil
	}

	// Standard sort will work (not a custom property format)
	return nil, "", nil
}

// applyCustomOrdering applies custom ordering logic for non-standard orderBy field
func (r *CatalogArtifactRepositoryImpl) applyCustomOrdering(query *gorm.DB, listOptions *models.CatalogArtifactListOptions) (*gorm.DB, error) {
	db := r.db
	artifactTable := utils.GetTableName(db, &schema.Artifact{})
	orderBy := listOptions.GetOrderBy()

	// Handle NAME ordering specially (catalog-specific)
	if orderBy == "NAME" {
		return catpagination.ApplyNameOrdering(query, artifactTable, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false), nil
	}

	subquery, sortColumn, err := r.sortValueQuery(listOptions, artifactTable+".id")
	if err != nil {
		// Error in custom property format (e.g., invalid value type)
		return nil, err
	}
	if subquery == nil {
		// Fall back to standard pagination with catalog-specific allowed columns
		// If the orderBy is not in catpagination.CatalogOrderByColumns, PaginateWithOptions will default to ID ordering
		// This handles invalid custom property formats (e.g., "accuracy" without ".double_value")
		pageSize := listOptions.GetPageSize()
		sortOrder := listOptions.GetSortOrder()
		nextPageToken := listOptions.GetNextPageToken()
		pagination := &dbmodels.Pagination{
			PageSize:      &pageSize,
			OrderBy:       &orderBy,
			SortOrder:     &sortOrder,
			NextPageToken: &nextPageToken,
		}
		return query.Scopes(scopes.PaginateWithOptions([]schema.Artifact{}, pagination, r.db, "Artifact", catpagination.CatalogOrderByColumns)), nil
	}
	subquery = subquery.Group(artifactTable + ".id")

	// Join the main query with the subquery
	query = query.
		Joins(fmt.Sprintf("LEFT JOIN (?) sort_value ON %s.id=sort_value.id", artifactTable), subquery)

	// Apply sorting order
	sortOrder := listOptions.GetSortOrder()
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}
	query = query.Order(fmt.Sprintf("sort_value.%s %s NULLS LAST, %s.id", sortColumn, sortOrder, artifactTable))

	// Handle cursor-based pagination with nextPageToken
	nextPageToken := listOptions.GetNextPageToken()
	if nextPageToken != "" {
		// Parse the cursor from the token
		if cursor, err := scopes.DecodeCursor(nextPageToken); err == nil {
			// Apply WHERE clause for cursor-based pagination
			query = r.applyCursorPagination(query, cursor, sortColumn, sortOrder)
		}
		// If token parsing fails, fall back to no cursor (first page)
	}

	// Apply pagination limit
	pageSize := listOptions.GetPageSize()
	if pageSize > 0 {
		query = query.Limit(int(pageSize) + 1) // +1 to detect if there are more pages
	}

	return query, nil
}

// applyCursorPagination applies WHERE clause for cursor-based pagination with custom property sorting
func (r *CatalogArtifactRepositoryImpl) applyCursorPagination(query *gorm.DB, cursor *scopes.Cursor, sortColumn, sortOrder string) *gorm.DB {
	artifactTable := utils.GetTableName(query, &schema.Artifact{})

	// Handle NULL values in cursor
	if cursor.Value == "" {
		// Items without the sort value will be sorted to the bottom, just use ID-based pagination.
		return query.Where(fmt.Sprintf("sort_value.%s IS NULL AND %s.id > ?", sortColumn, artifactTable), cursor.ID)
	}

	cmp := "<"
	if sortOrder == "ASC" {
		cmp = ">"
	}

	// Note that we sort ID ASCENDING as a tie-breaker, so ">" is correct below.
	return query.Where(fmt.Sprintf("(sort_value.%s %s ? OR (sort_value.%s = ? AND %s.id > ?) OR sort_value.%s IS NULL)", sortColumn, cmp, sortColumn, artifactTable, sortColumn),
		cursor.Value, cursor.Value, cursor.ID)
}

// CountByParentIDs returns artifact counts grouped by category for each parent model.
// The outer map key is the parent model ID (context_id from Attribution).
// The inner map key is the artifact category: "model-artifact" for CatalogModelArtifacts,
// or the metricsType value ("performance-metrics", "accuracy-metrics", etc.) for CatalogMetricsArtifacts.
// Categories with zero count are omitted from the inner map.
// Parents with no artifacts have no entry in the outer map.
func (r *CatalogArtifactRepositoryImpl) CountByParentIDs(parentIDs []int32) (map[int32]map[string]int32, error) {
	result := make(map[int32]map[string]int32, len(parentIDs))
	if len(parentIDs) == 0 {
		return result, nil
	}

	modelArtifactTypeID := r.nameToID[CatalogModelArtifactTypeName]
	metricsArtifactTypeID := r.nameToID[CatalogMetricsArtifactTypeName]

	artifactTable := utils.GetTableName(r.db, &schema.Artifact{})
	attributionTable := utils.GetTableName(r.db, &schema.Attribution{})
	propertyTable := utils.GetTableName(r.db, &schema.ArtifactProperty{})

	selectClause := fmt.Sprintf(
		`%s.context_id,
		CASE
			WHEN %s.type_id = %d THEN 'model-artifact'
			WHEN %s.type_id = %d THEN COALESCE(%s.string_value, 'unknown')
		END AS category,
		COUNT(*) AS count`,
		attributionTable,
		artifactTable, modelArtifactTypeID,
		artifactTable, metricsArtifactTypeID,
		propertyTable,
	)

	type countRow struct {
		ContextID int32  `gorm:"column:context_id"`
		Category  string `gorm:"column:category"`
		Count     int32  `gorm:"column:count"`
	}

	var rows []countRow
	err := r.db.Table(artifactTable).
		Select(selectClause).
		Joins(fmt.Sprintf("INNER JOIN %s ON %s.artifact_id = %s.id", attributionTable, attributionTable, artifactTable)).
		Joins(fmt.Sprintf("LEFT JOIN %s ON %s.artifact_id = %s.id AND %s.name = 'metricsType' AND %s.is_custom_property = false", propertyTable, propertyTable, artifactTable, propertyTable, propertyTable)).
		Where(fmt.Sprintf("%s.context_id IN ? AND %s.type_id IN ?", attributionTable, artifactTable), parentIDs, []int32{modelArtifactTypeID, metricsArtifactTypeID}).
		Group(fmt.Sprintf("%s.context_id, category", attributionTable)).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error counting catalog artifacts by parents: %w", err)
	}

	for _, row := range rows {
		if result[row.ContextID] == nil {
			result[row.ContextID] = make(map[string]int32)
		}
		result[row.ContextID][row.Category] = row.Count
	}

	return result, nil
}

func (r *CatalogArtifactRepositoryImpl) DeleteByParentID(artifactTypeName string, parentResourceID int32) error {
	typeID, ok := r.nameToID[artifactTypeName]
	if !ok {
		return fmt.Errorf("unknown artifact type name: %s", artifactTypeName)
	}

	return r.db.Exec(`DELETE FROM "Artifact" WHERE id IN (SELECT artifact_id from "Attribution" INNER JOIN "Artifact" artifact ON artifact.id=artifact_id where context_id=? and type_id=?)`, parentResourceID, typeID).Error
}
