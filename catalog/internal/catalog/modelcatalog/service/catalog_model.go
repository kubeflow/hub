package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	catpagination "github.com/kubeflow/hub/catalog/internal/db/pagination"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	dbfilter "github.com/kubeflow/hub/internal/platform/db/filter"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrCatalogModelNotFound = errors.New("catalog model by id not found")

type CatalogModelRepositoryImpl struct {
	*service.GenericRepository[models.CatalogModel, schema.Context, schema.ContextProperty, *models.CatalogModelListOptions]
}

func NewCatalogModelRepository(db *gorm.DB, typeID int32) models.CatalogModelRepository {
	r := &CatalogModelRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[models.CatalogModel, schema.Context, schema.ContextProperty, *models.CatalogModelListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapCatalogModelToContext,
		SchemaToEntity:          mapDataLayerToCatalogModel,
		EntityToProperties:      mapCatalogModelToContextProperties,
		NotFoundError:           ErrCatalogModelNotFound,
		EntityName:              "catalog model",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applyCatalogModelListFilters,
		CreatePaginationToken:   r.createPaginationToken,
		ApplyCustomOrdering:     r.applyCustomOrdering,
		IsNewEntity:             func(entity models.CatalogModel) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.CatalogModel) bool { return entity.GetCustomProperties() != nil },
		EntityMappingFuncs:      NewCatalogModelEntityMappings(),
		PreserveHistoricalTimes: true, // Catalog preserves timestamps from YAML source data
	})

	return r
}

func (r *CatalogModelRepositoryImpl) Save(model models.CatalogModel) (models.CatalogModel, error) {
	config := r.GetConfig()
	if model.GetTypeID() == nil {
		if config.TypeID > 0 {
			model.SetTypeID(config.TypeID)
		}
	}

	attr := model.GetAttributes()
	if model.GetID() == nil && attr != nil && attr.Name != nil {
		lookupName := *attr.Name
		// When source_id is present, use namespaced name (sourceId:modelName) for uniqueness and storage.
		// Only prefix if not already in sourceId:modelName form (avoid double-prefix; allow display names containing ":").
		if sourceID := getSourceIDFromProperties(model); sourceID != "" {
			prefix := sourceID + ":"
			if !strings.HasPrefix(lookupName, prefix) {
				namespacedName := prefix + lookupName
				attr.Name = &namespacedName
				lookupName = namespacedName
			}
		}
		existing, err := r.lookupModelByName(lookupName)
		if err != nil {
			if !errors.Is(err, ErrCatalogModelNotFound) {
				return nil, fmt.Errorf("error finding existing model named %s: %w", lookupName, err)
			}
		} else {
			model.SetID(existing.ID)
		}
	}

	return r.GenericRepository.Save(model, nil)
}

// getSourceIDFromProperties returns the source_id property value if present.
func getSourceIDFromProperties(model models.CatalogModel) string {
	if model.GetProperties() == nil {
		return ""
	}
	for _, p := range *model.GetProperties() {
		if p.Name == "source_id" && p.StringValue != nil {
			return *p.StringValue
		}
	}
	return ""
}

// getSourceIDFromContextProperties returns the source_id property value from DB context properties.
func getSourceIDFromContextProperties(propertiesCtx []schema.ContextProperty) string {
	for _, p := range propertiesCtx {
		if p.Name == "source_id" && p.StringValue != nil {
			return *p.StringValue
		}
	}
	return ""
}

