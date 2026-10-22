package repositories

import (
	"log/slog"
)

type ModelCatalogClientInterface interface {
	CatalogSourcesInterface
	CatalogModelsInterface
	CatalogSourcePreviewInterface
	CatalogSourceStatusInterface
	McpServerCatalogInterface
	AgentCatalogInterface
}

type ModelCatalogClient struct {
	logger *slog.Logger
	CatalogSources
	CatalogModels
	CatalogSourcePreview
	CatalogSourceStatus
	McpServerCatalog
	AgentCatalog
}

func NewModelCatalogClient(logger *slog.Logger) (ModelCatalogClientInterface, error) {
	return &ModelCatalogClient{logger: logger}, nil
}
