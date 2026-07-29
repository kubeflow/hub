package service

import (
	"errors"
	"fmt"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/pagination"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrSkillNotFound = errors.New("skill not found")

// SkillRepositoryImpl implements SkillRepository using GORM.
type SkillRepositoryImpl struct {
	*service.GenericRepository[models.Skill, schema.Context, schema.ContextProperty, *models.SkillListOptions]
}

// NewSkillRepository creates a new SkillRepository.
func NewSkillRepository(db *gorm.DB, typeID int32) models.SkillRepository {
	r := &SkillRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[models.Skill, schema.Context, schema.ContextProperty, *models.SkillListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapSkillToSchema,
		SchemaToEntity:          mapSchemaToSkill,
		EntityToProperties:      mapSkillToProperties,
		NotFoundError:           ErrSkillNotFound,
		EntityName:              "skill",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applySkillListFilters,
		CreatePaginationToken:   r.createSkillPaginationToken,
		ApplyCustomOrdering:     r.applySkillCustomOrdering,
		IsNewEntity:             func(entity models.Skill) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.Skill) bool { return entity.GetCustomProperties() != nil },
		EntityMappingFuncs:      newSkillEntityMappings(),
		PreserveHistoricalTimes: true,
	})

	return r
}

// Save creates or updates a skill, ensuring the type ID is set so the record is
// discoverable by List/Get queries.
func (r *SkillRepositoryImpl) Save(entity models.Skill) (models.Skill, error) {
	config := r.GetConfig()
	if entity.GetTypeID() == nil && config.TypeID > 0 {
		entity.SetTypeID(config.TypeID)
	}
	return r.GenericRepository.Save(entity, nil)
}

// mapSkillToSchema maps a Skill entity to a Context schema.
func mapSkillToSchema(entity models.Skill) schema.Context {
	attrs := entity.GetAttributes()
	ctx := schema.Context{}

	if typeID := entity.GetTypeID(); typeID != nil {
		ctx.TypeID = *typeID
	}
	if entity.GetID() != nil {
		ctx.ID = *entity.GetID()
	}
	if attrs != nil {
		if attrs.Name != nil {
			ctx.Name = *attrs.Name
		}
		ctx.ExternalID = attrs.ExternalID
		if attrs.CreateTimeSinceEpoch != nil {
			ctx.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			ctx.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return ctx
}

// mapSchemaToSkill maps a Context schema and its properties to a Skill entity.
func mapSchemaToSkill(ctx schema.Context, props []schema.ContextProperty) models.Skill {
	entity := &models.SkillImpl{
		ID:     &ctx.ID,
		TypeID: &ctx.TypeID,
		Attributes: &models.SkillAttributes{
			Name:                     &ctx.Name,
			ExternalID:               ctx.ExternalID,
			CreateTimeSinceEpoch:     &ctx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &ctx.LastUpdateTimeSinceEpoch,
		},
	}

	properties := []dbmodels.Properties{}
	customProperties := []dbmodels.Properties{}
	for _, prop := range props {
		mapped := service.MapContextPropertyToProperties(prop)
		if prop.IsCustomProperty {
			customProperties = append(customProperties, mapped)
		} else {
			properties = append(properties, mapped)
		}
	}
	entity.Properties = &properties
	entity.CustomProperties = &customProperties

	return entity
}

// mapSkillToProperties maps a Skill entity's properties to ContextProperty schema.
func mapSkillToProperties(entity models.Skill, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty
	if entity.GetProperties() != nil {
		for _, prop := range *entity.GetProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, false))
		}
	}
	if entity.GetCustomProperties() != nil {
		for _, prop := range *entity.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, true))
		}
	}
	return properties
}

// applySkillListFilters applies list filters to the query. Filtering is a no-op
// for now; entity-specific filters are added with the query API (SKC-108).
func applySkillListFilters(query *gorm.DB, _ *models.SkillListOptions) *gorm.DB {
	return query
}

