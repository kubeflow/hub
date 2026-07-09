package service

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTemplateArtifactRepository_DeleteByParentID(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	agentTypeID := getAgentTypeID(t, sharedDB)
	templateTypeID := getAgentTemplateArtifactTypeID(t, sharedDB)

	agentRepo := NewAgentRepository(sharedDB, agentTypeID)
	templateRepo := NewAgentTemplateArtifactRepository(sharedDB, templateTypeID)

	// Create two parent agents.
	parent1Name := "source:agent-one"
	parent1, err := agentRepo.Save(&models.AgentImpl{
		Attributes: &models.AgentAttributes{Name: &parent1Name},
	})
	require.NoError(t, err)
	parent1ID := *parent1.GetID()

	parent2Name := "source:agent-two"
	parent2, err := agentRepo.Save(&models.AgentImpl{
		Attributes: &models.AgentAttributes{Name: &parent2Name},
	})
	require.NoError(t, err)
	parent2ID := *parent2.GetID()

	saveTemplate := func(name, content string, parentID int32) {
		_, err := templateRepo.Save(&models.AgentTemplateArtifactImpl{
			Attributes: &models.AgentTemplateArtifactAttributes{
				Name:    &name,
				Content: &content,
			},
		}, &parentID)
		require.NoError(t, err)
	}

	// Two template artifacts under parent1, one under parent2.
	saveTemplate("source:agent-one:agent.yaml", "content-1", parent1ID)
	saveTemplate("source:agent-one:extra.yaml", "content-2", parent1ID)
	parent2TemplateName := "source:agent-two:agent.yaml"
	saveTemplate(parent2TemplateName, "content-3", parent2ID)

	// Attach an artifact of a different type to parent1, to confirm
	// DeleteByParentID scopes its deletion to its own artifact type.
	otherTypeID := getTypeIDByName(t, sharedDB, "kf.OtherArtifact")
	otherArtifact := schema.Artifact{TypeID: otherTypeID, Name: new("unrelated-artifact")}
	require.NoError(t, sharedDB.Create(&otherArtifact).Error)
	require.NoError(t, sharedDB.Create(&schema.Attribution{ContextID: parent1ID, ArtifactID: otherArtifact.ID}).Error)

	impl, ok := templateRepo.(*AgentTemplateArtifactRepositoryImpl)
	require.True(t, ok)

	require.NoError(t, impl.DeleteByParentID(parent1ID))

	list1, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent1ID})
	require.NoError(t, err)
	assert.Empty(t, list1.Items, "parent1's template artifacts should be deleted")

	list2, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent2ID})
	require.NoError(t, err)
	require.Len(t, list2.Items, 1, "parent2's template artifact should be untouched")
	require.NotNil(t, list2.Items[0].GetAttributes().Name)
	assert.Equal(t, parent2TemplateName, *list2.Items[0].GetAttributes().Name)

	var remaining schema.Artifact
	err = sharedDB.Where("id = ?", otherArtifact.ID).First(&remaining).Error
	require.NoError(t, err, "an artifact of a different type attached to the same parent should not be deleted")
}

func TestAgentTemplateArtifactRepository_DeleteByParentID_NoArtifacts(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	agentTypeID := getAgentTypeID(t, sharedDB)
	templateTypeID := getAgentTemplateArtifactTypeID(t, sharedDB)

	agentRepo := NewAgentRepository(sharedDB, agentTypeID)
	templateRepo := NewAgentTemplateArtifactRepository(sharedDB, templateTypeID)

	name := "source:agent-empty"
	agent, err := agentRepo.Save(&models.AgentImpl{Attributes: &models.AgentAttributes{Name: &name}})
	require.NoError(t, err)
	agentID := *agent.GetID()

	impl, ok := templateRepo.(*AgentTemplateArtifactRepositoryImpl)
	require.True(t, ok)

	// A parent with no template artifacts should be a no-op, not an error.
	require.NoError(t, impl.DeleteByParentID(agentID))
}
