package service

import (
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staleStrProp(name, value string) dbmodels.Properties {
	v := value
	return dbmodels.Properties{Name: name, StringValue: &v}
}

func staleCustomStrProp(name, value string) dbmodels.Properties {
	v := value
	return dbmodels.Properties{Name: name, StringValue: &v, IsCustomProperty: true}
}

func definedPropNameSet(entity models.Skill) map[string]bool {
	names := map[string]bool{}
	if props := entity.GetProperties(); props != nil {
		for _, p := range *props {
			names[p.Name] = true
		}
	}
	return names
}

func customPropNameSet(entity models.Skill) map[string]bool {
	names := map[string]bool{}
	if props := entity.GetCustomProperties(); props != nil {
		for _, p := range *props {
			names[p.Name] = true
		}
	}
	return names
}

// Re-saving a skill is a full rebuild from the source (the loader upserts the
// re-resolved entity under the same composite name). A defined property the
// source dropped - a removed category, label or license - must not linger on
// the stored entity; GenericRepository.handleProperties only prunes stale
// custom properties. A custom property the source still sends must survive
// the re-sync unchanged.
func TestSkillRepositorySavePrunesStaleDefinedProperties(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	entityName := "src-a|repo/deploy|deploy|v1"

	v1 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "deploy-helper"),
			staleStrProp("category", "devops"),
			staleStrProp("labels", `["kubectl","deploy"]`),
			staleStrProp("license", "Apache-2.0"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("team", "platform"),
		},
	}
	saved, err := repo.Save(v1)
	require.NoError(t, err)
	require.NotNil(t, saved.GetID())

	// Re-sync: the source dropped category, labels and license and added an
	// author. The custom "team" metadata is still sent unchanged. Same
	// composite name -> upsert.
	v2 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "deploy-helper"),
			staleStrProp("author", "acme"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("team", "platform"),
		},
	}
	_, err = repo.Save(v2)
	require.NoError(t, err)

	reloaded, err := repo.GetByName(entityName)
	require.NoError(t, err)

	defined := definedPropNameSet(reloaded)
	assert.True(t, defined["name"], "kept defined property")
	assert.True(t, defined["source_id"], "kept defined property")
	assert.True(t, defined["author"], "newly added defined property")
	assert.False(t, defined["category"], "stale defined property must be pruned")
	assert.False(t, defined["labels"], "stale defined property must be pruned")
	assert.False(t, defined["license"], "stale defined property must be pruned")

	custom := customPropNameSet(reloaded)
	assert.True(t, custom["team"], "unchanged custom property must be preserved")
}

// Pre-existing behavior, covered here to pin it down while touching this
// code: a custom property the source stops sending is pruned by
// GenericRepository.handleProperties, which already handled stale custom
// properties before this fix.
func TestSkillRepositorySavePrunesStaleCustomProperties(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	entityName := "src-a|repo/rollback|rollback|v1"

	v1 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "rollback-helper"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("team", "platform"),
		},
	}
	_, err := repo.Save(v1)
	require.NoError(t, err)

	v2 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "rollback-helper"),
		},
		CustomProperties: &[]dbmodels.Properties{},
	}
	_, err = repo.Save(v2)
	require.NoError(t, err)

	reloaded, err := repo.GetByName(entityName)
	require.NoError(t, err)
	assert.False(t, customPropNameSet(reloaded)["team"], "dropped custom property must be pruned")
	assert.True(t, definedPropNameSet(reloaded)["name"], "defined property must survive")
}

// The prune and the upsert run in one transaction: if the upsert fails, the
// previously stored entity must survive untouched. The failure is injected
// with a defined property whose name exceeds the ContextProperty.name
// VARCHAR(255) column, which fails the property insert inside the save
// transaction.
func TestSkillRepositorySaveFailurePreservesStoredEntity(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	entityName := "src-a|repo/lint|lint|v1"

	v1 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "lint-check"),
			staleStrProp("category", "ci"),
		},
	}
	_, err := repo.Save(v1)
	require.NoError(t, err)

	tooLongName := strings.Repeat("x", 300)
	v2 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "lint-check"),
			staleStrProp(tooLongName, "boom"),
		},
	}
	_, err = repo.Save(v2)
	require.Error(t, err, "save with an over-long property name must fail")

	reloaded, err := repo.GetByName(entityName)
	require.NoError(t, err)
	defined := definedPropNameSet(reloaded)
	assert.True(t, defined["category"], "failed save must not lose the previously stored category")
	assert.True(t, defined["source_id"], "failed save must not lose the previously stored source_id")
	assert.False(t, defined[tooLongName], "failed save must not store the invalid property")
}

// A defined property and a custom property can share a name (a SKILL.md
// metadata key colliding with a defined field - see #3366). Pruning stale
// defined properties must not touch the custom row with the same name.
func TestSkillRepositorySavePruneLeavesSameNameCustomProperty(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	entityName := "src-a|repo/collide|collide|v1"

	v1 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "collide-check"),
			staleStrProp("category", "devops"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("category", "from-metadata"),
		},
	}
	_, err := repo.Save(v1)
	require.NoError(t, err)

	v2 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "collide-check"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("category", "from-metadata"),
		},
	}
	_, err = repo.Save(v2)
	require.NoError(t, err)

	reloaded, err := repo.GetByName(entityName)
	require.NoError(t, err)
	assert.False(t, definedPropNameSet(reloaded)["category"], "dropped defined property must be pruned")
	assert.True(t, customPropNameSet(reloaded)["category"], "same-name custom property must survive")
}

// A re-sync whose entity carries an empty defined-property set prunes every
// defined row from the stored version - the source dropped them all. A nil
// property set carries no property intent and leaves the stored defined
// properties untouched, mirroring nil custom properties.
func TestSkillRepositorySaveEmptyPropertiesPruneAllDefined(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	entityName := "src-a|repo/deploy|deploy|v2"

	v1 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
			staleStrProp("name", "deploy-helper"),
			staleStrProp("category", "devops"),
		},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("team", "platform"),
		},
	}
	_, err := repo.Save(v1)
	require.NoError(t, err)

	v2 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{},
		CustomProperties: &[]dbmodels.Properties{
			staleCustomStrProp("team", "platform"),
		},
	}
	_, err = repo.Save(v2)
	require.NoError(t, err)

	reloaded, err := repo.GetByName(entityName)
	require.NoError(t, err)
	assert.Empty(t, definedPropNameSet(reloaded), "empty incoming set must prune every defined property")
	assert.True(t, customPropNameSet(reloaded)["team"], "custom property must survive")

	// A nil property set carries no intent: stored defined properties stay.
	v3 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
		Properties: &[]dbmodels.Properties{
			staleStrProp("source_id", "src-a"),
		},
	}
	_, err = repo.Save(v3)
	require.NoError(t, err)
	v4 := &models.SkillImpl{
		Attributes: &models.SkillAttributes{Name: &entityName},
	}
	_, err = repo.Save(v4)
	require.NoError(t, err)
	reloaded, err = repo.GetByName(entityName)
	require.NoError(t, err)
	assert.True(t, definedPropNameSet(reloaded)["source_id"], "nil properties must leave stored defined properties untouched")
}
