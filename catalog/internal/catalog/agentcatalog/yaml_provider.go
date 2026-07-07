package agentcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type yamlAgentEnvVar struct {
	Name     string `yaml:"name" json:"name"`
	Required bool   `yaml:"required" json:"required"`
}

type yamlAgentArtifact struct {
	URI string `yaml:"uri" json:"uri"`
}

type yamlAgentTemplate struct {
	Name    *string `yaml:"name,omitempty" json:"name,omitempty"`
	Content string  `yaml:"content" json:"content"`
}

type yamlAgent struct {
	Name             string                              `yaml:"name" json:"name"`
	ExternalID       *string                             `yaml:"externalId,omitempty" json:"externalId,omitempty"`
	DisplayName      *string                             `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Description      *string                             `yaml:"description,omitempty" json:"description,omitempty"`
	Readme           *string                             `yaml:"readme,omitempty" json:"readme,omitempty"`
	Framework        *string                             `yaml:"framework,omitempty" json:"framework,omitempty"`
	Labels           []string                            `yaml:"labels,omitempty" json:"labels,omitempty"`
	Logo             *string                             `yaml:"logo,omitempty" json:"logo,omitempty"`
	RepositoryUrl    *string                             `yaml:"repositoryUrl,omitempty" json:"repositoryUrl,omitempty"`
	Env              []yamlAgentEnvVar                   `yaml:"env,omitempty" json:"env,omitempty"`
	Artifacts        []yamlAgentArtifact                 `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Templates        []yamlAgentTemplate                 `yaml:"templates,omitempty" json:"templates,omitempty"`
	CustomProperties *map[string]openapi.MetadataValue   `yaml:"customProperties,omitempty" json:"customProperties,omitempty"`
	CreateTimeSinceEpoch     *string                     `yaml:"createTimeSinceEpoch,omitempty" json:"createTimeSinceEpoch,omitempty"`
	LastUpdateTimeSinceEpoch *string                     `yaml:"lastUpdateTimeSinceEpoch,omitempty" json:"lastUpdateTimeSinceEpoch,omitempty"`
}

type yamlAgentCatalog struct {
	Agents []yamlAgent `yaml:"agents" json:"agents"`
}

func readYAMLAgentCatalog(path string) (*yamlAgentCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file %s: %w", path, err)
	}

	var catalog yamlAgentCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("error parsing YAML from %s: %w", path, err)
	}

	return &catalog, nil
}

