package basecatalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticPreviewFiles builds a PreviewFiles whose path callbacks return fixed
// lists, mimicking how the server wires config and source paths.
func staticPreviewFiles(configPaths, sourcePaths []string) *PreviewFiles {
	return NewPreviewFiles(
		func() []string { return configPaths },
		func() []string { return sourcePaths },
	)
}

func writePreviewFixture(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestPreviewFilesReadFileAllowed(t *testing.T) {
	t.Run("absolute path in a config directory", func(t *testing.T) {
		dir := t.TempDir()
		writePreviewFixture(t, filepath.Join(dir, "catalog.yaml"), "config-dir-file")
		pf := staticPreviewFiles([]string{filepath.Join(dir, "sources.yaml")}, nil)

		data, err := pf.ReadFile(filepath.Join(dir, "catalog.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "config-dir-file", string(data))
	})

	t.Run("absolute path in a source catalog directory", func(t *testing.T) {
		dir := t.TempDir()
		shared := t.TempDir()
		writePreviewFixture(t, filepath.Join(shared, "catalog.yaml"), "source-dir-file")
		pf := staticPreviewFiles([]string{filepath.Join(dir, "sources.yaml")}, []string{filepath.Join(shared, "catalog.yaml")})

		data, err := pf.ReadFile(filepath.Join(shared, "catalog.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "source-dir-file", string(data))
	})

	t.Run("relative path resolves against the config directory, not the working directory", func(t *testing.T) {
		dir := t.TempDir()
		writePreviewFixture(t, filepath.Join(dir, "catalog.yaml"), "config-dir-file")
		pf := staticPreviewFiles([]string{filepath.Join(dir, "sources.yaml")}, nil)

		// Place a different file with the same name in the working directory to
		// prove it is never consulted.
		workDir := t.TempDir()
		writePreviewFixture(t, filepath.Join(workDir, "catalog.yaml"), "working-dir-file")
		t.Chdir(workDir)

		data, err := pf.ReadFile("catalog.yaml")
		require.NoError(t, err)
		assert.Equal(t, "config-dir-file", string(data))
	})

	t.Run("relative path resolves in the second config directory when missing from the first", func(t *testing.T) {
		first := t.TempDir()
		second := t.TempDir()
		writePreviewFixture(t, filepath.Join(second, "catalog.yaml"), "second-dir-file")
		pf := staticPreviewFiles(
			[]string{filepath.Join(first, "sources.yaml"), filepath.Join(second, "sources.yaml")},
			nil,
		)

		data, err := pf.ReadFile("catalog.yaml")
		require.NoError(t, err)
		assert.Equal(t, "second-dir-file", string(data))
	})

	t.Run("the first config directory that has the file wins", func(t *testing.T) {
		first := t.TempDir()
		second := t.TempDir()
		writePreviewFixture(t, filepath.Join(first, "catalog.yaml"), "first-dir-file")
		writePreviewFixture(t, filepath.Join(second, "catalog.yaml"), "second-dir-file")
		pf := staticPreviewFiles(
			[]string{filepath.Join(first, "sources.yaml"), filepath.Join(second, "sources.yaml")},
			nil,
		)

		data, err := pf.ReadFile("catalog.yaml")
		require.NoError(t, err)
		assert.Equal(t, "first-dir-file", string(data))
	})

	t.Run("relative paths do not resolve against source catalog directories", func(t *testing.T) {
		dir := t.TempDir()
		shared := t.TempDir()
		writePreviewFixture(t, filepath.Join(shared, "catalog.yaml"), "source-dir-file")
		pf := staticPreviewFiles([]string{filepath.Join(dir, "sources.yaml")}, []string{filepath.Join(shared, "catalog.yaml")})

		_, err := pf.ReadFile("catalog.yaml")
		require.ErrorIs(t, err, ErrPreviewPathNotAllowed)
	})

	t.Run("ConfigMap-style symlink chain stays allowed", func(t *testing.T) {
		dir := t.TempDir()
		// A ConfigMap volume mounts ..data as a relative symlink to a timestamped
		// directory, and each mounted file symlinks through ..data.
		require.NoError(t, os.Mkdir(filepath.Join(dir, "..2026_10_22"), 0o755))
		writePreviewFixture(t, filepath.Join(dir, "..2026_10_22", "catalog.yaml"), "configmap-file")
		require.NoError(t, os.Symlink("..2026_10_22", filepath.Join(dir, "..data")))
		require.NoError(t, os.Symlink("..data/catalog.yaml", filepath.Join(dir, "catalog.yaml")))
		pf := staticPreviewFiles([]string{filepath.Join(dir, "sources.yaml")}, nil)

		data, err := pf.ReadFile(filepath.Join(dir, "catalog.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "configmap-file", string(data))

		data, err = pf.ReadFile("catalog.yaml")
		require.NoError(t, err)
		assert.Equal(t, "configmap-file", string(data))
	})
}

func TestPreviewFilesReadFileRejected(t *testing.T) {
	newFixture := func(t *testing.T) (pf *PreviewFiles, allowedDir, outsideDir string) {
		allowedDir = t.TempDir()
		outsideDir = t.TempDir()
		writePreviewFixture(t, filepath.Join(allowedDir, "catalog.yaml"), "allowed-file")
		writePreviewFixture(t, filepath.Join(outsideDir, "secret.yaml"), "outside-file")
		pf = staticPreviewFiles([]string{filepath.Join(allowedDir, "sources.yaml")}, nil)
		return pf, allowedDir, outsideDir
	}

	t.Run("absolute path outside all roots", func(t *testing.T) {
		pf, _, outsideDir := newFixture(t)

		_, err := pf.ReadFile(filepath.Join(outsideDir, "secret.yaml"))
		require.ErrorIs(t, err, ErrPreviewPathNotAllowed)
	})

	t.Run("dot-dot traversal out of a root", func(t *testing.T) {
		pf, _, outsideDir := newFixture(t)

		_, err := pf.ReadFile("../" + filepath.Base(outsideDir) + "/secret.yaml")
		require.ErrorIs(t, err, ErrPreviewPathNotAllowed)
	})

	t.Run("symlink inside a root pointing outside it", func(t *testing.T) {
		pf, allowedDir, outsideDir := newFixture(t)
		require.NoError(t, os.Symlink(filepath.Join(outsideDir, "secret.yaml"), filepath.Join(allowedDir, "escape.yaml")))

		_, err := pf.ReadFile(filepath.Join(allowedDir, "escape.yaml"))
		require.ErrorIs(t, err, ErrPreviewPathNotAllowed)
		assert.NotContains(t, err.Error(), "outside-file")
	})

	t.Run("directory", func(t *testing.T) {
		pf, allowedDir, _ := newFixture(t)
		require.NoError(t, os.Mkdir(filepath.Join(allowedDir, "subdir"), 0o755))

		_, err := pf.ReadFile(filepath.Join(allowedDir, "subdir"))
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrPreviewPathModeDisabled)

		_, err = pf.ReadFile(filepath.Join(allowedDir))
		require.Error(t, err)
	})

	t.Run("rejections do not reveal whether an outside path exists", func(t *testing.T) {
		pf, _, outsideDir := newFixture(t)

		existing, existingErr := pf.ReadFile(filepath.Join(outsideDir, "secret.yaml"))
		missing, missingErr := pf.ReadFile(filepath.Join(outsideDir, "does-not-exist.yaml"))

		require.ErrorIs(t, existingErr, ErrPreviewPathNotAllowed)
		require.ErrorIs(t, missingErr, ErrPreviewPathNotAllowed)
		assert.Equal(t, existingErr.Error(), missingErr.Error())
		assert.Nil(t, existing)
		assert.Nil(t, missing)
	})
}

func TestPreviewFilesPathModeDisabled(t *testing.T) {
	pf := staticPreviewFiles(nil, nil)

	_, err := pf.ReadFile("catalog.yaml")
	require.ErrorIs(t, err, ErrPreviewPathModeDisabled)
}

func TestResolveYAMLCatalogPath(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		origin     string
		properties map[string]any
		expected   string
		expectOK   bool
	}{
		{
			name:       "absolute path is cleaned",
			sourceType: "yaml",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{"yamlCatalogPath": "/shared-data//catalog.yaml"},
			expected:   "/shared-data/catalog.yaml",
			expectOK:   true,
		},
		{
			name:       "relative path resolves against the origin directory",
			sourceType: "yaml",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{"yamlCatalogPath": "models.yaml"},
			expected:   "/catalog/models.yaml",
			expectOK:   true,
		},
		{
			name:       "non-yaml source type",
			sourceType: "hf",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{"yamlCatalogPath": "models.yaml"},
			expectOK:   false,
		},
		{
			name:       "missing yamlCatalogPath",
			sourceType: "yaml",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{},
			expectOK:   false,
		},
		{
			name:       "non-string yamlCatalogPath",
			sourceType: "yaml",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{"yamlCatalogPath": 42},
			expectOK:   false,
		},
		{
			name:       "empty yamlCatalogPath",
			sourceType: "yaml",
			origin:     "/catalog/sources.yaml",
			properties: map[string]any{"yamlCatalogPath": ""},
			expectOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := ResolveYAMLCatalogPath(tt.sourceType, tt.origin, tt.properties)
			assert.Equal(t, tt.expectOK, ok)
			if tt.expectOK {
				assert.Equal(t, tt.expected, path)
			} else {
				assert.Empty(t, path)
			}
		})
	}
}

// TestPreviewFilesReadFileOverlappingRoots proves that a file readable through
// any containing allowed root is served regardless of root order, while paths
// escaping every root stay rejected.
func TestPreviewFilesReadFileOverlappingRoots(t *testing.T) {
	cases := []struct {
		name       string
		wideFirst  bool
		relative   bool
		outsideAll bool
	}{
		{name: "absolute narrow first"},
		{name: "absolute wide first", wideFirst: true},
		{name: "relative config root with wider source root", relative: true},
		{name: "escape all roots narrow first", outsideAll: true},
		{name: "escape all roots wide first", wideFirst: true, outsideAll: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wide := t.TempDir()
			narrow := filepath.Join(wide, "sub")
			require.NoError(t, os.Mkdir(narrow, 0o755))
			writePreviewFixture(t, filepath.Join(wide, "catalog.yaml"), "allowed-file")
			target := "../catalog.yaml"
			if tc.outsideAll {
				outside := t.TempDir()
				target = filepath.Join(outside, "secret.yaml")
				writePreviewFixture(t, target, "outside-file")
			}
			candidate := filepath.Join(narrow, "catalog.yaml")
			require.NoError(t, os.Symlink(target, candidate))

			sourcePaths := []string{
				filepath.Join(narrow, "registered.yaml"),
				filepath.Join(wide, "registered.yaml"),
			}
			if tc.wideFirst {
				sourcePaths[0], sourcePaths[1] = sourcePaths[1], sourcePaths[0]
			}
			var configPaths []string
			requestPath := candidate
			if tc.relative {
				configPaths = []string{filepath.Join(narrow, "sources.yaml")}
				requestPath = "catalog.yaml"
			}
			pf := staticPreviewFiles(configPaths, sourcePaths)
			data, err := pf.ReadFile(requestPath)
			if tc.outsideAll {
				require.ErrorIs(t, err, ErrPreviewPathNotAllowed)
				assert.Equal(t, ErrPreviewPathNotAllowed, err)
				assert.Nil(t, data)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "allowed-file", string(data))
		})
	}
}

// TestPreviewFilesReadFileOverlappingSymlinkRoot proves retries are not a
// preference for the widest root: a separately allowed narrow directory may
// itself be a symlink outside the wider directory.
func TestPreviewFilesReadFileOverlappingSymlinkRoot(t *testing.T) {
	for _, wideFirst := range []bool{false, true} {
		name := "narrow first"
		if wideFirst {
			name = "wide first"
		}
		t.Run(name, func(t *testing.T) {
			wide := t.TempDir()
			target := t.TempDir()
			narrow := filepath.Join(wide, "sub")
			writePreviewFixture(t, filepath.Join(target, "catalog.yaml"), "allowed-source-file")
			require.NoError(t, os.Symlink(target, narrow))
			sourcePaths := []string{
				filepath.Join(narrow, "registered.yaml"),
				filepath.Join(wide, "registered.yaml"),
			}
			if wideFirst {
				sourcePaths[0], sourcePaths[1] = sourcePaths[1], sourcePaths[0]
			}
			data, err := staticPreviewFiles(nil, sourcePaths).ReadFile(filepath.Join(narrow, "catalog.yaml"))
			require.NoError(t, err)
			assert.Equal(t, "allowed-source-file", string(data))
		})
	}
}
