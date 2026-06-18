package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/db/pagination"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/utils"
	"gorm.io/gorm"
)

var (
	ErrMCPServerNotFound      = errors.New("MCP server by id not found")
	ErrInvalidBaseName        = errors.New("base_name validation failed")
	ErrBaseNameContainsAtSign = errors.New("base_name cannot contain '@' character")
	ErrBaseNameEmpty          = errors.New("base_name cannot be empty")
	ErrBaseNameTooLong        = errors.New("base_name exceeds maximum length of 255 characters")
	ErrDisplayNameTooLong     = errors.New("displayName exceeds maximum length of 255 characters")
	ErrVersionTooLong         = errors.New("version exceeds maximum length of 100 characters")
	ErrVersionContainsAtSign  = errors.New("version cannot contain '@' character")
)

// MCPServerRepositoryImpl implements MCPServerRepository using GORM.
type MCPServerRepositoryImpl struct {
	*service.GenericRepository[models.MCPServer, schema.Context, schema.ContextProperty, *models.MCPServerListOptions]
}

// NewMCPServerRepository creates a new MCPServerRepository.
func NewMCPServerRepository(db *gorm.DB, typeID int32) models.MCPServerRepository {
	r := &MCPServerRepositoryImpl{}

	r.GenericRepository = service.NewGenericRepository(service.GenericRepositoryConfig[models.MCPServer, schema.Context, schema.ContextProperty, *models.MCPServerListOptions]{
		DB:                      db,
		TypeID:                  typeID,
		EntityToSchema:          mapMCPServerToContext,
		SchemaToEntity:          mapDataLayerToMCPServer,
		EntityToProperties:      mapMCPServerToContextProperties,
		NotFoundError:           ErrMCPServerNotFound,
		EntityName:              "MCP server",
		PropertyFieldName:       "context_id",
		ApplyListFilters:        applyMCPServerListFilters,
		CreatePaginationToken:   r.createPaginationToken,
		ApplyCustomOrdering:     r.applyCustomOrdering,
		IsNewEntity:             func(entity models.MCPServer) bool { return entity.GetID() == nil },
		HasCustomProperties:     func(entity models.MCPServer) bool { return entity.GetCustomProperties() != nil },
		EntityMappingFuncs:      NewMCPServerEntityMappings(),
		PreserveHistoricalTimes: true, // Preserve timestamps from YAML source data
	})

	return r
}

