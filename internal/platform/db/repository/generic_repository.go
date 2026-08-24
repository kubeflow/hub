package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	"github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/filter"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	platformerrors "github.com/kubeflow/hub/internal/platform/errors"
	"gorm.io/gorm"
)

// Generic constraints for different entity types
type SchemaEntity interface {
	schema.Artifact | schema.Context | schema.Execution
}

type PropertyEntity interface {
	schema.ArtifactProperty | schema.ContextProperty | schema.ExecutionProperty
}

// Mapper function types
type EntityToSchemaMapper[TEntity any, TSchema SchemaEntity] func(TEntity) TSchema
type SchemaToEntityMapper[TSchema SchemaEntity, TProp PropertyEntity, TEntity any] func(TSchema, []TProp) TEntity
type EntityToPropertiesMapper[TEntity any, TProp PropertyEntity] func(TEntity, int32) []TProp

// List options interface
type BaseListOptions interface {
	GetPageSize() int32
	GetNextPageToken() string
	SetNextPageToken(*string)
	GetOrderBy() string
	GetSortOrder() string
	GetFilterQuery() string
}

// Filter applier interface for entities that support advanced filtering
type FilterApplier interface {
	GetRestEntityType() filter.RestEntityType
}

// ApplyFilterQuery applies advanced filter query processing to a GORM query
func ApplyFilterQuery(query *gorm.DB, listOptions any, mappingFuncs filter.EntityMappingFunctions) (*gorm.DB, error) {
	if filterQueryGetter, ok := listOptions.(interface{ GetFilterQuery() string }); ok {
		if filterQuery := filterQueryGetter.GetFilterQuery(); filterQuery != "" {
			if filterApplier, ok := listOptions.(FilterApplier); ok {
				filterExpr, err := filter.Parse(filterQuery)
				if err != nil {
					enhancedErr := dbutil.EnhanceFilterQueryError(err, filterQuery)
					return nil, fmt.Errorf("%v: %w", enhancedErr, platformerrors.ErrBadRequest)
				}

				if filterExpr != nil {
					queryBuilder := filter.NewQueryBuilderForRestEntity(filterApplier.GetRestEntityType(), mappingFuncs)
					query = queryBuilder.BuildQuery(query, filterExpr)
				}
			}
		}
	}
	return query, nil
}

// Generic repository configuration
type GenericRepositoryConfig[TEntity any, TSchema SchemaEntity, TProp PropertyEntity, TListOpts BaseListOptions] struct {
	DB                      *gorm.DB
	TypeID                  int32
	EntityToSchema          EntityToSchemaMapper[TEntity, TSchema]
	SchemaToEntity          SchemaToEntityMapper[TSchema, TProp, TEntity]
	EntityToProperties      EntityToPropertiesMapper[TEntity, TProp]
	NotFoundError           error
	EntityName              string
	PropertyFieldName       string
	ApplyListFilters        func(*gorm.DB, TListOpts) *gorm.DB
	CreatePaginationToken   func(TSchema, TListOpts) string
	ApplyCustomOrdering     func(*gorm.DB, TListOpts) *gorm.DB
	IsNewEntity             func(TEntity) bool
	HasCustomProperties     func(TEntity) bool
	EntityMappingFuncs      filter.EntityMappingFunctions
	PreserveHistoricalTimes bool
}

// Generic repository implementation
type GenericRepository[TEntity any, TSchema SchemaEntity, TProp PropertyEntity, TListOpts BaseListOptions] struct {
	config GenericRepositoryConfig[TEntity, TSchema, TProp, TListOpts]
}

