package service

import (
	mcpcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/service"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

const (
	CatalogModelTypeName           = "kf.CatalogModel"
	CatalogModelArtifactTypeName   = "kf.CatalogModelArtifact"
	CatalogMetricsArtifactTypeName = "kf.CatalogMetricsArtifact"
	CatalogSourceTypeName          = "kf.CatalogSource"
	MCPServerTypeName              = "kf.MCPServer"
	MCPServerToolTypeName          = "kf.MCPServerTool"
)

func DatastoreSpec() *datastore.Spec {
	return datastore.NewSpec().
		AddContext(CatalogModelTypeName, datastore.NewSpecType(modelcatalogservice.NewCatalogModelRepository).
			AddString("source_id").
			AddString("description").
			AddString("owner").
			AddString("state").
			AddStruct("language").
			AddString("library_name").
			AddString("license_link").
			AddString("license").
			AddString("logo").
			AddString("maturity").
			AddString("provider").
			AddString("readme").
			AddStruct("tasks"),
		).
		AddContext(CatalogSourceTypeName, datastore.NewSpecType(NewCatalogSourceRepository).
			AddString("status").
			AddString("error"),
		).
		AddContext(MCPServerTypeName, datastore.NewSpecType(mcpcatalogservice.NewMCPServerRepository).
			AddString("source_id").
			AddString("base_name").
			AddString("description").
			AddString("provider").
			AddString("license").
			AddString("license_link").
			AddString("logo").
			AddString("readme").
			AddString("version").
			AddStruct("tags").
			AddStruct("transports").
			AddString("deploymentMode").
			AddBoolean("verifiedSource").
			AddBoolean("secureEndpoint").
			AddBoolean("sast").
			AddBoolean("readOnlyTools"),
		).
		AddExecution(MCPServerToolTypeName, datastore.NewSpecType(mcpcatalogservice.NewMCPServerToolRepository).
			AddString("accessType").
			AddString("description").
			AddString("externalId").
			AddString("parameters"),
		).
		AddArtifact(CatalogModelArtifactTypeName, datastore.NewSpecType(modelcatalogservice.NewCatalogModelArtifactRepository).
			AddString("uri"),
		).
		AddArtifact(CatalogMetricsArtifactTypeName, datastore.NewSpecType(modelcatalogservice.NewCatalogMetricsArtifactRepository).
			AddString("metricsType"),
		).
		AddOther(NewCatalogArtifactRepository).
		AddOther(NewPropertyOptionsRepository)
}

