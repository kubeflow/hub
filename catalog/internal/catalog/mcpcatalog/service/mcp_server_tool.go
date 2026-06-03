package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var (
	ErrMCPServerToolNotFound  = errors.New("MCP server tool by id not found")
	ErrMCPServerToolNameEmpty = errors.New("MCP server tool name cannot be empty")
)

// MCPServerToolRepositoryImpl implements MCPServerToolRepository using GORM with Execution schema.
type MCPServerToolRepositoryImpl struct {
	*service.GenericRepository[models.MCPServerTool, schema.Execution, schema.ExecutionProperty, *models.MCPServerToolListOptions]
}

// NewMCPServerToolRepository creates a new MCPServerToolRepository.
func NewMCPServerToolRepository(db *gorm.DB, typeID int32) models.MCPServerToolRepository {
	config := service.GenericRepositoryConfig[models.MCPServerTool, schema.Execution, schema.ExecutionProperty, *models.MCPServerToolListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapMCPServerToolToExecution,
		SchemaToEntity:          mapDataLayerToMCPServerTool,
		EntityToProperties:      mapMCPServerToolToExecutionProperties,
		NotFoundError:           ErrMCPServerToolNotFound,
		EntityName:              "MCP server tool",
		PropertyFieldName:       "execution_id",
		ApplyListFilters:        applyMCPServerToolListFilters,
		IsNewEntity:             func(entity models.MCPServerTool) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.MCPServerTool) bool { return entity.GetCustomProperties() != nil },
		EntityMappingFuncs:      NewMCPServerToolEntityMappings(),
		PreserveHistoricalTimes: true,
	}

	return &MCPServerToolRepositoryImpl{
		GenericRepository: service.NewGenericRepository(config),
	}
}

// GetByID retrieves an MCP server tool by its ID.
func (r *MCPServerToolRepositoryImpl) GetByID(id int32) (models.MCPServerTool, error) {
	return r.GenericRepository.GetByID(id)
}

// List retrieves all tools for a given parent MCP server with optional filterQuery and pagination support.
func (r *MCPServerToolRepositoryImpl) List(listOptions models.MCPServerToolListOptions) (*dbmodels.ListWrapper[models.MCPServerTool], error) {
	return r.GenericRepository.List(&listOptions)
}

// applyMCPServerToolListFilters adds the Association join and parent ID filter so that
// GenericRepository.List() returns only the tools belonging to a specific MCP server.
// Note: type_id filtering is already applied by buildBaseQuery().
func applyMCPServerToolListFilters(query *gorm.DB, listOptions *models.MCPServerToolListOptions) *gorm.DB {
	associationTable := utils.GetTableName(query.Statement.DB, &schema.Association{})
	executionTable := utils.GetTableName(query.Statement.DB, &schema.Execution{})

	query = query.
		Joins(fmt.Sprintf("INNER JOIN %s ON %s.execution_id = %s.id",
			associationTable, associationTable, executionTable)).
		Where(fmt.Sprintf("%s.context_id = ?", associationTable), listOptions.ParentID)

	if listOptions.ToolName != nil {
		escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(*listOptions.ToolName)
		query = query.Where(fmt.Sprintf("%s.name LIKE ?", executionTable), fmt.Sprintf("%%:%s", escaped))
	}

	return query
}

// Save creates or updates an MCP server tool.
func (r *MCPServerToolRepositoryImpl) Save(tool models.MCPServerTool, parentID *int32) (models.MCPServerTool, error) {
	config := r.GetConfig()
	if tool.GetTypeID() == nil {
		if config.TypeID > 0 {
			tool.SetTypeID(config.TypeID)
		}
	}

	// Validate tool name
	attrs := tool.GetAttributes()
	if attrs == nil || attrs.Name == nil || *attrs.Name == "" {
		return nil, ErrMCPServerToolNameEmpty
	}

	return r.GenericRepository.Save(tool, parentID)
}

