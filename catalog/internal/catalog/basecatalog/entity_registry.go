package basecatalog

import (
	"strings"
	"sync"

	"github.com/kubeflow/hub/internal/platform/db/filter"
)

// EntityTypeDefinition describes a single catalog entity type's filtering behavior.
type EntityTypeDefinition struct {
	// MLMDEntityType is the underlying MLMD entity type (Context, Artifact, Execution).
	MLMDEntityType filter.EntityType

	// Properties maps REST property names to their definitions.
	Properties map[string]filter.PropertyDefinition

	// IsChild indicates whether the entity uses prefixed names (parentId:name).
	IsChild bool

	// RelatedEntityPrefix is the dotted prefix for related entity property paths (e.g. "artifacts.").
	RelatedEntityPrefix string

	// RelatedEntityType is the type of the related entity referenced by RelatedEntityPrefix.
	RelatedEntityType filter.RelatedEntityType

	// RelatedEntityJoinTable is the table used to join the related entity. Defaults to "Attribution".
	RelatedEntityJoinTable string
}

// CatalogEntityRegistry is a registry-based implementation of filter.EntityMappingFunctions.
// Entity types are registered declaratively; no hand-coded switch/if chains are needed.
//
// Note: the production path uses per-plugin structs in each service package
// (e.g. modelcatalog/service/entity_mappings.go) since each GenericRepository
// only ever queries its own entity type. This registry remains available for
// cases where a single EntityMappingFunctions instance must handle multiple
// entity types.
type CatalogEntityRegistry struct {
	mu          sync.RWMutex
	definitions map[filter.RestEntityType]EntityTypeDefinition
}

// NewCatalogEntityRegistry creates a new empty registry.
func NewCatalogEntityRegistry() *CatalogEntityRegistry {
	return &CatalogEntityRegistry{
		definitions: make(map[filter.RestEntityType]EntityTypeDefinition),
	}
}

// Register adds (or replaces) an entity type definition in the registry.
func (r *CatalogEntityRegistry) Register(entityType filter.RestEntityType, def EntityTypeDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions[entityType] = def
}

// GetMLMDEntityType returns the underlying MLMD entity type for a REST entity type.
// Unknown types default to Context.
func (r *CatalogEntityRegistry) GetMLMDEntityType(restEntityType filter.RestEntityType) filter.EntityType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.definitions[restEntityType]; ok {
		return def.MLMDEntityType
	}
	return filter.EntityTypeContext
}

// GetPropertyDefinitionForRestEntity returns the property definition for a REST entity type.
//
// Resolution order:
//  1. Well-known property from the registered Properties map.
//  2. Related-entity prefix match (e.g. "artifacts.<prop>").
//  3. Custom property fallback.
func (r *CatalogEntityRegistry) GetPropertyDefinitionForRestEntity(restEntityType filter.RestEntityType, propertyName string) filter.PropertyDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.definitions[restEntityType]
	if !ok {
		return customFallback(propertyName)
	}

	// 1. Well-known property
	if propDef, found := def.Properties[propertyName]; found {
		return propDef
	}

	// 2. Related entity prefix
	if def.RelatedEntityPrefix != "" {
		if relatedPath, found := strings.CutPrefix(propertyName, def.RelatedEntityPrefix); found && relatedPath != "" {
			joinTable := def.RelatedEntityJoinTable
			if joinTable == "" {
				joinTable = "Attribution"
			}
			return filter.PropertyDefinition{
				Location:          filter.RelatedEntity,
				ValueType:         "", // empty to enable runtime type inference
				Column:            relatedPath,
				RelatedEntityType: def.RelatedEntityType,
				RelatedProperty:   relatedPath,
				JoinTable:         joinTable,
			}
		}
	}

	// 3. Custom property fallback
	return customFallback(propertyName)
}

// IsChildEntity returns whether the entity type uses prefixed names.
// Unknown types default to false.
func (r *CatalogEntityRegistry) IsChildEntity(entityType filter.RestEntityType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.definitions[entityType]; ok {
		return def.IsChild
	}
	return false
}

// customFallback returns a custom property definition for an unknown property.
func customFallback(propertyName string) filter.PropertyDefinition {
	return filter.PropertyDefinition{
		Location:  filter.Custom,
		ValueType: filter.StringValueType,
		Column:    propertyName,
	}
}
