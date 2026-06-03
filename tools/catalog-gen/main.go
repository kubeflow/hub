package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "catalog-gen",
		Short: "Code generator for catalog plugins",
	}

	initCmd := newInitCmd()
	root.AddCommand(initCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	var (
		name        string
		description string
		entities    []string
		dryRun      bool
		rootDir     string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a new catalog plugin scaffold",
		Long: `Generate all boilerplate files for a new catalog plugin.

Example:
  catalog-gen init \
    --name dataset \
    --description "Dataset catalog" \
    --entity CatalogDataset:context \
    --entity CatalogDatasetArtifact:artifact`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if description == "" {
				return fmt.Errorf("--description is required")
			}
			if len(entities) == 0 {
				return fmt.Errorf("at least one --entity is required")
			}

			parsed, err := parseEntities(entities)
			if err != nil {
				return err
			}

			resolvedRoot, err := resolveRoot(rootDir)
			if err != nil {
				return err
			}

			cfg := &PluginConfig{
				Name:        name,
				Description: description,
				Entities:    parsed,
				RootDir:     resolvedRoot,
				DryRun:      dryRun,
			}

			gen := NewGenerator(cfg)
			if err := gen.Run(); err != nil {
				return err
			}

			if !dryRun {
				runGoimports(resolvedRoot, cfg)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Plugin name in snake_case (e.g., dataset)")
	cmd.Flags().StringVar(&description, "description", "", "Human-readable plugin description")
	cmd.Flags().StringSliceVar(&entities, "entity", nil, "Entity in Name:type format (repeatable). Type: context, artifact, execution")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview files without writing")
	cmd.Flags().StringVar(&rootDir, "root", ".", "Repository root directory")

	return cmd
}

func resolveRoot(rootDir string) (string, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("resolving root: %w", err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		return "", fmt.Errorf("root %s does not contain go.mod — use --root to set the repository root", abs)
	}
	return abs, nil
}

func parseEntities(raw []string) ([]EntityConfig, error) {
	var result []EntityConfig
	for _, e := range raw {
		parts := strings.SplitN(e, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --entity format %q: expected Name:type (e.g., CatalogDataset:context)", e)
		}
		name := parts[0]
		dsType := strings.ToLower(parts[1])

		switch dsType {
		case "context", "artifact", "execution":
		default:
			return nil, fmt.Errorf("invalid datastore type %q for entity %s: must be context, artifact, or execution", dsType, name)
		}

		result = append(result, EntityConfig{
			Name:          name,
			DatastoreType: dsType,
			TypeName:      fmt.Sprintf("kf.%s", name),
		})
	}
	return result, nil
}

func runGoimports(rootDir string, cfg *PluginConfig) {
	goimports := filepath.Join(rootDir, "bin", "goimports")
	if _, err := os.Stat(goimports); err != nil {
		goimports = "goimports"
	}

	paths := []string{
		filepath.Join(rootDir, "catalog", "internal", "plugins", cfg.Name),
		filepath.Join(rootDir, "catalog", "internal", "catalog", cfg.Name+"catalog"),
		filepath.Join(rootDir, "catalog", "internal", "catalog", "basecatalog", "config.go"),
		filepath.Join(rootDir, "catalog", "cmd", "catalog.go"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(goimports, "-w", p)
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: goimports failed on %s: %v\n", relPath(rootDir, p), err)
			}
		}
	}
}
