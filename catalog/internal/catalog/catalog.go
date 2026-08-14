package catalog

import (
	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog"
	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
)

type (
	ModelSource = basecatalog.ModelSource
	FieldFilter = basecatalog.FieldFilter
)

type (
	SourceCollection               = modelcatalog.SourceCollection
	LabelCollection                = modelcatalog.LabelCollection
	ModelProviderRecord            = modelcatalog.ModelProviderRecord
	APIProvider                    = modelcatalog.APIProvider
	ListModelsParams               = modelcatalog.ListModelsParams
	ListArtifactsParams            = modelcatalog.ListArtifactsParams
	ListPerformanceArtifactsParams = modelcatalog.ListPerformanceArtifactsParams

	// Agent catalog types
	AgentSourceCollection = agentcatalog.AgentSourceCollection

	// Skill catalog types
	SkillSourceCollection = skillcatalog.SkillSourceCollection

	// MCP catalog types
	MCPSourceCollection      = mcpcatalog.MCPSourceCollection
	MCPProvider              = mcpcatalog.MCPCatalogProvider
	ListMCPServersParams     = mcpcatalog.ListMCPServersParams
	ListMCPServerToolsParams = mcpcatalog.ListMCPServerToolsParams
)

var (
	NewSourceCollection         = modelcatalog.NewSourceCollection
	NewLabelCollection          = modelcatalog.NewLabelCollection
	NewDBCatalog                = modelcatalog.NewDBCatalog
	NewPerformanceMetricsLoader = modelcatalog.NewPerformanceMetricsLoader

	// MCP catalog constructors
	NewDBMCPCatalog = mcpcatalog.NewDBMCPCatalog
)

var (
	ParsePreviewConfig  = modelcatalog.ParsePreviewConfig
	PreviewSourceModels = modelcatalog.PreviewSourceModels
)
