// Registers datastore entries for tests that call service.DatastoreSpec().
// Import this package from any test package that needs a complete spec.
// Keep entries in sync with each plugin's DatastoreEntries() implementation.
package testhelpers

import (
	agentcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/service"
	mcpcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/service"
	modelcatalogservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/plugin"
	"github.com/kubeflow/hub/internal/platform/datastore"
)

func init() {
	plugin.RegisterDatastoreEntries(
		plugin.DatastoreEntry{
			TypeName: service.CatalogModelTypeName,
			Category: "context",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogModelRepository).
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
		},
		plugin.DatastoreEntry{
			TypeName: service.CatalogModelArtifactTypeName,
			Category: "artifact",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogModelArtifactRepository).
				AddString("uri"),
		},
		plugin.DatastoreEntry{
			TypeName: service.CatalogMetricsArtifactTypeName,
			Category: "artifact",
			Spec: datastore.NewSpecType(modelcatalogservice.NewCatalogMetricsArtifactRepository).
				AddString("metricsType"),
		},
		plugin.DatastoreEntry{
			TypeName: "kf.Agent",
			Category: "context",
			Spec: datastore.NewSpecType(agentcatalogservice.NewAgentRepository).
				AddString("source_id").
				AddString("displayName").
				AddString("description").
				AddString("readme").
				AddString("framework").
				AddStruct("labels").
				AddString("logo").
				AddString("repositoryUrl").
				AddStruct("env").
				AddStruct("artifacts"),
		},
		plugin.DatastoreEntry{
			TypeName: service.MCPServerTypeName,
			Category: "context",
			Spec: datastore.NewSpecType(mcpcatalogservice.NewMCPServerRepository).
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
		},
		plugin.DatastoreEntry{
			TypeName: service.MCPServerToolTypeName,
			Category: "execution",
			Spec: datastore.NewSpecType(mcpcatalogservice.NewMCPServerToolRepository).
				AddString("accessType").
				AddString("description").
				AddString("externalId").
				AddString("parameters"),
		},
	)
}