// escapeLike escapes SQL LIKE metacharacters (%, _, \) in s so it can be used safely as a literal in a LIKE pattern.
func escapeLike(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch c {
		case '\\', '%', '_':
			b.WriteRune('\\')
			b.WriteRune(c)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// ApplyStandardPagination overrides the base implementation to use catalog-specific allowed columns
func (r *CatalogModelRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.CatalogModelListOptions, entities any) *gorm.DB {
	pageSize := listOptions.GetPageSize()
	orderBy := listOptions.GetOrderBy()
	sortOrder := listOptions.GetSortOrder()
	nextPageToken := listOptions.GetNextPageToken()

	pagination := &dbmodels.Pagination{
		PageSize:      &pageSize,
		OrderBy:       &orderBy,
		SortOrder:     &sortOrder,
		NextPageToken: &nextPageToken,
	}

	// Use catalog-specific allowed columns (includes NAME)
	return query.Scopes(scopes.PaginateWithOptions(entities, pagination, r.GetConfig().DB, "Context", catpagination.CatalogOrderByColumns))
}

func (r *CatalogModelRepositoryImpl) List(listOptions models.CatalogModelListOptions) (*dbmodels.ListWrapper[models.CatalogModel], error) {
	return r.GenericRepository.List(&listOptions)
}

func (r *CatalogModelRepositoryImpl) GetByName(name string) (models.CatalogModel, error) {
	var zeroEntity models.CatalogModel
	entity, err := r.lookupModelByName(name)
	if err != nil {
		return zeroEntity, err
	}

	config := r.GetConfig()

	// Query properties
	var properties []schema.ContextProperty
	if err := config.DB.Where(config.PropertyFieldName+" = ?", entity.ID).Find(&properties).Error; err != nil {
		// Sanitize database errors to avoid exposing internal details to users
		err = dbutil.SanitizeDatabaseError(err)
		return zeroEntity, fmt.Errorf("error getting properties by %s id: %w", config.EntityName, err)
	}

	// Map to domain model
	return config.SchemaToEntity(*entity, properties), nil
}

func (r *CatalogModelRepositoryImpl) lookupModelByName(name string) (*schema.Context, error) {
	var entity schema.Context

	config := r.GetConfig()

	if err := config.DB.Where("name = ? AND type_id = ?", name, config.TypeID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", config.NotFoundError, err)
		}
		// Sanitize database errors to avoid exposing internal details to users
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting %s by name: %w", config.EntityName, err)
	}

	return &entity, nil
}

func (r *CatalogModelRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()

	return config.DB.Transaction(func(tx *gorm.DB) error {
		// Delete artifacts linked to models in this source that aren't linked
		// with an Event.
		deleteArtifactsQuery :=
			`DELETE FROM "Artifact" WHERE id IN (
			SELECT "Artifact".id
			FROM "Context"
			INNER JOIN "ContextProperty" ON "Context".id="ContextProperty".context_id
				AND "ContextProperty".name='source_id' AND
				"ContextProperty".string_value=?
			INNER JOIN "Attribution" ON "Context".id="Attribution".context_id
			INNER JOIN "Artifact" ON "Attribution".artifact_id="Artifact".id
			WHERE "Context".type_id=? AND NOT EXISTS (
				SELECT 1 FROM "Event" WHERE "Event".artifact_id="Artifact".id
			)
		)`
		err := tx.Exec(deleteArtifactsQuery, sourceID, config.TypeID).Error
		if err != nil {
			return fmt.Errorf("unable to delete artifacts from source %q: %w", sourceID, err)
		}

		// Delete all Context records where there's a ContextProperty with name='source_id' and string_value=sourceID
		query := `DELETE FROM "Context" WHERE id IN (
			SELECT "Context".id
			FROM "Context"
			INNER JOIN "ContextProperty" ON "Context".id="ContextProperty".context_id
			AND "ContextProperty".name='source_id'
			WHERE "ContextProperty".string_value=?
			AND "Context".type_id=?
		)`

		err = tx.Exec(query, sourceID, config.TypeID).Error
		if err != nil {
			return fmt.Errorf("unable to delete contexts from source %q: %w", sourceID, err)
		}

		return nil
	})
}

func (r *CatalogModelRepositoryImpl) DeleteByID(id int32) error {
	config := r.GetConfig()

	var rowsAffected int64

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Delete artifacts linked to this model that aren't linked with an Event.
		deleteArtifactsQuery :=
			`DELETE FROM "Artifact" WHERE id IN (
			SELECT "Artifact".id FROM "Context"
				INNER JOIN "Attribution" ON "Context".id="Attribution".context_id
				INNER JOIN "Artifact" ON "Attribution".artifact_id="Artifact".id
			WHERE "Context".id=? AND "Context".type_id=?
				AND NOT EXISTS (
					SELECT 1 FROM "Event" WHERE "Event".artifact_id="Artifact".id
				)
		)`

		err := tx.Exec(deleteArtifactsQuery, id, config.TypeID).Error
		if err != nil {
			return fmt.Errorf("unable to delete linked artifacts for model %d: %w", id, err)
		}

		// Delete the Context record by ID and type_id
		// ContextProperty records will be deleted via foreign key cascade
		result := tx.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&schema.Context{})

		if result.Error != nil {
			return result.Error
		}

		rowsAffected = result.RowsAffected
		return nil
	})

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: id %d", config.NotFoundError, id)
	}

	return nil
}

