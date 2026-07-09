package agentcatalog

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlTemplateToEntity_DefaultNameWhenNil(t *testing.T) {
	tmpl := yamlAgentTemplate{
		Name:    nil,
		Content: "template body",
	}

	entity := yamlTemplateToEntity(tmpl, "my-agent", "my-source")

	attrs := entity.GetAttributes()
	require.NotNil(t, attrs)
	require.NotNil(t, attrs.Name)
	assert.Equal(t, "my-source:my-agent:agent.yaml", *attrs.Name)
}

func TestYamlTemplateToEntity_DefaultNameWhenEmptyString(t *testing.T) {
	emptyName := ""
	tmpl := yamlAgentTemplate{
		Name:    &emptyName,
		Content: "template body",
	}

	entity := yamlTemplateToEntity(tmpl, "my-agent", "my-source")

	attrs := entity.GetAttributes()
	require.NotNil(t, attrs)
	require.NotNil(t, attrs.Name)
	assert.Equal(t, "my-source:my-agent:agent.yaml", *attrs.Name)
}

func TestYamlTemplateToEntity_CustomNameUsed(t *testing.T) {
	customName := "deployment.yaml"
	tmpl := yamlAgentTemplate{
		Name:    &customName,
		Content: "template body",
	}

	entity := yamlTemplateToEntity(tmpl, "my-agent", "my-source")

	attrs := entity.GetAttributes()
	require.NotNil(t, attrs)
	require.NotNil(t, attrs.Name)
	assert.Equal(t, "my-source:my-agent:deployment.yaml", *attrs.Name)
}

func TestYamlTemplateToEntity_QualifiedNameOrdering(t *testing.T) {
	customName := "config.yaml"
	tmpl := yamlAgentTemplate{Name: &customName, Content: "x"}

	entity := yamlTemplateToEntity(tmpl, "agent-b", "source-a")

	attrs := entity.GetAttributes()
	require.NotNil(t, attrs.Name)
	assert.Equal(t, "source-a:agent-b:config.yaml", *attrs.Name)
}

func TestYamlTemplateToEntity_ContentAndArtifactTypeSet(t *testing.T) {
	tmpl := yamlAgentTemplate{Content: "some yaml content"}

	entity := yamlTemplateToEntity(tmpl, "agent", "source")

	attrs := entity.GetAttributes()
	require.NotNil(t, attrs.Content)
	assert.Equal(t, "some yaml content", *attrs.Content)
	require.NotNil(t, attrs.ArtifactType)
	assert.Equal(t, models.AgentTemplateArtifactType, *attrs.ArtifactType)

	props := entity.GetProperties()
	require.NotNil(t, props)
	require.Len(t, *props, 1)
	assert.Equal(t, "content", (*props)[0].Name)
	require.NotNil(t, (*props)[0].StringValue)
	assert.Equal(t, "some yaml content", *(*props)[0].StringValue)
	assert.False(t, (*props)[0].IsCustomProperty)
}

func TestYamlTemplateToEntity_IDNotSet(t *testing.T) {
	tmpl := yamlAgentTemplate{Content: "x"}

	entity := yamlTemplateToEntity(tmpl, "agent", "source")

	assert.Nil(t, entity.GetID())
}
