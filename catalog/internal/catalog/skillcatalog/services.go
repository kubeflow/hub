package skillcatalog

import (
	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
)

type Services struct {
	SkillRepository           skillmodels.SkillRepository
	CatalogSourceRepository   sharedmodels.CatalogSourceRepository
	PropertyOptionsRepository sharedmodels.PropertyOptionsRepository
}