func (r *SkillRepositoryImpl) createSkillPaginationToken(lastItem schema.Context, listOptions *models.SkillListOptions) string {
	if listOptions.GetOrderBy() == "NAME" {
		return pagination.CreateNamePaginationToken(lastItem.ID, &lastItem.Name)
	}
	return r.CreateDefaultPaginationToken(lastItem, listOptions)
}

// SkillOrderByColumns are the allowed orderBy columns for skills.
var SkillOrderByColumns = map[string]string{
	"ID":               "id",
	"CREATE_TIME":      "create_time_since_epoch",
	"LAST_UPDATE_TIME": "last_update_time_since_epoch",
	"NAME":             "name",
	"id":               "id",
}

// applySkillCustomOrdering applies ordering and pagination to the query.
func (r *SkillRepositoryImpl) applySkillCustomOrdering(query *gorm.DB, listOptions *models.SkillListOptions) *gorm.DB {
	db := r.GetConfig().DB
	contextTable := utils.GetTableName(db, &schema.Context{})
	orderBy := listOptions.GetOrderBy()

	if orderBy == "NAME" {
		return pagination.ApplyNameOrdering(query, contextTable, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false)
	}

	return r.ApplyStandardPagination(query, listOptions, []models.Skill{})
}

// ApplyStandardPagination overrides the base implementation to pass the skill
// orderBy columns to the pagination scope.
func (r *SkillRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.SkillListOptions, entities any) *gorm.DB {
	pageSize := listOptions.GetPageSize()
	orderBy := listOptions.GetOrderBy()
	sortOrder := listOptions.GetSortOrder()
	nextPageToken := listOptions.GetNextPageToken()

	pag := &dbmodels.Pagination{
		PageSize:      &pageSize,
		OrderBy:       &orderBy,
		SortOrder:     &sortOrder,
		NextPageToken: &nextPageToken,
	}

	return query.Scopes(scopes.PaginateWithOptions(entities, pag, r.GetConfig().DB, "Context", SkillOrderByColumns))
}

// DeleteBySource deletes all skills from a given source.
func (r *SkillRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()
	tableName := utils.GetTableName(config.DB, &schema.Context{})
	propTableName := utils.GetTableName(config.DB, &schema.ContextProperty{})

	subQuery := config.DB.Table(tableName).
		Select(tableName+".id").
		Joins("INNER JOIN "+propTableName+" ON "+
			tableName+".id = "+propTableName+".context_id").
		Where(propTableName+".name = ? AND "+
			propTableName+".string_value = ? AND "+
			tableName+".type_id = ?",
			"source_id", sourceID, config.TypeID)

	return config.DB.Where("id IN (?)", subQuery).Delete(&schema.Context{}).Error
}

// DeleteByID deletes a skill by its ID.
func (r *SkillRepositoryImpl) DeleteByID(id int32) error {
	config := r.GetConfig()
	result := config.DB.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&schema.Context{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: id %d", config.NotFoundError, id)
	}
	return nil
}

// GetDistinctSourceIDs retrieves all unique source_id values from skills.
func (r *SkillRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	config := r.GetConfig()
	var sourceIDs []string

	propTableName := utils.GetTableName(config.DB, &schema.ContextProperty{})
	tableName := utils.GetTableName(config.DB, &schema.Context{})

	err := config.DB.Table(propTableName+" cp").
		Select("DISTINCT cp.string_value").
		Joins("INNER JOIN "+tableName+" c ON cp.context_id = c.id").
		Where("cp.name = ? AND c.type_id = ?", "source_id", config.TypeID).
		Pluck("string_value", &sourceIDs).Error

	if err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
	}
	return sourceIDs, nil
}

// GetTypeID returns the datastore type ID for the skill context type.
func (r *SkillRepositoryImpl) GetTypeID() int32 {
	return r.GetConfig().TypeID
}
