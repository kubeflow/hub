package service

import (
	"fmt"

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

func DatastoreSpec() (*datastore.Spec, error) {
	spec := datastore.NewSpec().
		AddContext(CatalogSourceTypeName, datastore.NewSpecType(NewCatalogSourceRepository).
			AddString("status").
			AddString("error"),
		).
		AddOther(NewCatalogArtifactRepository).
		AddOther(NewPropertyOptionsRepository)

	if err := applyEntries(spec, plugin.All(), plugin.ExtraDatastoreEntries()); err != nil {
		return nil, err
	}
	return spec, nil
}

func applyEntries(spec *datastore.Spec, plugins []plugin.CatalogPlugin, extra []plugin.DatastoreEntry) error {
	for _, p := range plugins {
		if dsp, ok := p.(plugin.DatastoreSpecProvider); ok {
			for _, e := range dsp.DatastoreEntries() {
				if err := addEntry(spec, e); err != nil {
					return err
				}
			}
		}
	}
	for _, e := range extra {
		if err := addEntry(spec, e); err != nil {
			return err
		}
	}
	return nil
}

func addEntry(spec *datastore.Spec, e plugin.DatastoreEntry) error {
	switch e.Category {
	case "context":
		spec.AddContext(e.TypeName, e.Spec)
	case "artifact":
		spec.AddArtifact(e.TypeName, e.Spec)
	case "execution":
		spec.AddExecution(e.TypeName, e.Spec)
	default:
		return fmt.Errorf("unknown datastore entry category %q for type %q", e.Category, e.TypeName)
	}
	return nil
}
