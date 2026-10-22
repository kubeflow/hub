package basecatalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// YAMLCatalogPathKey is the source properties key holding the catalog file
// path of a YAML source.
const YAMLCatalogPathKey = "yamlCatalogPath"

var (
	// ErrPreviewPathModeDisabled is returned by PreviewFiles.ReadFile when the
	// server has no catalog directories to read from, because no sources files
	// or source catalog paths are configured. Preview path mode is off.
	ErrPreviewPathModeDisabled = errors.New("yamlCatalogPath preview is not enabled on this server")

	// ErrPreviewPathNotAllowed is returned by PreviewFiles.ReadFile when the
	// requested path is not inside a directory the server loads catalog
	// sources from. The error is identical for paths outside those directories
	// and for paths that do not exist, so it cannot be used as an existence
	// oracle for files outside them.
	ErrPreviewPathNotAllowed = errors.New("yamlCatalogPath is not inside a catalog source directory")
)

// PreviewFiles reads yamlCatalogPath files for source previews, confined to
// directories the server is already configured to load catalogs from: the
// directory of each configured sources file, and the directory of each loaded
// source's catalog file. Reads go through os.Root, which refuses .. traversal
// and symlinks that resolve outside the root.
//
// The path lists are captured as callbacks and re-evaluated on every read, so
// sources added or removed by a hot reload count immediately. A nil
// *PreviewFiles, or one whose callbacks yield no directories, disables path
// mode (ErrPreviewPathModeDisabled).
type PreviewFiles struct {
	configPaths func() []string // --catalogs-path files
	sourcePaths func() []string // resolved yamlCatalogPath of loaded sources
}

// NewPreviewFiles builds a PreviewFiles from callbacks returning the configured
// sources file paths and the resolved catalog file paths of loaded YAML
// sources. Either callback may be nil.
func NewPreviewFiles(configPaths, sourcePaths func() []string) *PreviewFiles {
	return &PreviewFiles{
		configPaths: configPaths,
		sourcePaths: sourcePaths,
	}
}

// ReadFile returns the contents of the catalog file at path. Absolute paths
// must fall inside one of the allowed directories. Relative paths are resolved
// against each configured sources file directory, in order, and the first file
// that exists wins — the same way saved sources resolve them. Every candidate
// is tried through each allowed root containing it before it is rejected, so a
// file readable through any containing root is served regardless of root order.
func (p *PreviewFiles) ReadFile(path string) ([]byte, error) {
	if p == nil {
		return nil, ErrPreviewPathModeDisabled
	}
	configDirs, roots := p.directories()
	if len(roots) == 0 {
		return nil, ErrPreviewPathModeDisabled
	}
	if path == "" {
		return nil, ErrPreviewPathNotAllowed
	}

	var candidates []string
	if filepath.IsAbs(path) {
		candidates = []string{filepath.Clean(path)}
	} else {
		for _, dir := range configDirs {
			candidates = append(candidates, filepath.Join(dir, path))
		}
	}

	for _, candidate := range candidates {
		var readErr error
		escaped := false
		for _, root := range roots {
			rel, err := filepath.Rel(root, candidate)
			if err != nil || !filepath.IsLocal(rel) {
				continue
			}
			data, err := readFromRoot(root, rel)
			if err == nil {
				return data, nil
			}
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if isPathEscape(err) {
				escaped = true
				continue
			}
			if readErr == nil {
				readErr = err
			}
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, readErr)
		}
		if escaped {
			return nil, ErrPreviewPathNotAllowed
		}
	}

	return nil, ErrPreviewPathNotAllowed
}

// directories returns the configured sources file directories (used to resolve
// relative paths) and every allowed root (those directories plus the
// directories of loaded source catalog files). Both lists are absolute,
// cleaned, and deduplicated, preserving configuration order.
func (p *PreviewFiles) directories() (configDirs, roots []string) {
	seen := make(map[string]struct{})
	add := func(file string) (dir string, added bool) {
		abs, err := filepath.Abs(file)
		if err != nil {
			return "", false
		}
		dir = filepath.Dir(abs)
		if _, ok := seen[dir]; ok {
			return dir, false
		}
		seen[dir] = struct{}{}
		return dir, true
	}

	if p.configPaths != nil {
		for _, file := range p.configPaths() {
			if dir, added := add(file); added {
				configDirs = append(configDirs, dir)
				roots = append(roots, dir)
			}
		}
	}
	if p.sourcePaths != nil {
		for _, file := range p.sourcePaths() {
			if dir, added := add(file); added {
				roots = append(roots, dir)
			}
		}
	}
	return configDirs, roots
}

// readFromRoot reads rel within root through os.Root, which refuses symlinks
// (and symlink chains) that resolve outside root. Relative ..data symlinks
// used by ConfigMap mounts stay readable because they resolve inside root.
func readFromRoot(root, rel string) ([]byte, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return r.ReadFile(rel)
}

// isPathEscape reports whether err is the error os.Root returns when a path or
// symlink resolves outside the root. The error has no exported sentinel, so it
// is matched by message; if the message ever changes, escapes fail closed into
// the wrapped-error path and are still refused.
func isPathEscape(err error) bool {
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Err == nil {
		return false
	}
	return pathErr.Err.Error() == "path escapes from parent"
}

// ResolveYAMLCatalogPath returns the catalog file path of a YAML source,
// resolved the way the loaders resolve it: absolute paths are cleaned, and
// relative paths are joined to dir, the directory of the config file the
// source was defined in. It returns false when the source is not a YAML
// source or has no yamlCatalogPath string property.
func ResolveYAMLCatalogPath(sourceType, origin string, properties map[string]any) (string, bool) {
	if sourceType != "yaml" {
		return "", false
	}
	path, ok := properties[YAMLCatalogPathKey].(string)
	if !ok || path == "" {
		return "", false
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), true
	}
	return filepath.Join(filepath.Dir(origin), path), true
}
