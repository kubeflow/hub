package openapi_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/api"
	docs "github.com/kubeflow/hub/ui/bff/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggo/swag/v2"
	"gopkg.in/yaml.v3"
)

func TestGeneratedOpenAPI(t *testing.T) {
	jsonSpec, err := os.ReadFile("swagger.json")
	require.NoError(t, err)
	assert.JSONEq(t, string(jsonSpec), string(docs.Document))
	yamlSpec, err := os.ReadFile("swagger.yaml")
	require.NoError(t, err)
	var yamlDoc map[string]interface{}
	require.NoError(t, yaml.Unmarshal(yamlSpec, &yamlDoc))
	yamlJSON, err := json.Marshal(yamlDoc)
	require.NoError(t, err)
	assert.JSONEq(t, string(jsonSpec), string(yamlJSON))

	doc := swag.New(swag.GenerateOpenAPI3Doc(true)).GetOpenAPI()
	require.NoError(t, json.Unmarshal(jsonSpec, doc))
	assert.Equal(t, "3.1.0", doc.OpenAPI)
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "/", doc.Servers[0].Spec.URL)
	assert.Empty(t, doc.Paths.Spec.Paths["/healthcheck"].Spec.Spec.Get.Spec.Security)
	assert.NotContains(t, doc.Paths.Spec.Paths, "/api/v1/healthcheck")
	security := doc.Components.Spec.SecuritySchemes
	assert.Equal(t, "http", security["BearerAuth"].Spec.Spec.Type)
	assert.Equal(t, "bearer", security["BearerAuth"].Spec.Spec.Scheme)
	assert.Equal(t, "apiKey", security["KubeflowUserID"].Spec.Spec.Type)

	// Compare generated operations with actual router registrations, including
	// aliases and conditionally registered routes, without starting Kubernetes.
	files, err := filepath.Glob("../internal/api/*.go")
	require.NoError(t, err)
	sources := map[string]*ast.File{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		require.NoError(t, err)
		sources[name] = file
	}
	constants := map[string]ast.Expr{}
	for _, file := range sources {
		for _, decl := range file.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.CONST {
				for _, item := range decl.Specs {
					value := item.(*ast.ValueSpec)
					for i, name := range value.Names {
						if i < len(value.Values) {
							constants[name.Name] = value.Values[i]
						}
					}
				}
			}
		}
	}
	var eval func(ast.Expr) string
	eval = func(expr ast.Expr) string {
		switch expr := expr.(type) {
		case *ast.BasicLit:
			value, err := strconv.Unquote(expr.Value)
			require.NoError(t, err)
			return value
		case *ast.Ident:
			return eval(constants[expr.Name])
		case *ast.BinaryExpr:
			require.Equal(t, token.ADD, expr.Op)
			return eval(expr.X) + eval(expr.Y)
		default:
			t.Fatalf("unsupported route expression: %T", expr)
			return ""
		}
	}
	pathParam := regexp.MustCompile(`[:*]([A-Za-z_][A-Za-z_0-9]*)`)
	expected := map[string]bool{}
	ast.Inspect(sources["../internal/api/app.go"], func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !strings.Contains(" GET POST PATCH DELETE ", " "+method.Sel.Name+" ") {
			return true
		}
		path := eval(call.Args[0])
		if strings.HasPrefix(path, "/api/v1/swagger/") {
			return true // Swagger UI serves its own assets, not a business API.
		}
		path = pathParam.ReplaceAllString(path, "{$1}")
		expected[strings.ToLower(method.Sel.Name)+" "+path] = true
		return true
	})
	require.NotEmpty(t, expected)
	actual := map[string]bool{}
	for path, pathRef := range doc.Paths.Spec.Paths {
		item := pathRef.Spec.Spec
		for _, method := range []string{"get", "post", "patch", "delete"} {
			op := item.Get
			switch method {
			case "post":
				op = item.Post
			case "patch":
				op = item.Patch
			case "delete":
				op = item.Delete
			}
			if op == nil {
				continue
			}
			key := method + " " + path
			actual[key] = true
			params := map[string]bool{}
			for _, paramRef := range op.Spec.Parameters {
				param := paramRef.Spec.Spec
				if param.In == "path" {
					assert.Contains(t, path, "{"+param.Name+"}", key)
					assert.True(t, param.Required, key)
					params[param.Name] = true
				}
			}
			for _, match := range regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(path, -1) {
				assert.True(t, params[match[1]], "%s missing path parameter %s", key, match[1])
			}
		}
	}
	assert.Equal(t, expected, actual, "generated spec must cover exactly the registered API routes")

	// Union wrappers and nil metadata must not leak Go field names into the API.
	schemas := doc.Components.Spec.Schemas
	assert.Contains(t, schemas["api.SwaggerMetadataValue"].Spec.Properties, "string_value")
	assert.NotContains(t, schemas["api.SwaggerMetadataValue"].Spec.Properties, "MetadataStringValue")
	assert.Contains(t, schemas["api.SwaggerArtifact"].Spec.Properties, "uri")
	assert.NotContains(t, schemas["api.SwaggerArtifact"].Spec.Properties, "ModelArtifact")
	assert.Len(t, schemas["api.SwaggerArtifact"].Spec.Properties["value"].Spec.OneOf, 2)
	assert.Contains(t, schemas["api.RegisteredModelEnvelope"].Spec.Properties, "data")
	assert.NotContains(t, schemas["api.RegisteredModelEnvelope"].Spec.Properties, "metadata")
	assert.Empty(t, schemas["openapi.RegisteredModelUpdate"].Spec.Required)

	createModel := doc.Paths.Spec.Paths["/api/v1/model_registry/{model_registry_id}/registered_models"].Spec.Spec.Post.Spec
	require.NotNil(t, createModel.RequestBody)
	assert.True(t, createModel.RequestBody.Spec.Spec.Required)
	assert.Equal(t, "#/components/schemas/api.RegisteredModelEnvelope", createModel.RequestBody.Spec.Spec.Content["application/json"].Spec.Schema.Ref.Ref)
	logo := doc.Paths.Spec.Paths["/api/v1/mcp_catalog/mcp_servers/{server_id}/logo"].Spec.Spec.Get.Spec
	assert.Contains(t, logo.Responses.Spec.Response["200"].Spec.Spec.Content, "image/png")
	assert.NotContains(t, logo.Responses.Spec.Response["200"].Spec.Spec.Content, "application/json")
	assert.Len(t, logo.Responses.Spec.Response["500"].Spec.Spec.Content, 1)
}

func TestSwaggerHandler(t *testing.T) {
	app := api.App{}
	for _, path := range []string{api.SwaggerDocPath, api.SwaggerPath + "/index.html", api.SwaggerPath + "/swagger-ui-bundle.js"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			app.GetSwaggerHandler(response, httptest.NewRequest(http.MethodGet, path, nil), nil)
			assert.Equal(t, http.StatusOK, response.Code)
			if path == api.SwaggerDocPath {
				assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
				assert.JSONEq(t, string(docs.Document), response.Body.String())
			}
		})
	}
}
