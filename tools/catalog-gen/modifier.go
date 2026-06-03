package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// modifyConfig adds a PluginSource field to SourceConfig and a validateSourceIDs call to Validate().
func (g *Generator) modifyConfig() error {
	path := filepath.Join(g.cfg.RootDir, "catalog", "internal", "catalog", "basecatalog", "config.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config.go: %w", err)
	}
	text := string(content)

	fieldName := g.cfg.SourceConfigField()
	if strings.Contains(text, fieldName) {
		if g.cfg.DryRun {
			fmt.Printf("  would skip (already present): %s\n", relPath(g.cfg.RootDir, path))
		} else {
			fmt.Printf("  skipped (already present): %s\n", relPath(g.cfg.RootDir, path))
		}
		return nil
	}

	if g.cfg.DryRun {
		fmt.Printf("  would modify: %s\n", relPath(g.cfg.RootDir, path))
		return nil
	}

	// Add field to SourceConfig struct — insert before the Labels field
	configField := fmt.Sprintf("\n\t// %s contains %s catalog source definitions\n\t%s []PluginSource `yaml:\"%s,omitempty\" json:\"%s,omitempty\"`\n",
		fieldName, g.cfg.Name, fieldName, g.cfg.SourceYAMLKey(), g.cfg.SourceYAMLKey())

	// Find the Labels comment+field block and insert before it
	labelsCommentIdx := strings.Index(text, "// Labels contains")
	if labelsCommentIdx < 0 {
		labelsCommentIdx = strings.Index(text, "Labels []map[string]any")
	}
	if labelsCommentIdx > 0 {
		lineStart := strings.LastIndex(text[:labelsCommentIdx], "\n")
		if lineStart > 0 {
			text = text[:lineStart] + configField + text[lineStart:]
		}
	}

	// Add validateSourceIDs call in Validate() — insert before the named queries validation
	validationLine := fmt.Sprintf("\tif err := validateSourceIDs(%q, c.%s, seen); err != nil {\n\t\treturn err\n\t}\n",
		g.cfg.Name, fieldName)

	namedQueriesIdx := strings.Index(text, "if err := ValidateNamedQueries")
	if namedQueriesIdx > 0 {
		lineStart := strings.LastIndex(text[:namedQueriesIdx], "\n")
		if lineStart > 0 {
			text = text[:lineStart+1] + validationLine + text[lineStart+1:]
		}
	}

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing config.go: %w", err)
	}

	fmt.Printf("  modified: %s\n", relPath(g.cfg.RootDir, path))
	return nil
}

// modifyBlankImport adds the blank import to catalog.go.
func (g *Generator) modifyBlankImport() error {
	path := filepath.Join(g.cfg.RootDir, "catalog", "cmd", "catalog.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading catalog.go: %w", err)
	}
	text := string(content)

	importPath := fmt.Sprintf(`_ "github.com/kubeflow/hub/catalog/internal/plugins/%s"`, g.cfg.Name)
	if strings.Contains(text, importPath) {
		if g.cfg.DryRun {
			fmt.Printf("  would skip (already present): %s\n", relPath(g.cfg.RootDir, path))
		} else {
			fmt.Printf("  skipped (already present): %s\n", relPath(g.cfg.RootDir, path))
		}
		return nil
	}

	if g.cfg.DryRun {
		fmt.Printf("  would modify: %s\n", relPath(g.cfg.RootDir, path))
		return nil
	}

	blankImportLine := fmt.Sprintf("\t%s", importPath)

	lastBlankImport := strings.LastIndex(text, `_ "github.com/kubeflow/hub/catalog/internal/plugins/`)
	if lastBlankImport > 0 {
		lineEnd := strings.Index(text[lastBlankImport:], "\n")
		if lineEnd > 0 {
			insertPos := lastBlankImport + lineEnd
			text = text[:insertPos] + "\n" + blankImportLine + text[insertPos:]
		}
	}

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing catalog.go: %w", err)
	}

	fmt.Printf("  modified: %s\n", relPath(g.cfg.RootDir, path))
	return nil
}
