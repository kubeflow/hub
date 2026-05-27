package modelcatalog

import (
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
)

type Services struct {
	CatalogModelRepository           models.CatalogModelRepository
	CatalogArtifactRepository        sharedmodels.CatalogArtifactRepository
	CatalogModelArtifactRepository   models.CatalogModelArtifactRepository
	CatalogMetricsArtifactRepository models.CatalogMetricsArtifactRepository
	CatalogSourceRepository          sharedmodels.CatalogSourceRepository
	PropertyOptionsRepository        sharedmodels.PropertyOptionsRepository
}
