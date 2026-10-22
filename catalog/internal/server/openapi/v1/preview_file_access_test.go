package v1_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	v1 "github.com/kubeflow/hub/catalog/internal/server/openapi/v1"
	v1alpha1 "github.com/kubeflow/hub/catalog/internal/server/openapi/v1alpha1"
	"github.com/stretchr/testify/require"
)

// osErrorFragments must never appear in a rejection response body: preview
// path rejections are confinement decisions, not OS file errors.
var osErrorFragments = []string{
	"no such file",
	"is a directory",
	"permission denied",
	"invalid argument",
	"path escapes",
}

// TestPreviewCatalogSourcePathMode covers path-mode YAML previews: uploaded
// catalogData always wins, and yamlCatalogPath is only read when it stays
// inside the directories the server loads catalog sources from.
func TestPreviewCatalogSourcePathMode(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	kinds := []struct {
		name       string
		assetType  string
		listKey    string
		sentinel   string // name in the catalog file inside the allowed directory
		outside    string // name in the file outside the allowed directories
		uploadName string // name in uploaded catalogData
	}{
		{
			name:       "models",
			assetType:  "",
			listKey:    "models",
			sentinel:   "allowed-path-model",
			outside:    "outside-path-model",
			uploadName: "uploaded-model",
		},
		{
			name:       "mcp_servers",
			assetType:  "mcp_servers",
			listKey:    "mcp_servers",
			sentinel:   "allowed-path-server",
			outside:    "outside-path-server",
			uploadName: "uploaded-server",
		},
	}

	for _, kind := range kinds {
		t.Run(kind.name, func(t *testing.T) {
			// The allowed directory holds a sources file, a catalog file, and a
			// symlink that escapes it; the outside directory holds the file the
			// symlink and rejected paths point at.
			allowedDir := t.TempDir()
			outsideDir := t.TempDir()
			allowedCatalog := filepath.Join(allowedDir, "catalog.yaml")
			require.NoError(t, os.WriteFile(allowedCatalog, fmt.Appendf(nil, "%s:\n  - name: %s\n", kind.listKey, kind.sentinel), 0o600))
			outsideSecret := filepath.Join(outsideDir, "secret.yaml")
			require.NoError(t, os.WriteFile(outsideSecret, fmt.Appendf(nil, "%s:\n  - name: %s\n", kind.listKey, kind.outside), 0o600))
			require.NoError(t, os.Symlink(outsideSecret, filepath.Join(allowedDir, "escape.yaml")))
			previewFiles := basecatalog.NewPreviewFiles(
				func() []string { return []string{filepath.Join(allowedDir, "sources.yaml")} },
				nil,
			)

			versions := []struct {
				name    string
				handler http.Handler
			}{
				{
					name: "v1",
					handler: v1.NewRouter(v1.NewModelCatalogServiceAPIController(
						v1.NewModelCatalogServiceAPIService(nil, nil, nil, nil, nil, nil, v1.WithPreviewFiles(previewFiles)))),
				},
				{
					name: "v1alpha1",
					handler: v1alpha1.NewRouter(v1alpha1.NewModelCatalogServiceAPIController(
						v1alpha1.NewModelCatalogServiceAPIService(nil, nil, nil, nil, nil, nil, v1alpha1.WithPreviewFiles(previewFiles)))),
				},
			}

			cases := []struct {
				name          string
				path          string
				uploadCatalog bool
				expectCode    int
				expectName    string
				forbidNames   []string
			}{
				{
					name:          "uploaded catalogData wins over any path",
					path:          outsideSecret,
					uploadCatalog: true,
					expectCode:    http.StatusOK,
					expectName:    kind.uploadName,
					forbidNames:   []string{kind.outside},
				},
				{
					name:       "absolute path in an allowed directory",
					path:       allowedCatalog,
					expectCode: http.StatusOK,
					expectName: kind.sentinel,
				},
				{
					name:       "relative path resolves in the config directory",
					path:       "catalog.yaml",
					expectCode: http.StatusOK,
					expectName: kind.sentinel,
				},
				{
					name:        "absolute path outside the allowed directories",
					path:        outsideSecret,
					expectCode:  http.StatusUnprocessableEntity,
					forbidNames: []string{kind.outside},
				},
				{
					name:        "dot-dot traversal out of an allowed directory",
					path:        filepath.Join("..", filepath.Base(outsideDir), "secret.yaml"),
					expectCode:  http.StatusUnprocessableEntity,
					forbidNames: []string{kind.outside},
				},
				{
					name:        "symlink in an allowed directory pointing outside it",
					path:        filepath.Join(allowedDir, "escape.yaml"),
					expectCode:  http.StatusUnprocessableEntity,
					forbidNames: []string{kind.outside},
				},
			}

			for _, version := range versions {
				for _, tt := range cases {
					t.Run(version.name+"/"+tt.name, func(t *testing.T) {
						config := fmt.Sprintf("type: yaml\nproperties:\n  yamlCatalogPath: %q\n", tt.path)
						if kind.assetType != "" {
							config = "assetType: " + kind.assetType + "\n" + config
						}

						var body bytes.Buffer
						writer := multipart.NewWriter(&body)
						part, err := writer.CreateFormFile("config", "config.yaml")
						require.NoError(t, err)
						_, err = part.Write([]byte(config))
						require.NoError(t, err)
						if tt.uploadCatalog {
							part, err = writer.CreateFormFile("catalogData", "catalog.yaml")
							require.NoError(t, err)
							_, err = part.Write(fmt.Appendf(nil, "%s:\n  - name: %s\n", kind.listKey, kind.uploadName))
							require.NoError(t, err)
						}
						require.NoError(t, writer.Close())

						request := httptest.NewRequest(http.MethodPost, "/api/model_catalog/"+version.name+"/sources/preview", &body)
						request.Header.Set("Content-Type", writer.FormDataContentType())
						recorder := httptest.NewRecorder()
						version.handler.ServeHTTP(recorder, request)

						require.Equal(t, tt.expectCode, recorder.Code, recorder.Body.String())
						if tt.expectName != "" {
							require.Contains(t, recorder.Body.String(), tt.expectName)
						}
						for _, forbidden := range tt.forbidNames {
							require.NotContains(t, recorder.Body.String(), forbidden)
						}
						if tt.expectCode != http.StatusOK {
							for _, fragment := range osErrorFragments {
								require.NotContains(t, recorder.Body.String(), fragment)
							}
						}
					})
				}
			}
		})
	}
}