func yamlAgentToEntity(ya yamlAgent, sourceID string) models.Agent {
	namespacedName := sourceID + ":" + ya.Name
	attrs := &models.AgentAttributes{
		Name:       &namespacedName,
		ExternalID: ya.ExternalID,
	}

	if ya.CreateTimeSinceEpoch != nil {
		if v, err := strconv.ParseInt(*ya.CreateTimeSinceEpoch, 10, 64); err == nil {
			attrs.CreateTimeSinceEpoch = &v
		}
	}
	if ya.LastUpdateTimeSinceEpoch != nil {
		if v, err := strconv.ParseInt(*ya.LastUpdateTimeSinceEpoch, 10, 64); err == nil {
			attrs.LastUpdateTimeSinceEpoch = &v
		}
	}

	agent := &models.AgentImpl{
		Attributes: attrs,
	}

	properties := []dbmodels.Properties{}

	properties = append(properties, dbmodels.NewStringProperty("source_id", sourceID, false))

	if ya.DisplayName != nil {
		properties = append(properties, dbmodels.NewStringProperty("displayName", *ya.DisplayName, false))
	}
	if ya.Description != nil {
		properties = append(properties, dbmodels.NewStringProperty("description", *ya.Description, false))
	}
	if ya.Readme != nil {
		properties = append(properties, dbmodels.NewStringProperty("readme", *ya.Readme, false))
	}
	if ya.Framework != nil {
		properties = append(properties, dbmodels.NewStringProperty("framework", *ya.Framework, false))
	}
	if len(ya.Labels) > 0 {
		if jsonBytes, err := json.Marshal(ya.Labels); err == nil {
			properties = append(properties, dbmodels.NewStringProperty("labels", string(jsonBytes), false))
		}
	}
	if ya.Logo != nil {
		properties = append(properties, dbmodels.NewStringProperty("logo", *ya.Logo, false))
	}
	if ya.RepositoryUrl != nil {
		properties = append(properties, dbmodels.NewStringProperty("repositoryUrl", *ya.RepositoryUrl, false))
	}
	if len(ya.Env) > 0 {
		if jsonBytes, err := json.Marshal(ya.Env); err == nil {
			properties = append(properties, dbmodels.NewStringProperty("env", string(jsonBytes), false))
		}
	}
	if len(ya.Artifacts) > 0 {
		for i, artifact := range ya.Artifacts {
			if err := basecatalog.ValidateArtifactURI(artifact.URI); err != nil {
				glog.Warningf("agent %q artifact %d: %v", ya.Name, i, err)
			}
		}
		if jsonBytes, err := json.Marshal(ya.Artifacts); err == nil {
			properties = append(properties, dbmodels.NewStringProperty("artifacts", string(jsonBytes), false))
		}
	}

	agent.Properties = &properties

	if ya.CustomProperties != nil {
		customProps := []dbmodels.Properties{}
		for key, value := range *ya.CustomProperties {
			customProps = append(customProps, convertAgentMetadataToProperty(key, value))
		}
		agent.CustomProperties = &customProps
	}

	return agent
}

func convertAgentMetadataToProperty(key string, value openapi.MetadataValue) dbmodels.Properties {
	if value.MetadataStringValue != nil {
		return dbmodels.NewStringProperty(key, value.MetadataStringValue.StringValue, true)
	} else if value.MetadataBoolValue != nil {
		return dbmodels.NewBoolProperty(key, value.MetadataBoolValue.BoolValue, true)
	} else if value.MetadataIntValue != nil {
		return dbmodels.NewStringProperty(key, value.MetadataIntValue.IntValue, true)
	} else if value.MetadataDoubleValue != nil {
		return dbmodels.NewDoubleProperty(key, value.MetadataDoubleValue.DoubleValue, true)
	}
	if jsonBytes, err := json.Marshal(value); err == nil {
		return dbmodels.NewStringProperty(key, string(jsonBytes), true)
	}
	return dbmodels.NewStringProperty(key, "", true)
}

func yamlTemplateToEntity(tmpl yamlAgentTemplate, agentName string, sourceID string) models.AgentTemplateArtifact {
	name := "agent.yaml"
	if tmpl.Name != nil && *tmpl.Name != "" {
		name = *tmpl.Name
	}
	qualifiedName := sourceID + ":" + agentName + ":" + name

	attrs := &models.AgentTemplateArtifactAttributes{
		Name:         &qualifiedName,
		Content:      &tmpl.Content,
		ArtifactType: strPtr(models.AgentTemplateArtifactType),
	}

	entity := &models.AgentTemplateArtifactImpl{
		Attributes: attrs,
	}

	properties := []dbmodels.Properties{
		dbmodels.NewStringProperty("content", tmpl.Content, false),
	}
	entity.Properties = &properties

	return entity
}

func strPtr(s string) *string { return &s }

func resolveYAMLPath(source basecatalog.PluginSource) (string, error) {
	yamlPath, ok := source.Properties["yamlCatalogPath"].(string)
	if !ok {
		return "", fmt.Errorf("yamlCatalogPath property is required for YAML agent provider")
	}

	if filepath.IsAbs(yamlPath) {
		return yamlPath, nil
	}

	sourceDir := filepath.Dir(source.Origin)
	return filepath.Join(sourceDir, yamlPath), nil
}
