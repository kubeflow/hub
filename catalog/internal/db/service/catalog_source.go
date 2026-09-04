package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/kubeflow/hub/catalog/internal/db/models"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	service "github.com/kubeflow/hub/internal/platform/db/repository"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"gorm.io/gorm"
)

var ErrCatalogSourceNotFound = errors.New("catalog source not found")

// CatalogSourceRepositoryImpl implements CatalogSourceRepository using GORM.
type CatalogSourceRepositoryImpl struct {
	db     *gorm.DB
	typeID int32
}

// NewCatalogSourceRepository creates a new CatalogSourceRepository.
func NewCatalogSourceRepository(db *gorm.DB, typeID int32) models.CatalogSourceRepository {
	return &CatalogSourceRepositoryImpl{
		db:     db,
		typeID: typeID,
	}
}

// GetBySourceID retrieves a catalog source by its source ID.
func (r *CatalogSourceRepositoryImpl) GetBySourceID(sourceID string) (models.CatalogSource, error) {
	var context schema.Context

	if err := r.db.Where("name = ? AND type_id = ?", sourceID, r.typeID).First(&context).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrCatalogSourceNotFound, sourceID)
		}
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting catalog source by id: %w", err)
	}

	// Get properties
	var properties []schema.ContextProperty
	if err := r.db.Where("context_id = ?", context.ID).Find(&properties).Error; err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting catalog source properties: %w", err)
	}

	return r.mapSchemaToEntity(context, properties), nil
}

// Save creates or updates a catalog source.
func (r *CatalogSourceRepositoryImpl) Save(source models.CatalogSource) (models.CatalogSource, error) {
	if source.GetTypeID() == nil {
		source.SetTypeID(r.typeID)
	}

	attrs := source.GetAttributes()
	if attrs == nil || attrs.Name == nil {
		return nil, errors.New("source ID (name) is required")
	}

	now := time.Now().UnixMilli()

	var savedContext schema.Context

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Check if exists
		var existing schema.Context
		err := tx.Where("name = ? AND type_id = ?", *attrs.Name, r.typeID).First(&existing).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("error checking existing catalog source: %w", err)
		}

		savedContext = schema.Context{
			TypeID:                   r.typeID,
			Name:                     *attrs.Name,
			LastUpdateTimeSinceEpoch: now,
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			savedContext.CreateTimeSinceEpoch = now
			if err := tx.Create(&savedContext).Error; err != nil {
				return fmt.Errorf("error creating catalog source: %w", err)
			}
		} else {
			// Update existing
			savedContext.ID = existing.ID
			savedContext.CreateTimeSinceEpoch = existing.CreateTimeSinceEpoch
			if err := tx.Save(&savedContext).Error; err != nil {
				return fmt.Errorf("error updating catalog source: %w", err)
			}

			// Delete old properties
			if err := tx.Where("context_id = ?", savedContext.ID).Delete(&schema.ContextProperty{}).Error; err != nil {
				return fmt.Errorf("error deleting old catalog source properties: %w", err)
			}
		}

		// Save properties
		properties := r.mapEntityToProperties(source, savedContext.ID)
		if len(properties) > 0 {
			if err := tx.Create(&properties).Error; err != nil {
				return fmt.Errorf("error saving catalog source properties: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, dbutil.SanitizeDatabaseError(err)
	}

	// Update the source with the ID and timestamps from the saved context
	source.SetID(savedContext.ID)
	attrs.CreateTimeSinceEpoch = &savedContext.CreateTimeSinceEpoch
	attrs.LastUpdateTimeSinceEpoch = &savedContext.LastUpdateTimeSinceEpoch

	return source, nil
}

// Delete removes a catalog source by its source ID.
func (r *CatalogSourceRepositoryImpl) Delete(sourceID string) error {
	result := r.db.Where("name = ? AND type_id = ?", sourceID, r.typeID).Delete(&schema.Context{})
	if result.Error != nil {
		err := dbutil.SanitizeDatabaseError(result.Error)
		return fmt.Errorf("error deleting catalog source: %w", err)
	}
	return nil
}

// GetAll retrieves all catalog sources.
func (r *CatalogSourceRepositoryImpl) GetAll() ([]models.CatalogSource, error) {
	var contexts []schema.Context
	if err := r.db.Where("type_id = ?", r.typeID).Find(&contexts).Error; err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting all catalog sources: %w", err)
	}

	if len(contexts) == 0 {
		return []models.CatalogSource{}, nil
	}

	// Get all context IDs
	contextIDs := make([]int32, len(contexts))
	for i, ctx := range contexts {
		contextIDs[i] = ctx.ID
	}

	// Get all properties for these contexts
	var allProperties []schema.ContextProperty
	if err := r.db.Where("context_id IN ?", contextIDs).Find(&allProperties).Error; err != nil {
		err = dbutil.SanitizeDatabaseError(err)
		return nil, fmt.Errorf("error getting catalog source properties: %w", err)
	}

	// Group properties by context ID
	propsByContext := make(map[int32][]schema.ContextProperty)
	for _, prop := range allProperties {
		propsByContext[prop.ContextID] = append(propsByContext[prop.ContextID], prop)
	}

	// Map to entities
	result := make([]models.CatalogSource, len(contexts))
	for i, ctx := range contexts {
		result[i] = r.mapSchemaToEntity(ctx, propsByContext[ctx.ID])
	}

	return result, nil
}

