package mcpcatalog

import (
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
)

type Services struct {
	MCPServerRepository       models.MCPServerRepository
	MCPServerToolRepository   models.MCPServerToolRepository
	CatalogSourceRepository   sharedmodels.CatalogSourceRepository
	PropertyOptionsRepository sharedmodels.PropertyOptionsRepository
}
