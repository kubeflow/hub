package models

import "github.com/kubeflow/hub/internal/db/filter"

type InferenceServiceListOptions struct {
	Pagination
	Name             *string
	ExternalID       *string
	ParentResourceID *int32
	Runtime          *string
}

// GetRestEntityType implements the FilterApplier interface.
// Required by ApplyFilterQuery in generic_repository.go to apply filterQuery expressions.
func (i *InferenceServiceListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityInferenceService
}

type InferenceServiceAttributes struct {
	Name                     *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

type InferenceService interface {
	Entity[InferenceServiceAttributes]
}

type InferenceServiceImpl = BaseEntity[InferenceServiceAttributes]

type InferenceServiceRepository interface {
	GetByID(id int32) (InferenceService, error)
	List(listOptions InferenceServiceListOptions) (*ListWrapper[InferenceService], error)
	Save(model InferenceService) (InferenceService, error)
}
