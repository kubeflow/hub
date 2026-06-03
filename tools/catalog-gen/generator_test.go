package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	setupTestRepo(t, tmpDir)

	cfg := &PluginConfig{
		Name:        "test",
		Description: "Test catalog",
		Entities: []EntityConfig{
			{Name: "TestEntity", DatastoreType: "context", TypeName: "kf.TestEntity"},
		},
		RootDir: tmpDir,
		DryRun:  true,
	}

	gen := NewGenerator(cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	pluginDir := filepath.Join(tmpDir, "catalog", "internal", "plugins", "test")
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Error("dry-run should not create plugin directory")
	}
}

func TestGeneratorCreateFiles(t *testing.T) {
	tmpDir := t.TempDir()

	setupTestRepo(t, tmpDir)

	cfg := &PluginConfig{
		Name:        "test",
		Description: "Test catalog",
		Entities: []EntityConfig{
			{Name: "TestEntity", DatastoreType: "context", TypeName: "kf.TestEntity"},
		},
		RootDir: tmpDir,
		DryRun:  false,
	}

	gen := NewGenerator(cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("generator failed: %v", err)
	}

	expectedFiles := []string{
		"catalog/internal/plugins/test/plugin.go",
		"catalog/internal/plugins/test/register.go",
		"catalog/internal/catalog/testcatalog/services.go",
		"catalog/internal/catalog/testcatalog/loader.go",
		"catalog/internal/catalog/testcatalog/sources.go",
		"catalog/internal/catalog/testcatalog/db_test.go",
		"catalog/internal/catalog/testcatalog/models/test_entity.go",
		"catalog/internal/catalog/testcatalog/service/test_entity.go",
		"catalog/internal/catalog/testcatalog/service/test_entity_entity_mappings.go",
		"catalog/internal/catalog/testcatalog/service/test_entity_entity_mappings_test.go",
		"api/openapi/src/plugins/test.yaml",
		"catalog/plugins/test/.openapi-generator-ignore",
		"catalog/plugins/test/scripts/gen_openapi_server.sh",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not created: %s", f)
		}
	}
}

func TestGeneratorIdempotency(t *testing.T) {
	tmpDir := t.TempDir()

	setupTestRepo(t, tmpDir)

	cfg := &PluginConfig{
		Name:        "test",
		Description: "Test catalog",
		Entities: []EntityConfig{
			{Name: "TestEntity", DatastoreType: "context", TypeName: "kf.TestEntity"},
		},
		RootDir: tmpDir,
		DryRun:  false,
	}

	gen := NewGenerator(cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	specPath := filepath.Join(tmpDir, "catalog", "internal", "db", "service", "spec.go")
	specBefore, _ := os.ReadFile(specPath)

	if err := gen.Run(); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	specAfter, _ := os.ReadFile(specPath)
	if string(specBefore) != string(specAfter) {
		t.Error("spec.go changed on second run — not idempotent")
	}
}

func setupTestRepo(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		"catalog/internal/db/service",
		"catalog/internal/catalog/basecatalog",
		"catalog/cmd",
		"api/openapi/src/plugins",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	specContent := `package service

import (
	"github.com/kubeflow/hub/internal/platform/datastore"
)

const (
	ExistingTypeName = "kf.Existing"
)

func DatastoreSpec() *datastore.Spec {
	return datastore.NewSpec().
		AddOther(nil)
}
`
	writeFile(t, root, "catalog/internal/db/service/spec.go", specContent)

	sourceTypesContent := `package basecatalog

type ModelSource struct {
	Name string
}
`
	writeFile(t, root, "catalog/internal/catalog/basecatalog/source_types.go", sourceTypesContent)

	configContent := `package basecatalog

import "fmt"

type PluginSource struct {
	ID string
}

func (s PluginSource) GetId() string { return s.ID }

type SourceConfig struct {
	Labels []map[string]any ` + "`" + `yaml:"labels,omitempty"` + "`" + `
}

type sourceIdentifiable interface {
	GetId() string
}

func validateSourceIDs[T sourceIdentifiable](section string, sources []T, globalSeen map[string]bool) error {
	for _, s := range sources {
		id := s.GetId()
		if globalSeen[id] {
			return fmt.Errorf("duplicate %s id: %s", section, id)
		}
		globalSeen[id] = true
	}
	return nil
}

func (c *SourceConfig) Validate() error {
	seen := make(map[string]bool)
	_ = seen
	if err := ValidateNamedQueries(nil); err != nil {
		return fmt.Errorf("invalid: %w", err)
	}
	return nil
}

func ValidateNamedQueries(_ any) error { return nil }
`
	writeFile(t, root, "catalog/internal/catalog/basecatalog/config.go", configContent)

	catalogContent := `package cmd

import (
	_ "github.com/kubeflow/hub/catalog/internal/plugins/model"
)
`
	writeFile(t, root, "catalog/cmd/catalog.go", catalogContent)

	makefileContent := `.PHONY: gen/openapi-server
gen/openapi-server: internal/server/openapi/api_model_catalog_service.go

internal/server/openapi/api_model_catalog_service.go: ../api/openapi/src/plugins/model.yaml
	./plugins/model/scripts/gen_openapi_server.sh
`
	writeFile(t, root, "catalog/Makefile", makefileContent)
}

func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestModifierConfigIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestRepo(t, tmpDir)

	cfg := &PluginConfig{
		Name:        "test",
		Description: "Test catalog",
		Entities: []EntityConfig{
			{Name: "TestEntity", DatastoreType: "context", TypeName: "kf.TestEntity"},
		},
		RootDir: tmpDir,
	}

	gen := NewGenerator(cfg)

	if err := gen.modifyConfig(); err != nil {
		t.Fatalf("first modifyConfig failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, "catalog", "internal", "catalog", "basecatalog", "config.go")
	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "TestCatalogs") {
		t.Error("TestCatalogs field not added to SourceConfig")
	}

	if err := gen.modifyConfig(); err != nil {
		t.Fatalf("second modifyConfig failed: %v", err)
	}

	content2, _ := os.ReadFile(configPath)
	if string(content) != string(content2) {
		t.Error("config.go changed on second run — modifyConfig not idempotent")
	}
}

func TestModifierBlankImport(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestRepo(t, tmpDir)

	cfg := &PluginConfig{
		Name:    "test",
		RootDir: tmpDir,
		Entities: []EntityConfig{
			{Name: "TestEntity", DatastoreType: "context", TypeName: "kf.TestEntity"},
		},
	}

	gen := NewGenerator(cfg)

	if err := gen.modifyBlankImport(); err != nil {
		t.Fatalf("modifyBlankImport failed: %v", err)
	}

	catalogPath := filepath.Join(tmpDir, "catalog", "cmd", "catalog.go")
	content, _ := os.ReadFile(catalogPath)
	expected := `_ "github.com/kubeflow/hub/catalog/internal/plugins/test"`
	if !strings.Contains(string(content), expected) {
		t.Error("blank import not added")
	}

	// Second run should be idempotent
	if err := gen.modifyBlankImport(); err != nil {
		t.Fatalf("second modifyBlankImport failed: %v", err)
	}

	content2, _ := os.ReadFile(catalogPath)
	count := strings.Count(string(content2), expected)
	if count != 1 {
		t.Errorf("expected 1 occurrence of blank import, got %d", count)
	}
}
