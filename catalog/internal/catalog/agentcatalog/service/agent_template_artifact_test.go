package service

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTemplateArtifactRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	agentTypeID := getAgentTypeID(t, sharedDB)
	templateTypeID := getAgentTemplateArtifactTypeID(t, sharedDB)

	agentRepo := NewAgentRepository(sharedDB, agentTypeID)
	templateRepo := NewAgentTemplateArtifactRepository(sharedDB, templateTypeID)

	saveAgent := func(name string) int32 {
		saved, err := agentRepo.Save(&models.AgentImpl{Attributes: &models.AgentAttributes{Name: &name}})
		require.NoError(t, err)
		return *saved.GetID()
	}

	saveTemplate := func(name, content string, parentID int32) models.AgentTemplateArtifact {
		saved, err := templateRepo.Save(&models.AgentTemplateArtifactImpl{
			Attributes: &models.AgentTemplateArtifactAttributes{
				Name:    &name,
				Content: &content,
			},
		}, &parentID)
		require.NoError(t, err)
		return saved
	}

	t.Run("TestDeleteByParentID_RemovesOnlyOwnParentAndType", func(t *testing.T) {
		parent1ID := saveAgent("source:agent-one")
		parent2ID := saveAgent("source:agent-two")

		saveTemplate("source:agent-one:agent.yaml", "content-1", parent1ID)
		tmpl2 := saveTemplate("source:agent-one:extra.yaml", "content-2", parent1ID)
		parent2Template := saveTemplate("source:agent-two:agent.yaml", "content-3", parent2ID)

		// Attach an artifact of a different type to parent1, to confirm
		// DeleteByParentID scopes its deletion to its own artifact type.
		otherTypeID := getTypeIDByName(t, sharedDB, "kf.OtherArtifact")
		otherArtifact := schema.Artifact{TypeID: otherTypeID, Name: new("unrelated-artifact")}
		require.NoError(t, sharedDB.Create(&otherArtifact).Error)
		require.NoError(t, sharedDB.Create(&schema.Attribution{ContextID: parent1ID, ArtifactID: otherArtifact.ID}).Error)

		require.NoError(t, templateRepo.DeleteByParentID(parent1ID))

		list1, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent1ID})
		require.NoError(t, err)
		assert.Empty(t, list1.Items, "parent1's template artifacts should be deleted")

		_, err = templateRepo.GetByID(*tmpl2.GetID())
		assert.ErrorIs(t, err, ErrAgentTemplateArtifactNotFound)

		list2, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent2ID})
		require.NoError(t, err)
		require.Len(t, list2.Items, 1, "parent2's template artifact should be untouched")
		require.NotNil(t, list2.Items[0].GetAttributes().Name)
		assert.Equal(t, *parent2Template.GetAttributes().Name, *list2.Items[0].GetAttributes().Name)

		var remaining schema.Artifact
		err = sharedDB.Where("id = ?", otherArtifact.ID).First(&remaining).Error
		require.NoError(t, err, "an artifact of a different type attached to the same parent should not be deleted")
	})

	t.Run("TestDeleteByParentID_NoArtifacts", func(t *testing.T) {
		agentID := saveAgent("source:agent-empty")

		// A parent with no template artifacts should be idempotent, not an error.
		require.NoError(t, templateRepo.DeleteByParentID(agentID))
	})
}