// CountByParentIDs returns the tool counts for multiple parent MCP servers in a single query.
// The returned map keys are parent IDs; parents with zero tools are included with count 0.
func (r *MCPServerToolRepositoryImpl) CountByParentIDs(parentIDs []int32) (map[int32]int32, error) {
	result := make(map[int32]int32, len(parentIDs))
	if len(parentIDs) == 0 {
		return result, nil
	}

	// Initialize all requested IDs to 0
	for _, id := range parentIDs {
		result[id] = 0
	}

	config := r.GetConfig()
	associationTable := utils.GetTableName(config.DB, &schema.Association{})
	executionTable := utils.GetTableName(config.DB, &schema.Execution{})

	type countRow struct {
		ContextID int32 `gorm:"column:context_id"`
		Count     int32 `gorm:"column:count"`
	}

	var rows []countRow
	err := config.DB.Table(executionTable).
		Select(fmt.Sprintf("%s.context_id, COUNT(*) as count", associationTable)).
		Joins(fmt.Sprintf("INNER JOIN %s ON %s.execution_id = %s.id",
			associationTable, associationTable, executionTable)).
		Where(fmt.Sprintf("%s.context_id IN ? AND %s.type_id = ?",
			associationTable, executionTable), parentIDs, config.TypeID).
		Group(fmt.Sprintf("%s.context_id", associationTable)).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("error counting %s by parents: %w", config.EntityName, err)
	}

	for _, row := range rows {
		result[row.ContextID] = row.Count
	}

	return result, nil
}

// DeleteByParentID deletes all tools belonging to a parent MCP server.
func (r *MCPServerToolRepositoryImpl) DeleteByParentID(parentID int32) error {
	config := r.GetConfig()

	// Find all execution IDs linked to this parent context via Association
	associationTable := utils.GetTableName(config.DB, &schema.Association{})
	executionTable := utils.GetTableName(config.DB, &schema.Execution{})

	subQuery := config.DB.Table(associationTable).
		Select(associationTable+".execution_id").
		Joins(fmt.Sprintf("INNER JOIN %s ON %s.execution_id = %s.id",
			executionTable, associationTable, executionTable)).
		Where(fmt.Sprintf("%s.context_id = ? AND %s.type_id = ?",
			associationTable, executionTable), parentID, config.TypeID)

	// Delete executions matching the subquery
	result := config.DB.Where("id IN (?)", subQuery).Delete(&schema.Execution{})
	if result.Error != nil {
		return fmt.Errorf("error deleting %s by parent: %w", config.EntityName, result.Error)
	}

	return nil
}

// DeleteByID deletes an MCP server tool by its ID.
func (r *MCPServerToolRepositoryImpl) DeleteByID(id int32) error {
	config := r.GetConfig()

	result := config.DB.Where("id = ? AND type_id = ?", id, config.TypeID).Delete(&schema.Execution{})

	if result.Error != nil {
		return fmt.Errorf("error deleting %s: %w", config.EntityName, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: id %d", config.NotFoundError, id)
	}

	return nil
}

// mapMCPServerToolToExecution maps an MCPServerTool entity to Execution schema.
// Note: Tool name validation is enforced in the Save method before this function is called.
func mapMCPServerToolToExecution(tool models.MCPServerTool) schema.Execution {
	attrs := tool.GetAttributes()
	execution := schema.Execution{}

	if typeID := tool.GetTypeID(); typeID != nil {
		execution.TypeID = *typeID
	}

	if tool.GetID() != nil {
		execution.ID = *tool.GetID()
	}

	if attrs != nil && attrs.Name != nil {
		execution.Name = attrs.Name
		if attrs.CreateTimeSinceEpoch != nil {
			execution.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			execution.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return execution
}

// mapMCPServerToolToExecutionProperties maps an MCPServerTool entity to ExecutionProperty schema.
func mapMCPServerToolToExecutionProperties(tool models.MCPServerTool, executionID int32) []schema.ExecutionProperty {
	var properties []schema.ExecutionProperty

	if tool.GetProperties() != nil {
		for _, prop := range *tool.GetProperties() {
			properties = append(properties, service.MapPropertiesToExecutionProperty(prop, executionID, false))
		}
	}

	if tool.GetCustomProperties() != nil {
		for _, prop := range *tool.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToExecutionProperty(prop, executionID, true))
		}
	}

	return properties
}

// mapDataLayerToMCPServerTool maps database schema to an MCPServerTool entity.
func mapDataLayerToMCPServerTool(execution schema.Execution, properties []schema.ExecutionProperty) models.MCPServerTool {
	tool := &models.MCPServerToolImpl{
		ID:     &execution.ID,
		TypeID: &execution.TypeID,
		Attributes: &models.MCPServerToolAttributes{
			Name:                     execution.Name,
			CreateTimeSinceEpoch:     &execution.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &execution.LastUpdateTimeSinceEpoch,
		},
	}

	modelProperties := []dbmodels.Properties{}
	customProperties := []dbmodels.Properties{}

	for _, prop := range properties {
		mappedProperty := service.MapExecutionPropertyToProperties(prop)

		if prop.IsCustomProperty {
			customProperties = append(customProperties, mappedProperty)
		} else {
			modelProperties = append(modelProperties, mappedProperty)
		}
	}

	tool.Properties = &modelProperties
	tool.CustomProperties = &customProperties

	return tool
}
