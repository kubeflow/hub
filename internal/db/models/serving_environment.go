package models

import "github.com/kubeflow/hub/internal/db/filter"

type ServingEnvironmentListOptions struct {
	Pagination
	Name       *string
	ExternalID *string
}

// GetRestEntityType implements the FilterApplier interface
func (s *ServingEnvironmentListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityServingEnvironment
}

type ServingEnvironmentAttributes struct {
	Name                     *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

type ServingEnvironment interface {
	Entity[ServingEnvironmentAttributes]
}

type ServingEnvironmentImpl = BaseEntity[ServingEnvironmentAttributes]

type ServingEnvironmentRepository interface {
	GetByID(id int32) (ServingEnvironment, error)
	List(listOptions ServingEnvironmentListOptions) (*ListWrapper[ServingEnvironment], error)
	Save(model ServingEnvironment) (ServingEnvironment, error)
}
