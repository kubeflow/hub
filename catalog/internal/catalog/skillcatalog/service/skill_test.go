package service

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	typeID := getSkillTypeID(t, sharedDB)
	repo := NewSkillRepository(sharedDB, typeID)

	seed := []struct {
		entityName  string
		name        string
		description string
		readme      string
		sourceID    string
	}{
		{"src-a|repo/deploy|deploy|latest", "deploy-helper", "Deploys a service to the cluster", "# Deploy Helper\nUses kubectl under the hood.", "src-a"},
		{"src-a|repo/rollback|rollback|latest", "rollback-helper", "Rolls back a deployment", "# Rollback Helper\nReverts to the previous revision.", "src-a"},
		{"src-b|repo/lint|lint|latest", "lint-check", "Runs the project's linter", "# Lint Check\nWraps golangci-lint.", "src-b"},
	}

	for _, s := range seed {
		_, err := repo.Save(&models.SkillImpl{
			Attributes: &models.SkillAttributes{
				Name: &s.entityName,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: &s.sourceID},
				{Name: "name", StringValue: &s.name},
				{Name: "description", StringValue: &s.description},
				{Name: "readme", StringValue: &s.readme},
			},
		})
		require.NoError(t, err)
	}

	t.Run("FilterByName", func(t *testing.T) {
		// Name is a raw SQL LIKE pattern: the caller supplies wildcards.
		nameFilter := "%helper"
		result, err := repo.List(&models.SkillListOptions{Name: &nameFilter})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)

		exactFilter := "lint-check"
		result, err = repo.List(&models.SkillListOptions{Name: &exactFilter})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)

		noMatch := "%nonexistent%"
		result, err = repo.List(&models.SkillListOptions{Name: &noMatch})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("FilterByQuery", func(t *testing.T) {
		// Matches via the description property.
		q := "linter"
		result, err := repo.List(&models.SkillListOptions{Query: &q})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)

		// Matches via the name property.
		q = "rollback"
		result, err = repo.List(&models.SkillListOptions{Query: &q})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)

		// Matches via the readme property, case-insensitively.
		q = "GOLANGCI-LINT"
		result, err = repo.List(&models.SkillListOptions{Query: &q})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
	})

	t.Run("FilterBySourceIDs", func(t *testing.T) {
		result, err := repo.List(&models.SkillListOptions{SourceIDs: &[]string{"src-b"}})
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)

		result, err = repo.List(&models.SkillListOptions{SourceIDs: &[]string{"src-a", "src-b"}})
		require.NoError(t, err)
		assert.Len(t, result.Items, 3)

		result, err = repo.List(&models.SkillListOptions{SourceIDs: &[]string{"nonexistent"}})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("FilterBySourceIDs_IgnoresCollidingCustomProperty", func(t *testing.T) {
		// A SKILL.md frontmatter metadata key literally named "source_id" becomes a
		// customProperty (customPropertiesFromMetadata has no reserved-name guard).
		// It must not duplicate the skill in a source_id-filtered result.
		collidingName := "src-a|repo/collide|collide|latest"
		realSourceID := "src-a"
		_, err := repo.Save(&models.SkillImpl{
			Attributes: &models.SkillAttributes{
				Name: &collidingName,
			},
			Properties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: &realSourceID},
			},
			CustomProperties: &[]dbmodels.Properties{
				{Name: "source_id", StringValue: &realSourceID, IsCustomProperty: true},
			},
		})
		require.NoError(t, err)

		result, err := repo.List(&models.SkillListOptions{SourceIDs: &[]string{realSourceID}})
		require.NoError(t, err)
		ids := map[int32]int{}
		for _, item := range result.Items {
			ids[*item.GetID()]++
		}
		for id, count := range ids {
			assert.Equal(t, 1, count, "skill %d appeared %d times in a source_id-filtered list", id, count)
		}
	})
}
