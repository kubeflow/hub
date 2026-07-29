package models

import (
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/filter"
)

// SkillListOptions holds the options for listing Skill entities.
type SkillListOptions struct {
	dbmodels.Pagination
	SourceIDs   *[]string
	FilterQuery *string
}

// GetRestEntityType implements the FilterApplier interface.
func (o *SkillListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityType("skill")
}

// GetFilterQuery returns the filter query string for advanced filtering.
func (o *SkillListOptions) GetFilterQuery() string {
	if o.FilterQuery == nil {
		return ""
	}
	return *o.FilterQuery
}

// SkillAttributes holds the attributes for a Skill record.
type SkillAttributes struct {
	Name                     *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

// Skill represents a Skill stored in the database.
type Skill interface {
	dbmodels.Entity[SkillAttributes]
}

// SkillImpl is the concrete implementation of Skill.
type SkillImpl = dbmodels.BaseEntity[SkillAttributes]

// SkillRepository defines the interface for Skill persistence.
type SkillRepository interface {
	GetByID(id int32) (Skill, error)
	GetByName(name string) (Skill, error)
	List(listOptions *SkillListOptions) (*dbmodels.ListWrapper[Skill], error)
	Save(entity Skill) (Skill, error)
	DeleteBySource(sourceID string) error
	DeleteByID(id int32) error
	GetDistinctSourceIDs() ([]string, error)
	GetTypeID() int32
}
