package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/pagination"
	catalogsvc "github.com/kubeflow/hub/catalog/internal/db/service"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var ErrSkillNotFound = errors.New("skill not found")

// SkillTypeName is the datastore Context type name for skill entities.
const SkillTypeName = "kf.Skill"

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
	// Upsert by composite name: if a skill with this name already exists, update it
	// in place rather than inserting a duplicate that would violate the Context
	// UNIQUE(type_id, name) constraint. This lets the loader reconcile a source by
	// re-saving its current skills without first deleting them all, so reads never
	// see the source empty mid-sync.
	if entity.GetID() == nil {
		if attrs := entity.GetAttributes(); attrs != nil && attrs.Name != nil {
			if existing, err := r.GetByName(*attrs.Name); err == nil && existing.GetID() != nil {
				entity.SetID(*existing.GetID())
			}
		}
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

// applySkillListFilters applies the name/q/source list parameters to the query.
// filterQuery is applied separately by the generic repository via FilterApplier.
func applySkillListFilters(query *gorm.DB, listOptions *models.SkillListOptions) *gorm.DB {
	contextTable := utils.GetTableName(query.Statement.DB, &schema.Context{})
	propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})

	// Name is passed through as-is: the caller supplies the SQL LIKE pattern
	// (per the OpenAPI description), matching the mcpcatalog convention.
	// is_custom_property = false excludes a same-named customProperty (e.g. a
	// SKILL.md frontmatter metadata key literally called "name", which
	// customPropertiesFromMetadata does not guard against) — without it, a
	// colliding custom property's value could satisfy the LIKE pattern and
	// produce a false-positive match.
	if listOptions.Name != nil {
		query = query.Where(
			fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name = ? AND cp.is_custom_property = ? AND cp.string_value LIKE ?)", propertyTable, contextTable),
			"name", false, *listOptions.Name,
		)
	}

	// q searches name, description, and readme, per the KEP execution plan (SKC-108).
	// Same is_custom_property = false guard as above, applied to all three fields.
	if listOptions.Query != nil && *listOptions.Query != "" {
		queryPattern := fmt.Sprintf("%%%s%%", strings.ToLower(*listOptions.Query))
		query = query.Where(
			fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name IN (?, ?, ?) AND cp.is_custom_property = ? AND LOWER(cp.string_value) LIKE ?)", propertyTable, contextTable),
			"name", "description", "readme", false, queryPattern,
		)
	}

	var nonEmptySourceIDs []string
	if listOptions.SourceIDs != nil {
		for _, sourceID := range *listOptions.SourceIDs {
			if sourceID != "" {
				nonEmptySourceIDs = append(nonEmptySourceIDs, sourceID)
			}
		}
	}

	if len(nonEmptySourceIDs) > 0 {
		// is_custom_property = false excludes a same-named customProperty (e.g. a
		// SKILL.md frontmatter metadata key literally called "source_id", which
		// customPropertiesFromMetadata does not guard against) — without it, a
		// colliding row would duplicate the skill in the joined result set.
		joinClause := fmt.Sprintf("JOIN %s cp ON cp.context_id = %s.id", propertyTable, contextTable)
		query = query.Joins(joinClause).
			Where("cp.name = ? AND cp.is_custom_property = ? AND cp.string_value IN ?", "source_id", false, nonEmptySourceIDs)
	}

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

// GetTypeID returns the datastore type ID for the skill context type.
func (r *SkillRepositoryImpl) GetTypeID() int32 {
	return r.GetConfig().TypeID
}

func (r *SkillRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	cfg := r.GetConfig()
	entityTable := utils.GetTableName(cfg.DB, &schema.Context{})
	propTable := utils.GetTableName(cfg.DB, &schema.ContextProperty{})
	return catalogsvc.GetDistinctSourceIDs(cfg.DB, entityTable, propTable, cfg.PropertyFieldName, cfg.TypeID)
}
