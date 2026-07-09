package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/pagination"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

const AgentTypeName = "kf.Agent"

var ErrAgentNotFound = errors.New("agent not found")

// AgentRepositoryImpl implements AgentRepository using GORM.
type AgentRepositoryImpl struct {
	*service.GenericRepository[models.Agent, schema.Context, schema.ContextProperty, *models.AgentListOptions]
}

// NewAgentRepository creates a new AgentRepository.
func NewAgentRepository(db *gorm.DB, typeID int32) models.AgentRepository {
	r := &AgentRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[models.Agent, schema.Context, schema.ContextProperty, *models.AgentListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapAgentToSchema,
		SchemaToEntity:          mapSchemaToAgent,
		EntityToProperties:      mapAgentToProperties,
		NotFoundError:           ErrAgentNotFound,
		EntityName:              "agent",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applyAgentListFilters,
		CreatePaginationToken:   r.createAgentPaginationToken,
		ApplyCustomOrdering:     r.applyAgentCustomOrdering,
		IsNewEntity:             func(entity models.Agent) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.Agent) bool { return entity.GetCustomProperties() != nil },
		EntityMappingFuncs:      newAgentEntityMappings(),
		PreserveHistoricalTimes: true,
	})

	return r
}

// Save creates or updates an agent, ensuring TypeID is set.
// If the entity has no ID but has a name, it looks up an existing agent by name
// to enable upsert (update on reload instead of duplicate-key error).
func (r *AgentRepositoryImpl) Save(entity models.Agent) (models.Agent, error) {
	config := r.GetConfig()
	if entity.GetTypeID() == nil && config.TypeID > 0 {
		entity.SetTypeID(config.TypeID)
	}

	attr := entity.GetAttributes()
	if entity.GetID() == nil && attr != nil && attr.Name != nil {
		existing, err := r.lookupAgentByName(*attr.Name)
		if err != nil {
			if !errors.Is(err, ErrAgentNotFound) {
				return nil, fmt.Errorf("error finding existing agent named %s: %w", *attr.Name, err)
			}
		} else {
			entity.SetID(existing.ID)
		}
	}

	return r.GenericRepository.Save(entity, nil)
}

func (r *AgentRepositoryImpl) lookupAgentByName(name string) (*schema.Context, error) {
	config := r.GetConfig()
	var entity schema.Context
	if err := config.DB.Where("name = ? AND type_id = ?", name, config.TypeID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: name=%s", ErrAgentNotFound, name)
		}
		return nil, fmt.Errorf("error querying agent by name: %w", err)
	}
	return &entity, nil
}

func mapAgentToSchema(agent models.Agent) schema.Context {
	attrs := agent.GetAttributes()
	ctx := schema.Context{}
	if typeID := agent.GetTypeID(); typeID != nil {
		ctx.TypeID = *typeID
	}
	if agent.GetID() != nil {
		ctx.ID = *agent.GetID()
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

func mapSchemaToAgent(agentCtx schema.Context, propertiesCtx []schema.ContextProperty) models.Agent {
	agent := &models.AgentImpl{
		ID:     &agentCtx.ID,
		TypeID: &agentCtx.TypeID,
		Attributes: &models.AgentAttributes{
			Name:                     &agentCtx.Name,
			ExternalID:               agentCtx.ExternalID,
			CreateTimeSinceEpoch:     &agentCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &agentCtx.LastUpdateTimeSinceEpoch,
		},
	}

	properties := []dbmodels.Properties{}
	customProperties := []dbmodels.Properties{}
	for _, prop := range propertiesCtx {
		mapped := service.MapContextPropertyToProperties(prop)
		if prop.IsCustomProperty {
			customProperties = append(customProperties, mapped)
		} else {
			properties = append(properties, mapped)
		}
	}
	agent.Properties = &properties
	agent.CustomProperties = &customProperties
	return agent
}

func mapAgentToProperties(agent models.Agent, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty
	if agent.GetProperties() != nil {
		for _, prop := range *agent.GetProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, false))
		}
	}
	if agent.GetCustomProperties() != nil {
		for _, prop := range *agent.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, true))
		}
	}
	return properties
}

