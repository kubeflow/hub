package main

import (
	"testing"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"dataset", "Dataset"},
		{"mcp_server", "McpServer"},
		{"my_cool_plugin", "MyCoolPlugin"},
		{"already", "Already"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CatalogDataset", "catalog_dataset"},
		{"MCPServer", "m_c_p_server"},
		{"Simple", "simple"},
		{"alreadyLower", "already_lower"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPluginConfigMethods(t *testing.T) {
	cfg := &PluginConfig{Name: "dataset", Description: "Dataset catalog"}

	if got := cfg.PascalName(); got != "Dataset" {
		t.Errorf("PascalName() = %q, want %q", got, "Dataset")
	}
	if got := cfg.CamelName(); got != "dataset" {
		t.Errorf("CamelName() = %q, want %q", got, "dataset")
	}
	if got := cfg.CatalogPkg(); got != "datasetcatalog" {
		t.Errorf("CatalogPkg() = %q, want %q", got, "datasetcatalog")
	}
	if got := cfg.BasePath(); got != "/api/dataset_catalog/v1alpha1" {
		t.Errorf("BasePath() = %q, want %q", got, "/api/dataset_catalog/v1alpha1")
	}
	if got := cfg.SourceStructName(); got != "DatasetSource" {
		t.Errorf("SourceStructName() = %q, want %q", got, "DatasetSource")
	}
	if got := cfg.SourceConfigField(); got != "DatasetCatalogs" {
		t.Errorf("SourceConfigField() = %q, want %q", got, "DatasetCatalogs")
	}
	if got := cfg.SourceYAMLKey(); got != "dataset_catalogs" {
		t.Errorf("SourceYAMLKey() = %q, want %q", got, "dataset_catalogs")
	}
}

func TestEntityConfigMethods(t *testing.T) {
	entity := EntityConfig{
		Name:          "CatalogDataset",
		DatastoreType: "context",
		TypeName:      "kf.CatalogDataset",
	}

	if got := entity.SnakeName(); got != "catalog_dataset" {
		t.Errorf("SnakeName() = %q, want %q", got, "catalog_dataset")
	}
	if got := entity.RepoInterface(); got != "CatalogDatasetRepository" {
		t.Errorf("RepoInterface() = %q, want %q", got, "CatalogDatasetRepository")
	}
	if got := entity.SchemaType(); got != "schema.Context" {
		t.Errorf("SchemaType() = %q, want %q", got, "schema.Context")
	}
	if got := entity.PropertyFieldName(); got != "context_id" {
		t.Errorf("PropertyFieldName() = %q, want %q", got, "context_id")
	}
	if got := entity.DatastoreAddMethod(); got != "AddContext" {
		t.Errorf("DatastoreAddMethod() = %q, want %q", got, "AddContext")
	}
}

func TestEntityConfigArtifact(t *testing.T) {
	entity := EntityConfig{
		Name:          "CatalogDatasetArtifact",
		DatastoreType: "artifact",
		TypeName:      "kf.CatalogDatasetArtifact",
	}

	if got := entity.SchemaType(); got != "schema.Artifact" {
		t.Errorf("SchemaType() = %q, want %q", got, "schema.Artifact")
	}
	if got := entity.PropertyFieldName(); got != "artifact_id" {
		t.Errorf("PropertyFieldName() = %q, want %q", got, "artifact_id")
	}
	if got := entity.DatastoreAddMethod(); got != "AddArtifact" {
		t.Errorf("DatastoreAddMethod() = %q, want %q", got, "AddArtifact")
	}
}

func TestParseEntities(t *testing.T) {
	t.Run("valid entities", func(t *testing.T) {
		entities, err := parseEntities([]string{
			"CatalogDataset:context",
			"CatalogDatasetArtifact:artifact",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entities) != 2 {
			t.Fatalf("expected 2 entities, got %d", len(entities))
		}
		if entities[0].Name != "CatalogDataset" {
			t.Errorf("expected CatalogDataset, got %s", entities[0].Name)
		}
		if entities[0].DatastoreType != "context" {
			t.Errorf("expected context, got %s", entities[0].DatastoreType)
		}
		if entities[1].DatastoreType != "artifact" {
			t.Errorf("expected artifact, got %s", entities[1].DatastoreType)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := parseEntities([]string{"BadFormat"})
		if err == nil {
			t.Error("expected error for bad format")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := parseEntities([]string{"Foo:badtype"})
		if err == nil {
			t.Error("expected error for bad datastore type")
		}
	})
}

func TestPluginConfigEntityFilters(t *testing.T) {
	cfg := &PluginConfig{
		Name: "test",
		Entities: []EntityConfig{
			{Name: "A", DatastoreType: "context"},
			{Name: "B", DatastoreType: "artifact"},
			{Name: "C", DatastoreType: "execution"},
			{Name: "D", DatastoreType: "context"},
		},
	}

	contexts := cfg.ContextEntities()
	if len(contexts) != 2 {
		t.Errorf("expected 2 context entities, got %d", len(contexts))
	}

	artifacts := cfg.ArtifactEntities()
	if len(artifacts) != 1 {
		t.Errorf("expected 1 artifact entity, got %d", len(artifacts))
	}

	executions := cfg.ExecutionEntities()
	if len(executions) != 1 {
		t.Errorf("expected 1 execution entity, got %d", len(executions))
	}

	primary := cfg.PrimaryEntity()
	if primary.Name != "A" {
		t.Errorf("expected primary entity A, got %s", primary.Name)
	}
}
