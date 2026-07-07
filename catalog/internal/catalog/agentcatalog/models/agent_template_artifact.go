package models

import (
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/filter"
)

const AgentTemplateArtifactType = "template-artifact"

type AgentTemplateArtifactListOptions struct {
	dbmodels.Pagination
	Name             *string
	ParentResourceID *int32
}

func (o *AgentTemplateArtifactListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityType("agent-template-artifact")
}

func (o *AgentTemplateArtifactListOptions) GetFilterQuery() string {
	return ""
}

type AgentTemplateArtifactAttributes struct {
	Name                     *string
	Content                  *string
	ArtifactType             *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

type AgentTemplateArtifact interface {
	dbmodels.Entity[AgentTemplateArtifactAttributes]
}

type AgentTemplateArtifactImpl = dbmodels.BaseEntity[AgentTemplateArtifactAttributes]

type AgentTemplateArtifactRepository interface {
	GetByID(id int32) (AgentTemplateArtifact, error)
	List(listOptions AgentTemplateArtifactListOptions) (*dbmodels.ListWrapper[AgentTemplateArtifact], error)
	Save(artifact AgentTemplateArtifact, parentResourceID *int32) (AgentTemplateArtifact, error)
	DeleteByParentID(parentID int32) error
}
