package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

type Generator struct {
	cfg  *PluginConfig
	tmpl *template.Template
}

func NewGenerator(cfg *PluginConfig) *Generator {
	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*"))
	return &Generator{cfg: cfg, tmpl: tmpl}
}

func (g *Generator) Run() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"plugin entry point", g.generatePlugin},
		{"domain package", g.generateDomain},
		{"entity models", g.generateModels},
		{"entity services", g.generateServices},
		{"OpenAPI spec", g.generateOpenAPI},
		{"OpenAPI build scripts", g.generateOpenAPIBuildScripts},
		{"shared file: config.go", g.modifyConfig},
		{"shared file: catalog.go", g.modifyBlankImport},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}

func (g *Generator) generatePlugin() error {
	dir := filepath.Join(g.cfg.RootDir, "catalog", "internal", "plugins", g.cfg.Name)
	return g.renderFiles(dir, map[string]string{
		"plugin.go":   "plugin.go.tmpl",
		"register.go": "register.go.tmpl",
	})
}

func (g *Generator) generateDomain() error {
	dir := filepath.Join(g.cfg.RootDir, "catalog", "internal", "catalog", g.cfg.CatalogPkg())
	return g.renderFiles(dir, map[string]string{
		"services.go":                  "services.go.tmpl",
		"loader.go":                    "loader.go.tmpl",
		"db_" + g.cfg.Name + ".go":    "db_provider.go.tmpl",
		"sources.go":                   "sources.go.tmpl",
	})
}

func (g *Generator) generateModels() error {
	dir := filepath.Join(g.cfg.RootDir, "catalog", "internal", "catalog", g.cfg.CatalogPkg(), "models")
	for _, entity := range g.cfg.Entities {
		data := struct {
			*PluginConfig
			Entity EntityConfig
		}{g.cfg, entity}

		filename := entity.SnakeName() + ".go"
		if err := g.renderFile(dir, filename, "entity_model.go.tmpl", data); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateServices() error {
	dir := filepath.Join(g.cfg.RootDir, "catalog", "internal", "catalog", g.cfg.CatalogPkg(), "service")
	for _, entity := range g.cfg.Entities {
		data := struct {
			*PluginConfig
			Entity EntityConfig
		}{g.cfg, entity}

		filename := entity.SnakeName() + ".go"
		if err := g.renderFile(dir, filename, "entity_service.go.tmpl", data); err != nil {
			return err
		}

		mappingsFilename := "entity_mappings_" + entity.SnakeName() + ".go"
		if err := g.renderFile(dir, mappingsFilename, "entity_mappings.go.tmpl", data); err != nil {
			return err
		}

		mappingsTestFilename := "entity_mappings_" + entity.SnakeName() + "_test.go"
		if err := g.renderFile(dir, mappingsTestFilename, "entity_mappings_test.go.tmpl", data); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateOpenAPI() error {
	dir := filepath.Join(g.cfg.RootDir, "api", "openapi", "src", "plugins")
	return g.renderFiles(dir, map[string]string{
		g.cfg.Name + ".yaml": "openapi.yaml.tmpl",
	})
}

func (g *Generator) generateOpenAPIBuildScripts() error {
	pluginDir := filepath.Join(g.cfg.RootDir, "catalog", "plugins", g.cfg.Name)
	scriptsDir := filepath.Join(pluginDir, "scripts")

	if err := g.renderFile(pluginDir, ".openapi-generator-ignore", "openapi-generator-ignore.tmpl", g.cfg); err != nil {
		return err
	}

	if err := g.renderFile(scriptsDir, "gen_openapi_server.sh", "gen_openapi_server.sh.tmpl", g.cfg); err != nil {
		return err
	}

	// Make the script executable
	if !g.cfg.DryRun {
		scriptPath := filepath.Join(scriptsDir, "gen_openapi_server.sh")
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			return fmt.Errorf("making script executable: %w", err)
		}
	}

	return nil
}

func (g *Generator) renderFiles(dir string, files map[string]string) error {
	for outFile, tmplName := range files {
		if err := g.renderFile(dir, outFile, tmplName, g.cfg); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) renderFile(dir, filename, tmplName string, data any) error {
	outPath := filepath.Join(dir, filename)

	if g.cfg.DryRun {
		fmt.Printf("  would create: %s\n", relPath(g.cfg.RootDir, outPath))
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if _, err := os.Stat(outPath); err == nil {
		fmt.Printf("  skipped (exists): %s\n", relPath(g.cfg.RootDir, outPath))
		return nil
	}

	var buf bytes.Buffer
	if err := g.tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return fmt.Errorf("executing template %s: %w", tmplName, err)
	}

	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	fmt.Printf("  created: %s\n", relPath(g.cfg.RootDir, outPath))
	return nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