// GetAllStatuses returns a map of source ID to status/error for all sources.
func (r *CatalogSourceRepositoryImpl) GetAllStatuses() (map[string]models.SourceStatus, error) {
	sources, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string]models.SourceStatus, len(sources))
	for _, source := range sources {
		attrs := source.GetAttributes()
		if attrs == nil || attrs.Name == nil {
			continue
		}

		result[*attrs.Name] = extractStatus(source)
	}

	return result, nil
}

// GetStatus returns the status/error persisted for a single source. If the source does not exist,
// it returns an empty SourceStatus rather than an error, since callers treat "no persisted status"
// and "unknown source" the same way.
func (r *CatalogSourceRepositoryImpl) GetStatus(sourceID string) (models.SourceStatus, error) {
	source, err := r.GetBySourceID(sourceID)
	if err != nil {
		if errors.Is(err, ErrCatalogSourceNotFound) {
			return models.SourceStatus{}, nil
		}
		return models.SourceStatus{}, err
	}

	return extractStatus(source), nil
}

// extractStatus reads the "status" and "error" properties off a catalog source.
func extractStatus(source models.CatalogSource) models.SourceStatus {
	status := models.SourceStatus{}

	if props := source.GetProperties(); props != nil {
		for _, prop := range *props {
			switch prop.Name {
			case "status":
				if prop.StringValue != nil {
					status.Status = *prop.StringValue
				}
			case "error":
				if prop.StringValue != nil {
					status.Error = *prop.StringValue
				}
			}
		}
	}

	return status
}

// mapSchemaToEntity converts database schema to domain model.
func (r *CatalogSourceRepositoryImpl) mapSchemaToEntity(ctx schema.Context, properties []schema.ContextProperty) models.CatalogSource {
	source := &models.CatalogSourceImpl{
		ID:     &ctx.ID,
		TypeID: &ctx.TypeID,
		Attributes: &models.CatalogSourceAttributes{
			Name:                     &ctx.Name,
			CreateTimeSinceEpoch:     &ctx.CreateTimeSinceEpoch,
			LastUpdateTimeSinceEpoch: &ctx.LastUpdateTimeSinceEpoch,
		},
	}

	// Map properties
	props := make([]dbmodels.Properties, 0, len(properties))
	for _, prop := range properties {
		props = append(props, service.MapContextPropertyToProperties(prop))
	}
	source.Properties = &props

	return source
}

// mapEntityToProperties converts entity properties to database schema.
func (r *CatalogSourceRepositoryImpl) mapEntityToProperties(source models.CatalogSource, contextID int32) []schema.ContextProperty {
	var properties []schema.ContextProperty

	if source.GetProperties() != nil {
		for _, prop := range *source.GetProperties() {
			properties = append(properties, service.MapPropertiesToContextProperty(prop, contextID, false))
		}
	}

	return properties
}
