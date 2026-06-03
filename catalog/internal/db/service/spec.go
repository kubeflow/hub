package service

import (
	"github.com/kubeflow/hub/catalog/internal/plugin"
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
	spec := datastore.NewSpec().
		AddContext(CatalogSourceTypeName, datastore.NewSpecType(NewCatalogSourceRepository).
			AddString("status").
			AddString("error"),
		).
		AddOther(NewCatalogArtifactRepository).
		AddOther(NewPropertyOptionsRepository)

	applyEntries(spec, plugin.All(), plugin.ExtraDatastoreEntries())
	return spec
}

func applyEntries(spec *datastore.Spec, plugins []plugin.CatalogPlugin, extra []plugin.DatastoreEntry) {
	for _, p := range plugins {
		if dsp, ok := p.(plugin.DatastoreSpecProvider); ok {
			for _, e := range dsp.DatastoreEntries() {
				addEntry(spec, e)
			}
		}
	}
	for _, e := range extra {
		addEntry(spec, e)
	}
}

func addEntry(spec *datastore.Spec, e plugin.DatastoreEntry) {
	switch e.Category {
	case "context":
		spec.AddContext(e.TypeName, e.Spec)
	case "artifact":
		spec.AddArtifact(e.TypeName, e.Spec)
	case "execution":
		spec.AddExecution(e.TypeName, e.Spec)
	}
}