func applyAgentListFilters(query *gorm.DB, listOptions *models.AgentListOptions) *gorm.DB {
	contextTable := utils.GetTableName(query.Statement.DB, &schema.Context{})

	if listOptions.Name != nil {
		query = query.Where(fmt.Sprintf("%s.name LIKE ?", contextTable), fmt.Sprintf("%%:%s", *listOptions.Name))
	}

	if listOptions.Query != nil && *listOptions.Query != "" {
		queryPattern := fmt.Sprintf("%%%s%%", strings.ToLower(*listOptions.Query))
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})

		nameCondition := fmt.Sprintf("LOWER(%s.name) LIKE ?", contextTable)

		propertyCondition := fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name IN (?, ?, ?) AND LOWER(cp.string_value) LIKE ?)",
			propertyTable, contextTable)

		query = query.Where(fmt.Sprintf("(%s OR %s)", nameCondition, propertyCondition),
			queryPattern,
			"description", "displayName", "framework", queryPattern,
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
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})
		joinClause := fmt.Sprintf("JOIN %s cp ON cp.context_id = %s.id", propertyTable, contextTable)
		query = query.Joins(joinClause).
			Where("cp.name = ? AND cp.string_value IN ?", "source_id", nonEmptySourceIDs)
	}

	return query
}

func (r *AgentRepositoryImpl) createAgentPaginationToken(lastItem schema.Context, listOptions *models.AgentListOptions) string {
	if listOptions.GetOrderBy() == "NAME" {
		return pagination.CreateNamePaginationToken(lastItem.ID, &lastItem.Name)
	}
	return r.CreateDefaultPaginationToken(lastItem, listOptions)
}

func (r *AgentRepositoryImpl) applyAgentCustomOrdering(query *gorm.DB, listOptions *models.AgentListOptions) *gorm.DB {
	db := r.GetConfig().DB
	contextTable := utils.GetTableName(db, &schema.Context{})
	orderBy := listOptions.GetOrderBy()

	if orderBy == "NAME" {
		return pagination.ApplyNameOrdering(query, contextTable, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false)
	}

	return r.ApplyStandardPagination(query, listOptions, []models.Agent{})
}

// ApplyStandardPagination overrides the base implementation.
func (r *AgentRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.AgentListOptions, entities any) *gorm.DB {
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

	return query.Scopes(scopes.PaginateWithOptions(entities, pag, r.GetConfig().DB, "Context", AgentOrderByColumns))
}

// AgentOrderByColumns are the allowed orderBy columns for agents.
var AgentOrderByColumns = map[string]string{
	"ID":               "id",
	"CREATE_TIME":      "create_time_since_epoch",
	"LAST_UPDATE_TIME": "last_update_time_since_epoch",
	"NAME":             "name",
	"id":               "id",
}

// DeleteBySource deletes all agents from a given source.
func (r *AgentRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()
	tableName := utils.GetTableName(config.DB, &schema.Context{})
	propTableName := utils.GetTableName(config.DB, &schema.ContextProperty{})

	subQuery := config.DB.Table(tableName).
		Select(tableName + ".id").
		Joins("INNER JOIN " + propTableName + " ON " +
			tableName + ".id = " + propTableName + ".context_id").
		Where(propTableName + ".name = ? AND " +
			propTableName + ".string_value = ? AND " +
			tableName + ".type_id = ?",
			"source_id", sourceID, config.TypeID)

	return config.DB.Where("id IN (?)", subQuery).Delete(&schema.Context{}).Error
}

// DeleteByID deletes an agent by its ID.
func (r *AgentRepositoryImpl) DeleteByID(id int32) error {
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

// GetDistinctSourceIDs retrieves all unique source_id values from agents.
func (r *AgentRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	config := r.GetConfig()
	var sourceIDs []string

	propTableName := utils.GetTableName(config.DB, &schema.ContextProperty{})
	tableName := utils.GetTableName(config.DB, &schema.Context{})

	err := config.DB.Table(propTableName + " cp").
		Select("DISTINCT cp.string_value").
		Joins("INNER JOIN " + tableName + " c ON cp.context_id = c.id").
		Where("cp.name = ? AND c.type_id = ?", "source_id", config.TypeID).
		Pluck("string_value", &sourceIDs).Error

	if err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
	}
	return sourceIDs, nil
}

func (r *AgentRepositoryImpl) GetTypeID() int32 {
	return r.GetConfig().TypeID
}