// Save creates or updates an MCP server.
// Uses (base_name, version) as the unique identifier.
// Stores composite name (base_name@version) in Context.name field.
func (r *MCPServerRepositoryImpl) Save(server models.MCPServer) (models.MCPServer, error) {
	config := r.GetConfig()
	if server.GetTypeID() == nil {
		if config.TypeID > 0 {
			server.SetTypeID(config.TypeID)
		}
	}

	attr := server.GetAttributes()
	if attr != nil && attr.Name != nil {
		baseName := strings.TrimSpace(*attr.Name)
		version := extractVersionProperty(server)
		displayName := extractDisplayNameProperty(server)

		// Validate base_name
		if baseName == "" {
			return nil, ErrBaseNameEmpty
		}
		if len(baseName) > 255 {
			return nil, fmt.Errorf("%w: length %d", ErrBaseNameTooLong, len(baseName))
		}
		if strings.Contains(baseName, "@") {
			return nil, fmt.Errorf("%w: %s", ErrBaseNameContainsAtSign, baseName)
		}

		// Validate version
		if len(version) > 100 {
			return nil, fmt.Errorf("%w: length %d", ErrVersionTooLong, len(version))
		}
		if strings.Contains(version, "@") {
			return nil, fmt.Errorf("%w: %s", ErrVersionContainsAtSign, version)
		}
		if displayName != nil && len(*displayName) > 255 {
			return nil, fmt.Errorf("%w: length %d", ErrDisplayNameTooLong, len(*displayName))
		}

		// Build composite name for Context.name field
		compositeName := buildCompositeName(baseName, version)

		// Set the composite name in attributes
		attr.Name = &compositeName

		// Add or update base_name property
		props := server.GetProperties()
		if props != nil {
			hasBaseName := false
			var baseNameProp *dbmodels.Properties
			for i := range *props {
				if (*props)[i].Name == "base_name" {
					hasBaseName = true
					baseNameProp = &(*props)[i]
					break
				}
			}
			if hasBaseName {
				// Update if different
				if baseNameProp.StringValue == nil || *baseNameProp.StringValue != baseName {
					baseNameProp.StringValue = &baseName
				}
			} else {
				// Add new property
				*props = append(*props, dbmodels.Properties{
					Name:        "base_name",
					StringValue: &baseName,
				})
			}
		} else {
			// Initialize properties if nil
			newProps := []dbmodels.Properties{
				{
					Name:        "base_name",
					StringValue: &baseName,
				},
			}
			// Use type assertion to set properties on the impl
			if impl, ok := server.(*models.MCPServerImpl); ok {
				impl.Properties = &newProps
			}
		}

		// Check for existing server with same (base_name, version)
		if server.GetID() == nil {
			existing, err := r.GetByNameAndVersion(baseName, version)
			if err != nil {
				if !errors.Is(err, ErrMCPServerNotFound) {
					return nil, fmt.Errorf("error finding existing MCP server named %s version %s: %w", baseName, version, err)
				}
				// If not found, continue with create
			} else {
				// Found existing - update it
				if existing.GetID() != nil {
					server.SetID(*existing.GetID())
				}
			}
		}
	}

	// Attempt to save the server
	saved, err := r.GenericRepository.Save(server, nil)
	if err != nil {
		// Handle race condition: if unique constraint violation occurs,
		// retry by fetching the existing record and updating it
		if dbutil.IsDuplicateKeyError(err) && attr != nil && attr.Name != nil {
			// Extract base name and version from the composite name
			compositeName := *attr.Name
			baseName, version := parseCompositeName(compositeName)

			// Try to get the existing server
			existing, getErr := r.GetByNameAndVersion(baseName, version)
			if getErr == nil && existing.GetID() != nil {
				// Found it - set the ID and retry the save as an update
				server.SetID(*existing.GetID())
				return r.GenericRepository.Save(server, nil)
			}
		}
		return nil, err
	}

	return saved, nil
}

// extractVersionProperty extracts the version property value from an MCP server.
// Returns empty string if no version property exists.
func extractVersionProperty(server models.MCPServer) string {
	if server.GetProperties() == nil {
		return ""
	}

	for _, prop := range *server.GetProperties() {
		if prop.Name == "version" && prop.StringValue != nil {
			return *prop.StringValue
		}
	}

	return ""
}

// extractDisplayNameProperty extracts the displayName property value from an MCP server.
// Returns nil if no displayName property exists.
func extractDisplayNameProperty(server models.MCPServer) *string {
	if server.GetProperties() == nil {
		return nil
	}

	for _, prop := range *server.GetProperties() {
		if prop.Name == "displayName" && prop.StringValue != nil {
			return prop.StringValue
		}
	}

	return nil
}

// buildCompositeName constructs the composite name (name@version) for storage in Context.name.
// If version is empty, returns just the base name.
func buildCompositeName(baseName string, version string) string {
	if version == "" {
		return baseName
	}
	return fmt.Sprintf("%s@%s", baseName, version)
}

// parseCompositeName parses a composite name into base name and version.
// Returns (baseName, version). If no @ separator, returns (name, "").
//
// Note: The validation logic (Save method) now forbids "@" in both base_name and version,
// so this function will only ever encounter a single "@" separator in valid data.
// The multi-@ handling is preserved for backward compatibility with any historical data.
func parseCompositeName(compositeName string) (string, string) {
	parts := strings.Split(compositeName, "@")
	if len(parts) == 1 {
		return compositeName, ""
	}
	// Handle case like "name@v1.0@extra" - take first @ as separator
	// (This case should not occur in new data due to validation)
	baseName := parts[0]
	version := strings.Join(parts[1:], "@")
	return baseName, version
}

// List returns a paginated list of MCP servers.
func (r *MCPServerRepositoryImpl) List(listOptions models.MCPServerListOptions) (*dbmodels.ListWrapper[models.MCPServer], error) {
	return r.GenericRepository.List(&listOptions)
}