func NewGenericRepository[TEntity any, TSchema SchemaEntity, TProp PropertyEntity, TListOpts BaseListOptions](
	config GenericRepositoryConfig[TEntity, TSchema, TProp, TListOpts],
) *GenericRepository[TEntity, TSchema, TProp, TListOpts] {
	return &GenericRepository[TEntity, TSchema, TProp, TListOpts]{
		config: config,
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) GetByID(id int32) (TEntity, error) {
	var schemaEntity TSchema
	var properties []TProp
	var zeroEntity TEntity

	if err := r.config.DB.Where("id = ? AND type_id = ?", id, r.config.TypeID).First(&schemaEntity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zeroEntity, fmt.Errorf("%w: %v", r.config.NotFoundError, err)
		}
		return zeroEntity, fmt.Errorf("error getting %s by id: %w", r.config.EntityName, err)
	}

	entityID := r.getEntityID(schemaEntity)
	if err := r.config.DB.Where(r.config.PropertyFieldName+" = ?", entityID).Find(&properties).Error; err != nil {
		return zeroEntity, fmt.Errorf("error getting properties by %s id: %w", r.config.EntityName, err)
	}

	return r.config.SchemaToEntity(schemaEntity, properties), nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) GetByName(name string) (TEntity, error) {
	var schemaEntity TSchema
	var properties []TProp
	var zeroEntity TEntity

	if err := r.config.DB.Where("name = ? AND type_id = ?", name, r.config.TypeID).First(&schemaEntity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zeroEntity, fmt.Errorf("%w: %v", r.config.NotFoundError, err)
		}
		return zeroEntity, fmt.Errorf("error getting %s by name: %w", r.config.EntityName, err)
	}

	entityID := r.getEntityID(schemaEntity)
	if err := r.config.DB.Where(r.config.PropertyFieldName+" = ?", entityID).Find(&properties).Error; err != nil {
		return zeroEntity, fmt.Errorf("error getting properties by %s id: %w", r.config.EntityName, err)
	}

	return r.config.SchemaToEntity(schemaEntity, properties), nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) List(listOptions TListOpts) (*entity.ListWrapper[TEntity], error) {
	pageSize := listOptions.GetPageSize()

	list := entity.ListWrapper[TEntity]{
		PageSize: pageSize,
	}

	var entities []TEntity
	var schemaEntities []TSchema

	query := r.buildBaseQuery()

	if r.config.ApplyListFilters != nil {
		query = r.config.ApplyListFilters(query, listOptions)
	}

	query, err := ApplyFilterQuery(query, listOptions, r.config.EntityMappingFuncs)
	if err != nil {
		return nil, err
	}

	if r.config.ApplyCustomOrdering != nil {
		query = r.config.ApplyCustomOrdering(query, listOptions)
	} else {
		query = r.ApplyStandardPagination(query, listOptions, entities)
	}

	if err := query.Find(&schemaEntities).Error; err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error listing %ss: %w", r.config.EntityName, err)
	}

	hasMore := false
	if pageSize > 0 {
		hasMore = len(schemaEntities) > int(pageSize)
		if hasMore {
			schemaEntities = schemaEntities[:len(schemaEntities)-1]
		}
	}

	for _, se := range schemaEntities {
		var properties []TProp
		entityID := r.getEntityID(se)
		if err := r.config.DB.Where(r.config.PropertyFieldName+" = ?", entityID).Find(&properties).Error; err != nil {
			return nil, fmt.Errorf("error getting properties by %s id: %w", r.config.EntityName, err)
		}

		e := r.config.SchemaToEntity(se, properties)
		entities = append(entities, e)
	}

	if hasMore && len(schemaEntities) > 0 {
		lastEntity := schemaEntities[len(schemaEntities)-1]
		var nextToken string
		if r.config.CreatePaginationToken != nil {
			nextToken = r.config.CreatePaginationToken(lastEntity, listOptions)
		} else {
			nextToken = r.CreateDefaultPaginationToken(lastEntity, listOptions)
		}
		listOptions.SetNextPageToken(&nextToken)
	} else {
		listOptions.SetNextPageToken(nil)
	}

	list.Items = entities
	nextPageToken := listOptions.GetNextPageToken()
	list.NextPageToken = nextPageToken
	list.Size = int32(len(entities))

	return &list, nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) Save(e TEntity, parentResourceID *int32) (TEntity, error) {
	now := time.Now().UnixMilli()
	var zeroEntity TEntity

	schemaEntity := r.config.EntityToSchema(e)
	var finalProperties []TProp

	isNewEntity := r.config.IsNewEntity != nil && r.config.IsNewEntity(e)

	existingCreateTime := r.getCreateTime(schemaEntity)
	existingUpdateTime := r.getLastUpdateTime(schemaEntity)

	if r.config.PreserveHistoricalTimes {
		if existingUpdateTime == 0 {
			r.setLastUpdateTime(&schemaEntity, now)
		}
		if existingCreateTime == 0 {
			r.setCreateTime(&schemaEntity, now)
		}
	} else {
		if isNewEntity {
			r.setLastUpdateTime(&schemaEntity, now)
			r.setCreateTime(&schemaEntity, now)
		} else {
			r.setLastUpdateTime(&schemaEntity, now)
			if existingCreateTime == 0 {
				r.setCreateTime(&schemaEntity, now)
			}
		}
	}

	hasCustomProperties := r.config.HasCustomProperties != nil && r.config.HasCustomProperties(e)

	err := r.config.DB.Transaction(func(tx *gorm.DB) error {
		if isNewEntity {
			if err := tx.Save(&schemaEntity).Error; err != nil {
				return fmt.Errorf("error saving %s: %w", r.config.EntityName, err)
			}
		} else {
			omitFields := r.getNonUpdatableFields(schemaEntity)
			if err := tx.Model(&schemaEntity).Omit(omitFields...).Updates(&schemaEntity).Error; err != nil {
				return fmt.Errorf("error saving %s: %w", r.config.EntityName, err)
			}
		}

		if parentResourceID != nil {
			if err := r.handleParentRelationship(tx, schemaEntity, parentResourceID); err != nil {
				return err
			}
		}

		entityID := r.getEntityID(schemaEntity)
		properties := r.config.EntityToProperties(e, entityID)

		if err := r.handleProperties(tx, entityID, properties, hasCustomProperties); err != nil {
			return err
		}

		if err := tx.Where(r.config.PropertyFieldName+" = ?", entityID).Find(&finalProperties).Error; err != nil {
			return fmt.Errorf("error getting final properties by %s id: %w", r.config.EntityName, err)
		}

		return nil
	})
	if err != nil {
		return zeroEntity, err
	}

	return r.config.SchemaToEntity(schemaEntity, finalProperties), nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) buildBaseQuery() *gorm.DB {
	var schemaEntity TSchema
	var tableName string
	var model any

	switch any(schemaEntity).(type) {
	case schema.Artifact:
		tableName = "Artifact"
		model = &schema.Artifact{}
	case schema.Context:
		tableName = "Context"
		model = &schema.Context{}
	case schema.Execution:
		tableName = "Execution"
		model = &schema.Execution{}
	default:
		panic(fmt.Sprintf("unsupported schema entity type: %T", schemaEntity))
	}

	tableNameQuoted := dbutil.QuoteTableName(r.config.DB, tableName)
	whereClause := fmt.Sprintf("%s.type_id = ?", tableNameQuoted)

	return r.config.DB.Model(model).Where(whereClause, r.config.TypeID)
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getEntityID(e TSchema) int32 {
	switch v := any(e).(type) {
	case schema.Artifact:
		return v.ID
	case schema.Context:
		return v.ID
	case schema.Execution:
		return v.ID
	default:
		panic(fmt.Sprintf("unsupported entity type: %T", e))
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) setLastUpdateTime(e *TSchema, timestamp int64) {
	switch v := any(e).(type) {
	case *schema.Artifact:
		v.LastUpdateTimeSinceEpoch = timestamp
	case *schema.Context:
		v.LastUpdateTimeSinceEpoch = timestamp
	case *schema.Execution:
		v.LastUpdateTimeSinceEpoch = timestamp
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) setCreateTime(e *TSchema, timestamp int64) {
	switch v := any(e).(type) {
	case *schema.Artifact:
		v.CreateTimeSinceEpoch = timestamp
	case *schema.Context:
		v.CreateTimeSinceEpoch = timestamp
	case *schema.Execution:
		v.CreateTimeSinceEpoch = timestamp
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) handleParentRelationship(tx *gorm.DB, e TSchema, parentResourceID *int32) error {
	entityID := r.getEntityID(e)

	switch any(e).(type) {
	case schema.Artifact:
		var existing schema.Attribution
		result := tx.Where("context_id = ? AND artifact_id = ?", *parentResourceID, entityID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			attribution := schema.Attribution{ArtifactID: entityID, ContextID: *parentResourceID}
			if err := tx.Create(&attribution).Error; err != nil {
				return fmt.Errorf("error creating attribution: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("error checking existing attribution: %w", result.Error)
		}

	case schema.Context:
		var existing schema.ParentContext
		result := tx.Where("parent_context_id = ? AND context_id = ?", *parentResourceID, entityID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			parentContext := schema.ParentContext{ContextID: entityID, ParentContextID: *parentResourceID}
			if err := tx.Create(&parentContext).Error; err != nil {
				return fmt.Errorf("error creating parent context: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("error checking existing parent context: %w", result.Error)
		}

	case schema.Execution:
		var existing schema.Association
		result := tx.Where("context_id = ? AND execution_id = ?", *parentResourceID, entityID).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			association := schema.Association{ExecutionID: entityID, ContextID: *parentResourceID}
			if err := tx.Create(&association).Error; err != nil {
				return fmt.Errorf("error creating association: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("error checking existing association: %w", result.Error)
		}
	}

	return nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) handleProperties(tx *gorm.DB, entityID int32, properties []TProp, hasCustomProperties bool) error {
	if hasCustomProperties {
		var existingCustomProperties []TProp
		if err := tx.Where(r.config.PropertyFieldName+" = ? AND is_custom_property = ?", entityID, true).Find(&existingCustomProperties).Error; err != nil {
			return fmt.Errorf("error getting existing custom properties: %w", err)
		}

		for _, existingProp := range existingCustomProperties {
			found := false
			for _, prop := range properties {
				if r.propertiesMatch(prop, existingProp) {
					found = true
					break
				}
			}

			if !found {
				if err := tx.Delete(&existingProp).Error; err != nil {
					return fmt.Errorf("error deleting property: %w", err)
				}
			}
		}
	}

	for _, prop := range properties {
		var existingProp TProp
		result := tx.Where(r.config.PropertyFieldName+" = ? AND name = ? AND is_custom_property = ?",
			entityID, r.getPropertyName(prop), r.getPropertyIsCustom(prop)).First(&existingProp)

		switch result.Error {
		case nil:
			// Use explicit WHERE (all 3 PK columns) and select only value columns to avoid GORM
			// skipping is_custom_property=false in WHERE (zero-value bool).
			if err := tx.Where(r.config.PropertyFieldName+" = ? AND name = ? AND is_custom_property = ?",
				r.getPropertyEntityID(prop), r.getPropertyName(prop), r.getPropertyIsCustom(prop)).
				Select("int_value", "double_value", "string_value", "bool_value", "byte_value", "proto_value").
				Updates(prop).Error; err != nil {
				return fmt.Errorf("error updating property %s: %w", r.getPropertyName(prop), err)
			}
		case gorm.ErrRecordNotFound:
			if err := tx.Create(&prop).Error; err != nil {
				return fmt.Errorf("error creating property %s: %w", r.getPropertyName(prop), err)
			}
		default:
			return fmt.Errorf("error checking existing property %s: %w", r.getPropertyName(prop), result.Error)
		}
	}

	return nil
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) propertiesMatch(prop1, prop2 TProp) bool {
	return r.getPropertyName(prop1) == r.getPropertyName(prop2) &&
		r.getPropertyIsCustom(prop1) == r.getPropertyIsCustom(prop2) &&
		r.getPropertyEntityID(prop1) == r.getPropertyEntityID(prop2)
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getPropertyName(prop TProp) string {
	switch p := any(prop).(type) {
	case schema.ArtifactProperty:
		return p.Name
	case schema.ContextProperty:
		return p.Name
	case schema.ExecutionProperty:
		return p.Name
	default:
		panic(fmt.Sprintf("unsupported property type: %T", prop))
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getPropertyIsCustom(prop TProp) bool {
	switch p := any(prop).(type) {
	case schema.ArtifactProperty:
		return p.IsCustomProperty
	case schema.ContextProperty:
		return p.IsCustomProperty
	case schema.ExecutionProperty:
		return p.IsCustomProperty
	default:
		panic(fmt.Sprintf("unsupported property type: %T", prop))
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getPropertyEntityID(prop TProp) int32 {
	switch p := any(prop).(type) {
	case schema.ArtifactProperty:
		return p.ArtifactID
	case schema.ContextProperty:
		return p.ContextID
	case schema.ExecutionProperty:
		return p.ExecutionID
	default:
		panic(fmt.Sprintf("unsupported property type: %T", prop))
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) CreateDefaultPaginationToken(e TSchema, listOptions TListOpts) string {
	entityID := r.getEntityID(e)
	orderBy := listOptions.GetOrderBy()
	value := ""

	if orderBy != "" {
		switch orderBy {
		case "ID":
			value = fmt.Sprintf("%d", entityID)
		case "CREATE_TIME":
			value = fmt.Sprintf("%d", r.getCreateTime(e))
		case "LAST_UPDATE_TIME":
			value = fmt.Sprintf("%d", r.getLastUpdateTime(e))
		default:
			value = fmt.Sprintf("%d", entityID)
		}
	}

	return scopes.CreateNextPageToken(entityID, value)
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getCreateTime(e TSchema) int64 {
	switch v := any(e).(type) {
	case schema.Artifact:
		return v.CreateTimeSinceEpoch
	case schema.Context:
		return v.CreateTimeSinceEpoch
	case schema.Execution:
		return v.CreateTimeSinceEpoch
	default:
		return 0
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getLastUpdateTime(e TSchema) int64 {
	switch v := any(e).(type) {
	case schema.Artifact:
		return v.LastUpdateTimeSinceEpoch
	case schema.Context:
		return v.LastUpdateTimeSinceEpoch
	case schema.Execution:
		return v.LastUpdateTimeSinceEpoch
	default:
		return 0
	}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) getNonUpdatableFields(e TSchema) []string {
	return []string{"id", "name", "type_id", "create_time_since_epoch"}
}

func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) GetConfig() GenericRepositoryConfig[TEntity, TSchema, TProp, TListOpts] {
	return r.config
}


func (r *GenericRepository[TEntity, TSchema, TProp, TListOpts]) ApplyStandardPagination(query *gorm.DB, listOptions TListOpts, entities any) *gorm.DB {
	pageSize := listOptions.GetPageSize()
	orderBy := listOptions.GetOrderBy()
	sortOrder := listOptions.GetSortOrder()
	nextPageToken := listOptions.GetNextPageToken()

	pagination := &entity.Pagination{
		PageSize:      &pageSize,
		OrderBy:       &orderBy,
		SortOrder:     &sortOrder,
		NextPageToken: &nextPageToken,
	}

	var tablePrefix string
	var schemaEntity TSchema
	switch any(schemaEntity).(type) {
	case schema.Artifact:
		tablePrefix = "Artifact"
	case schema.Context:
		tablePrefix = "Context"
	case schema.Execution:
		tablePrefix = "Execution"
	default:
		tablePrefix = ""
	}

	return query.Scopes(scopes.PaginateWithTablePrefix(entities, pagination, r.config.DB, tablePrefix))
}