// GetDistinctSourceIDs retrieves all unique source_id values from catalog models.
// This method queries the ContextProperty table to find distinct string_value entries
// where the property name is 'source_id'.
func (r *CatalogModelRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	config := r.GetConfig()

	var sourceIDs []string

	// Execute the SQL query to get distinct source_id values
	query := `SELECT DISTINCT string_value FROM "ContextProperty" WHERE name='source_id'`

	rows, err := config.DB.Raw(query).Rows()
	if err != nil {
		// Sanitize database errors to avoid exposing internal details to users
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sourceID string
		if err := rows.Scan(&sourceID); err != nil {
			err = dbutil.SanitizeDatabaseError(err)
			return nil, fmt.Errorf("error scanning source ID: %w", err)
		}
		sourceIDs = append(sourceIDs, sourceID)
	}

	if err := rows.Err(); err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error iterating source ID rows: %w", err)
	}

	return sourceIDs, nil
}

func (r *CatalogModelRepositoryImpl) GetTypeID() int32 {
	return r.GetConfig().TypeID
}

func applyCatalogModelListFilters(query *gorm.DB, listOptions *models.CatalogModelListOptions) *gorm.DB {
	contextTable := utils.GetTableName(query.Statement.DB, &schema.Context{})

	if listOptions.Name != nil {
		query = query.Where(fmt.Sprintf("%s.name LIKE ?", contextTable), listOptions.Name)
	} else if listOptions.ExternalID != nil {
		extID := *listOptions.ExternalID
		// Match both exact and namespaced (sourceId:externalId) so existing integrations
		// that filter by external_id without source prefix still return all matching models.
		namespacedPattern := "%:" + escapeLike(extID)
		query = query.Where(fmt.Sprintf("(%s.external_id = ? OR %s.external_id LIKE ?)", contextTable, contextTable), extID, namespacedPattern)
	}

	if listOptions.Query != nil && *listOptions.Query != "" {
		queryPattern := fmt.Sprintf("%%%s%%", strings.ToLower(*listOptions.Query))
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})

		// Search in name (context table)
		nameCondition := fmt.Sprintf("LOWER(%s.name) LIKE ?", contextTable)

		// Search in description, provider, libraryName properties
		propertyCondition := fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name IN (?, ?, ?) AND LOWER(cp.string_value) LIKE ?)",
			propertyTable, contextTable)

		// Search in tasks and validated_tasks (stored as JSON arrays)
		tasksCondition := fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name IN (?, ?) AND LOWER(cp.string_value) LIKE ?)",
			propertyTable, contextTable)

		query = query.Where(fmt.Sprintf("(%s OR %s OR %s)", nameCondition, propertyCondition, tasksCondition),
			queryPattern,                                           // for name
			"description", "provider", "libraryName", queryPattern, // for properties
			"tasks", "validated_tasks", queryPattern, // for tasks and validated_tasks
		)
	}

	// Filter out empty strings from SourceIDs, for some reason it's passed if no sources are specified
	var nonEmptySourceIDs []string
	if listOptions.SourceIDs != nil {
		for _, sourceID := range *listOptions.SourceIDs {
			if sourceID != "" {
				nonEmptySourceIDs = append(nonEmptySourceIDs, sourceID)
			}
		}
	}

	if len(nonEmptySourceIDs) > 0 {
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})

		joinClause := fmt.Sprintf("JOIN %s cp ON cp.context_id = %s.id", propertyTable, contextTable)
		query = query.Joins(joinClause).
			Where("cp.name = ? AND cp.string_value IN ?", "source_id", nonEmptySourceIDs)
	}

	return query
}