// GetByNameAndVersion retrieves an MCP server by its base name and version.
// Uses composite name (base_name@version) stored in Context.name field.
func (r *MCPServerRepositoryImpl) GetByNameAndVersion(baseName string, version string) (models.MCPServer, error) {
	var zeroEntity models.MCPServer
	entity, err := r.lookupServerByNameAndVersion(baseName, version)
	if err != nil {
		return zeroEntity, err
	}

	config := r.GetConfig()

	// Query properties
	var properties []schema.ContextProperty
	if err := config.DB.Where(config.PropertyFieldName+" = ?", entity.ID).Find(&properties).Error; err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return zeroEntity, fmt.Errorf("error getting properties by %s id: %w", config.EntityName, err)
	}

	// Map to domain model
	return config.SchemaToEntity(*entity, properties), nil
}

// lookupServerByNameAndVersion finds an MCP server by base name and version using composite name.
func (r *MCPServerRepositoryImpl) lookupServerByNameAndVersion(baseName string, version string) (*schema.Context, error) {
	config := r.GetConfig()

	// Build composite name
	compositeName := buildCompositeName(baseName, version)

	var entity schema.Context
	if err := config.DB.Where("name = ? AND type_id = ?", compositeName, config.TypeID).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: name=%s, version=%s", config.NotFoundError, baseName, version)
		}
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting %s by name and version: %w", config.EntityName, err)
	}

	return &entity, nil
}

// DeleteBySource deletes all MCP servers from a given source.
func (r *MCPServerRepositoryImpl) DeleteBySource(sourceID string) error {
	config := r.GetConfig()

	// Build subquery to find matching context IDs
	subQuery := config.DB.Table(utils.GetTableName(config.DB, &schema.Context{})).
		Select(utils.GetTableName(config.DB, &schema.Context{})+".id").
		Joins("INNER JOIN "+utils.GetTableName(config.DB, &schema.ContextProperty{})+" ON "+
			utils.GetTableName(config.DB, &schema.Context{})+".id = "+
			utils.GetTableName(config.DB, &schema.ContextProperty{})+".context_id").
		Where(utils.GetTableName(config.DB, &schema.ContextProperty{})+".name = ? AND "+
			utils.GetTableName(config.DB, &schema.ContextProperty{})+".string_value = ? AND "+
			utils.GetTableName(config.DB, &schema.Context{})+".type_id = ?",
			"source_id", sourceID, config.TypeID)

	// Delete contexts with matching IDs
	return config.DB.Where("id IN (?)", subQuery).Delete(&schema.Context{}).Error
}

// DeleteByID deletes an MCP server by its ID.
func (r *MCPServerRepositoryImpl) DeleteByID(id int32) error {
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

// GetTypeID returns the MLMD type ID for the kf.MCPServer context type.
func (r *MCPServerRepositoryImpl) GetTypeID() int32 {
	return r.GetConfig().TypeID
}

// GetDistinctSourceIDs retrieves all unique source_id values from MCP servers.
func (r *MCPServerRepositoryImpl) GetDistinctSourceIDs() ([]string, error) {
	config := r.GetConfig()

	var sourceIDs []string

	contextPropertyTable := utils.GetTableName(config.DB, &schema.ContextProperty{})
	contextTable := utils.GetTableName(config.DB, &schema.Context{})

	err := config.DB.Table(contextPropertyTable+" cp").
		Select("DISTINCT cp.string_value").
		Joins("INNER JOIN "+contextTable+" c ON cp.context_id = c.id").
		Where("cp.name = ? AND c.type_id = ?", "source_id", config.TypeID).
		Pluck("string_value", &sourceIDs).Error

	if err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error querying distinct source IDs: %w", err)
	}

	return sourceIDs, nil
}

// applyMCPServerListFilters applies list filters to the query.
func applyMCPServerListFilters(query *gorm.DB, listOptions *models.MCPServerListOptions) *gorm.DB {
	contextTable := utils.GetTableName(query.Statement.DB, &schema.Context{})

	if listOptions.Name != nil {
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})
		query = query.Where(
			fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name = 'base_name' AND cp.string_value LIKE ?)",
				propertyTable, contextTable),
			listOptions.Name,
		)
	}

	if listOptions.Query != nil && *listOptions.Query != "" {
		queryPattern := fmt.Sprintf("%%%s%%", strings.ToLower(*listOptions.Query))
		propertyTable := utils.GetTableName(query.Statement.DB, &schema.ContextProperty{})

		// Search in name (context table)
		nameCondition := fmt.Sprintf("LOWER(%s.name) LIKE ?", contextTable)

		// Search in displayName, description, and provider properties
		propertyCondition := fmt.Sprintf("EXISTS (SELECT 1 FROM %s cp WHERE cp.context_id = %s.id AND cp.name IN (?, ?, ?) AND LOWER(cp.string_value) LIKE ?)",
			propertyTable, contextTable)

		query = query.Where(fmt.Sprintf("(%s OR %s)", nameCondition, propertyCondition),
			queryPattern,
			"displayName", "description", "provider", queryPattern,
		)
	}

	// Filter by source IDs
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

// mapMCPServerToContext maps an MCPServer entity to a Context schema.
func mapMCPServerToContext(server models.MCPServer) schema.Context {
	attrs := server.GetAttributes()
	context := schema.Context{}

	if typeID := server.GetTypeID(); typeID != nil {
		context.TypeID = *typeID
	}

	if server.GetID() != nil {
		context.ID = *server.GetID()
	}

	if attrs != nil {
		if attrs.Name != nil {
			context.Name = *attrs.Name
		}
		context.ExternalID = attrs.ExternalID
		if attrs.CreateTimeSinceEpoch != nil {
			context.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			context.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
		}
	}

	return context
}

// mapMCPServerToContextProperties maps an MCPServer entity to ContextProperty schema.
func mapMCPServerToContextProperties(server models.MCPServer, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if server.GetProperties() != nil {
		for _, prop := range *server.GetProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	if server.GetCustomProperties() != nil {
		for _, prop := range *server.GetCustomProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, true))
		}
	}

	return properties
}

// mapDataLayerToMCPServer maps database schema to an MCPServer entity.
func mapDataLayerToMCPServer(serverCtx schema.Context, propertiesCtx []schema.ContextProperty) models.MCPServer {
	// Parse composite name to get base name
	baseName, _ := parseCompositeName(serverCtx.Name)

	mcpServer := &models.MCPServerImpl{
		ID:     &serverCtx.ID,
		TypeID: &serverCtx.TypeID,
		Attributes: &models.MCPServerAttributes{
			Name:                     &baseName, // Use base name, not composite
			ExternalID:               serverCtx.ExternalID,
			CreateTimeSinceEpoch:     &serverCtx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &serverCtx.LastUpdateTimeSinceEpoch,
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

	mcpServer.Properties = &properties
	mcpServer.CustomProperties = &customProperties

	return mcpServer
}

// applyCustomOrdering applies custom ordering logic.
func (r *MCPServerRepositoryImpl) applyCustomOrdering(query *gorm.DB, listOptions *models.MCPServerListOptions) *gorm.DB {
	db := r.GetConfig().DB
	contextTable := utils.GetTableName(db, &schema.Context{})
	orderBy := listOptions.GetOrderBy()

	// Handle NAME ordering
	if orderBy == "NAME" {
		return pagination.ApplyNameOrdering(query, contextTable, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false)
	}

	// Fall back to standard pagination
	return r.ApplyStandardPagination(query, listOptions, []models.MCPServer{})
}

// ApplyStandardPagination overrides the base implementation.
func (r *MCPServerRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.MCPServerListOptions, entities any) *gorm.DB {
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

	return query.Scopes(scopes.PaginateWithOptions(entities, pagination, r.GetConfig().DB, "Context", McpOrderByColumns))
}

// createPaginationToken creates a pagination token for the last item.
func (r *MCPServerRepositoryImpl) createPaginationToken(lastItem schema.Context, listOptions *models.MCPServerListOptions) string {
	if listOptions.GetOrderBy() == "NAME" {
		return pagination.CreateNamePaginationToken(lastItem.ID, &lastItem.Name)
	}

	return r.CreateDefaultPaginationToken(lastItem, listOptions)
}

// McpOrderByColumns are the allowed orderBy columns for MCP servers.
var McpOrderByColumns = map[string]string{
	"ID":               "id",
	"CREATE_TIME":      "create_time_since_epoch",
	"LAST_UPDATE_TIME": "last_update_time_since_epoch",
	"NAME":             "name",
	"id":               "id", // default fallback
}