func mapCatalogModelToContext(model models.CatalogModel) schema.Context {
	attrs := model.GetAttributes()
	context := schema.Context{}

	if typeID := model.GetTypeID(); typeID != nil {
		context.TypeID = *typeID
	}

	if model.GetID() != nil {
		context.ID = *model.GetID()
	}

	if attrs != nil {
		if attrs.Name != nil {
			context.Name = *attrs.Name
		}
		context.ExternalID = attrs.ExternalID
		// When source_id is present, namespace external_id (sourceId:externalId) so the same
		// model in multiple sources does not violate UNIQUE(external_id) on Context.
		if context.ExternalID != nil && *context.ExternalID != "" {
			if sourceID := getSourceIDFromProperties(model); sourceID != "" {
				prefix := sourceID + ":"
				if !strings.HasPrefix(*context.ExternalID, prefix) {
					namespacedExtID := prefix + *context.ExternalID
					context.ExternalID = &namespacedExtID
				}
			}
		}
		if attrs.CreateTimeSinceEpoch != nil {
			context.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			context.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return context
}

func mapCatalogModelToContextProperties(model models.CatalogModel, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if model.GetProperties() != nil {
		for _, prop := range *model.GetProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if model.GetCustomProperties() != nil {
		for _, prop := range *model.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

func mapDataLayerToCatalogModel(modelCtx schema.Context, propertiesCtx []schema.ContextProperty) models.CatalogModel {
	externalID := modelCtx.ExternalID
	// When stored external_id is namespaced (sourceId:externalId), strip prefix for display/API.
	if externalID != nil && *externalID != "" {
		if sourceID := getSourceIDFromContextProperties(propertiesCtx); sourceID != "" {
			prefix := sourceID + ":"
			if strings.HasPrefix(*externalID, prefix) {
				displayExtID := (*externalID)[len(prefix):]
				externalID = &displayExtID
			}
		}
	}
	catalogModel := &models.CatalogModelImpl{
		ID:     &modelCtx.ID,
		TypeID: &modelCtx.TypeID,
		Attributes: &models.CatalogModelAttributes{
			Name:                     &modelCtx.Name,
			ExternalID:               externalID,
			CreateTimeSinceEpoch:     &modelCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &modelCtx.LastUpdateTimeSinceEpoch,
		},
	}

	properties := []dbmodels.Properties{}
	customProperties := []dbmodels.Properties{}

	for _, prop := range propertiesCtx {
		mappedProperty := service.MapContextPropertyToProperties(prop)

		if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			properties = append(properties, mappedProperty)
		}
	}

	catalogModel.Properties = &properties
	catalogModel.CustomProperties = &customProperties

	return catalogModel
}

// applyCustomOrdering applies custom ordering logic for non-standard orderBy field
func (r *CatalogModelRepositoryImpl) applyCustomOrdering(query *gorm.DB, listOptions *models.CatalogModelListOptions) *gorm.DB {

	db := r.GetConfig().DB
	contextTable := utils.GetTableName(db, &schema.Context{})
	orderBy := listOptions.GetOrderBy()

	// Handle NAME ordering specially (catalog-specific)
	if orderBy == "NAME" {
		return catpagination.ApplyNameOrdering(query, contextTable, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), true)
	}

	subquery, sortColumn := r.sortValueQuery(listOptions, contextTable+".id")
	if subquery == nil {
		// Fall back to standard pagination with catalog-specific allowed columns
		return r.ApplyStandardPagination(query, listOptions, []models.CatalogModel{})
	}
	subquery = subquery.Group(contextTable + ".id")

	// Join the main query with the subquery
	query = query.
		Joins(fmt.Sprintf("LEFT JOIN (?) sort_value ON %s.id=sort_value.id", contextTable), subquery)

	// Apply sorting order
	sortOrder := listOptions.GetSortOrder()
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}
	query = query.Order(fmt.Sprintf("sort_value.%s %s NULLS LAST, %s.id", sortColumn, sortOrder, contextTable))

	// Handle cursor-based pagination with nextPageToken
	nextPageToken := listOptions.GetNextPageToken()
	if nextPageToken != "" {
		// Parse the cursor from the token
		cursor, err := scopes.DecodeCursor(nextPageToken)
		if err != nil {
			_ = query.AddError(fmt.Errorf("invalid nextPageToken: %w", err))
		} else {
			// Apply WHERE clause for cursor-based pagination with ACCURACY
			query = r.applyCursorPagination(query, cursor, sortColumn, sortOrder)
		}
	}

	// Apply pagination limit
	pageSize := listOptions.GetPageSize()
	if pageSize > 0 {
		query = query.Limit(int(pageSize) + 1) // +1 to detect if there are more pages
	}

	return query
}

// applyCursorPagination applies WHERE clause for cursor-based pagination with ACCURACY sorting
func (r *CatalogModelRepositoryImpl) applyCursorPagination(query *gorm.DB, cursor *scopes.Cursor, sortColumn, sortOrder string) *gorm.DB {
	contextTable := utils.GetTableName(query, &schema.Context{})

	// Handle NULL values in cursor
	if cursor.Value == "" {
		// Items without the sort value will be sorted to the bottom, just use ID-based pagination.
		return query.Where(fmt.Sprintf("sort_value.%s IS NULL AND %s.id > ?", sortColumn, contextTable), cursor.ID)
	}

	cmp := "<"
	if sortOrder == "ASC" {
		cmp = ">"
	}

	// Note that we sort ID ASCENDING as a tie-breaker, so ">" is correct below.
	return query.Where(fmt.Sprintf("(sort_value.%s %s ? OR (sort_value.%s = ? AND %s.id > ?) OR sort_value.%s IS NULL)", sortColumn, cmp, sortColumn, contextTable, sortColumn),
		cursor.Value, cursor.Value, cursor.ID)
}

func (r *CatalogModelRepositoryImpl) createPaginationToken(lastItem schema.Context, listOptions *models.CatalogModelListOptions) string {
	// Handle NAME ordering (catalog-specific)
	if listOptions.GetOrderBy() == "NAME" {
		displayName := lastItem.Name
		if _, after, ok := strings.Cut(lastItem.Name, ":"); ok {
			displayName = after
		}
		return catpagination.CreateNamePaginationToken(lastItem.ID, &displayName)
	}

	sortValueQuery, column := r.sortValueQuery(listOptions)
	if sortValueQuery != nil {
		contextTable := utils.GetTableName(r.GetConfig().DB, &schema.Context{})
		sortValueQuery = sortValueQuery.Where(contextTable+".id=?", lastItem.ID)

		var result struct {
			IntValue    *int64   `gorm:"int_value"`
			DoubleValue *float64 `gorm:"double_value"`
			StringValue *string  `gorm:"string_value"`
		}
		err := sortValueQuery.Scan(&result).Error
		if err != nil {
			glog.Warningf("Failed to get sort value: %v", err)
		} else {
			switch column {
			case "int_value":
				return scopes.CreateNextPageToken(lastItem.ID, result.IntValue)
			case "double_value":
				return scopes.CreateNextPageToken(lastItem.ID, result.DoubleValue)
			case "string_value":
				fallthrough
			default:
				return scopes.CreateNextPageToken(lastItem.ID, result.StringValue)
			}
		}
	}

	return r.CreateDefaultPaginationToken(lastItem, listOptions)
}

// sortValueQuery returns a query that will produce the value to sort on for
// the List response. The returned string is the column name.
//
// If the sort does not require a subquery, sortValueQuery returns nil.
func (r *CatalogModelRepositoryImpl) sortValueQuery(listOptions *models.CatalogModelListOptions, extraColumns ...any) (*gorm.DB, string) {
	db := r.GetConfig().DB
	contextTable := utils.GetTableName(db, &schema.Context{})

	query := db.Table(contextTable).
		Where(contextTable+".type_id=?", r.GetConfig().TypeID)

	orderBy := strings.Split(listOptions.GetOrderBy(), ".")

	var valueColumn string

	switch {
	case len(orderBy) == 3 && orderBy[0] == "artifacts":
		// artifacts.<property>.<value_column> e.g. artifacts.ttft_p90.double_value

		attributionTable := utils.GetTableName(db, &schema.Attribution{})
		propertyTable := utils.GetTableName(db, &schema.ArtifactProperty{})

		aggFn := "max"
		if listOptions.GetSortOrder() == "ASC" {
			aggFn = "min"
		}
		valueColumn = orderBy[2]

		query = query.
			Select(fmt.Sprintf("%s(%s.%s) AS %s", aggFn, propertyTable, valueColumn, valueColumn), extraColumns...).
			Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id=%s.context_id", attributionTable, contextTable, attributionTable)).
			Joins(fmt.Sprintf("LEFT JOIN %s ON %s.artifact_id=%s.artifact_id AND %s.name=?", propertyTable, attributionTable, propertyTable, propertyTable), orderBy[1])

		// Apply artifact filter conditions to sorting subquery, so
		// that sorting only considers artifacts that pass the filter
		// criteria
		if filterQuery := listOptions.GetFilterQuery(); filterQuery != "" {
			artifactConditions := r.extractArtifactConditions(filterQuery)
			if len(artifactConditions) > 0 {
				// Apply each artifact filter condition to the sorting subquery
				for _, condition := range artifactConditions {
					// Create a separate ArtifactProperty alias for this filter condition
					// to avoid conflicts with the sorting property JOIN
					filterPropAlias := fmt.Sprintf("ArtifactProperty_filter_%s", condition.property)
					valueColumn := r.determineValueColumn(condition)

					// Add JOINs for the filter condition
					query = query.
						Joins(fmt.Sprintf("LEFT JOIN %s %s ON %s.artifact_id=%s.artifact_id AND %s.name=?",
							propertyTable, filterPropAlias, attributionTable, filterPropAlias, filterPropAlias), condition.property).
						Where(fmt.Sprintf("%s.%s %s ?", filterPropAlias, valueColumn, condition.operator), condition.value)
				}
			}
		}
	case len(orderBy) == 2:
		// <property>.<value_column> e.g. provider.string_value
		propertyTable := utils.GetTableName(db, &schema.ContextProperty{})
		valueColumn = orderBy[1]
		query = query.
			Select(fmt.Sprintf("max(%s.%s) AS %s", propertyTable, valueColumn, valueColumn), extraColumns...).
			Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id=%s.context_id AND %s.name=?", propertyTable, contextTable, propertyTable, propertyTable), orderBy[0])
	default:
		// Standard sort will work
		return nil, ""
	}

	// The query is built, but verify that the value column is valid before
	// returning it.
	switch valueColumn {
	case "int_value", "double_value", "string_value":
		// OK
	default:
		return nil, ""
	}

	return query, valueColumn
}

// artifactCondition represents a single artifact property filter condition
type artifactCondition struct {
	property  string // artifact property name (e.g., "status", "accuracy")
	operator  string // filter operator (e.g., "=", ">", "LIKE")
	value     any    // filter value
	valueType string // explicit value type if specified (e.g., "string_value", "double_value")
}

// extractArtifactConditions extracts all artifact property filter conditions from a filter query
// Returns slice of conditions that should be applied to artifact filtering
func (r *CatalogModelRepositoryImpl) extractArtifactConditions(filterQuery string) []artifactCondition {
	if filterQuery == "" {
		return nil
	}

	// Parse the filter query using the existing filter parser
	filterExpr, err := dbfilter.Parse(filterQuery)
	if err != nil {
		// If parsing fails, return no conditions (fail safe)
		return nil
	}

	if filterExpr == nil {
		return nil
	}

	return r.collectArtifactConditionsFromExpr(filterExpr)
}

// collectArtifactConditionsFromExpr recursively extracts artifact conditions from a filter expression
func (r *CatalogModelRepositoryImpl) collectArtifactConditionsFromExpr(expr *dbfilter.FilterExpression) []artifactCondition {
	if expr.IsLeaf {
		// Check if this is an artifact property condition
		if r.isArtifactProperty(expr.Property) {
			condition := r.parseArtifactCondition(expr)
			if condition != nil {
				return []artifactCondition{*condition}
			}
		}
		return nil
	}

	// For non-leaf nodes, collect conditions from left and right
	var conditions []artifactCondition
	if expr.Left != nil {
		conditions = append(conditions, r.collectArtifactConditionsFromExpr(expr.Left)...)
	}
	if expr.Right != nil {
		conditions = append(conditions, r.collectArtifactConditionsFromExpr(expr.Right)...)
	}

	return conditions
}

// isArtifactProperty checks if a property name refers to an artifact property
// Artifact properties have the format "artifacts.property_name[.value_type]"
func (r *CatalogModelRepositoryImpl) isArtifactProperty(propertyName string) bool {
	return strings.HasPrefix(propertyName, "artifacts.")
}

// parseArtifactCondition converts a filter expression to an artifact condition
func (r *CatalogModelRepositoryImpl) parseArtifactCondition(expr *dbfilter.FilterExpression) *artifactCondition {
	if !r.isArtifactProperty(expr.Property) {
		return nil
	}

	// Extract artifact property name and optional value type
	// Format: artifacts.property_name[.value_type]
	parts := strings.SplitN(expr.Property, ".", 3)
	if len(parts) < 2 {
		return nil // Invalid format
	}

	propertyName := parts[1] // "property_name"
	var valueType string

	// Check if there's an explicit value type (e.g., artifacts.accuracy.double_value)
	if len(parts) == 3 {
		valueType = parts[2]
	}

	return &artifactCondition{
		property:  propertyName,
		operator:  expr.Operator,
		value:     expr.Value,
		valueType: valueType,
	}
}

// determineValueColumn determines which database column to use for an artifact condition
func (r *CatalogModelRepositoryImpl) determineValueColumn(condition artifactCondition) string {
	// If explicit value type is specified, use it
	if condition.valueType != "" {
		switch condition.valueType {
		case "string_value", "double_value", "int_value", "bool_value":
			return condition.valueType
		}
	}

	// Infer value type from the value
	switch condition.value.(type) {
	case string:
		return "string_value"
	case int, int32, int64:
		return "int_value"
	case float32, float64:
		return "double_value"
	case bool:
		return "bool_value"
	default:
		return "string_value" // fallback
	}
}
