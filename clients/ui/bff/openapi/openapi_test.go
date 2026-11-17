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
	require.NotEmpty(t, doc.Servers)
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
		if !strings.HasPrefix(path, api.ApiPathPrefix) || strings.HasPrefix(path, api.SwaggerPath+"/") {
			return true
		}
		path = pathParam.ReplaceAllString(path, "{$1}")
		expected[strings.ToLower(method.Sel.Name)+" "+path] = true
		return true
	})
	require.NotEmpty(t, expected)
	actual := map[string]bool{}
	for path, pathRef := range doc.Paths.Spec.Paths {
		if !strings.HasPrefix(path, api.ApiPathPrefix) {
			continue
		}
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

	// Response schemas must reference the same envelopes used by the handlers,
	// rather than separate documentation-only model types.
	schemas := doc.Components.Spec.Schemas
	for name := range schemas {
		assert.NotContains(t, name, "Swagger", "schemas must use runtime types")
	}
	for path, envelope := range map[string]string{
		"/api/v1/model_registry":                                                       "ModelRegistryListEnvelope",
		"/api/v1/model_catalog/models":                                                 "CatalogModelListEnvelope",
		"/api/v1/model_catalog/sources/{source_id}/models/{model_name}":                "CatalogModelEnvelope",
		"/api/v1/model_catalog/sources/{source_id}/artifacts/{model_name}":             "catalogModelArtifactsListEnvelope",
		"/api/v1/model_catalog/sources/{source_id}/performance_artifacts/{model_name}": "catalogModelArtifactsListEnvelope",
		"/api/v1/model_catalog/sources/{source_id}/security_artifacts/{model_name}":    "catalogModelArtifactsListEnvelope",
		"/api/v1/agent_catalog/agents":                                                 "AgentListEnvelope",
		"/api/v1/agent_catalog/agents/{agent_id}":                                      "AgentEnvelope",
		"/api/v1/agent_catalog/agents/{agent_id}/artifacts":                            "AgentArtifactListEnvelope",
		"/api/v1/mcp_catalog/mcp_servers":                                              "McpServerListEnvelope",
		"/api/v1/mcp_catalog/mcp_servers/{server_id}":                                  "McpServerEnvelope",
		"/api/v1/mcp_catalog/mcp_servers/{server_id}/tools":                            "McpServerToolsListEnvelope",
	} {
		t.Run(envelope+path, func(t *testing.T) {
			_, ok := schemas["api."+envelope]
			require.True(t, ok, "missing runtime envelope schema: api.%s", envelope)
			response := doc.Paths.Spec.Paths[path].Spec.Spec.Get.Spec.Responses.Spec.Response["200"].Spec.Spec
			assert.Equal(t, "#/components/schemas/api."+envelope, response.Content["application/json"].Spec.Schema.Ref.Ref)
		})
	}
	assert.Contains(t, schemas["api.RegisteredModelEnvelope"].Spec.Properties, "data")
	for _, name := range []string{"openapi.RegisteredModelUpdate", "openapi.ModelVersionUpdate", "models.CatalogSourceConfigPayload", "models.McpCatalogSourceConfigPayload"} {
		assert.Empty(t, schemas[name].Spec.Required, name)
	}
	for name, required := range map[string][]string{
		"models.RegisteredModelCreate":                {"name"},
		"models.ModelVersionCreate":                   {"name", "registeredModelId"},
		"models.ModelVersionCreateForRegisteredModel": {"name"},
		"models.YamlCatalogSourceCreate":              {"id", "name", "type", "yaml"},
		"models.HFCatalogSourceCreate":                {"id", "name", "type", "allowedOrganization"},
		"models.McpCatalogSourceCreate":               {"id", "name", "type", "yaml"},
	} {
		require.Contains(t, schemas, name)
		assert.ElementsMatch(t, required, schemas[name].Spec.Required, name)
	}
	createSource := schemas["api.ModelCatalogSourceCreateEnvelope"].Spec
	assert.Contains(t, createSource.Required, "data")
	require.Len(t, createSource.Properties["data"].Spec.OneOf, 2)
	assert.Equal(t, "#/components/schemas/models.YamlCatalogSourceCreate", createSource.Properties["data"].Spec.OneOf[0].Ref.Ref)
	assert.Equal(t, "#/components/schemas/models.HFCatalogSourceCreate", createSource.Properties["data"].Spec.OneOf[1].Ref.Ref)
	assert.Equal(t, []interface{}{"yaml"}, schemas["models.YamlCatalogSourceCreate"].Spec.Properties["type"].Spec.Enum)
	assert.Equal(t, []interface{}{"hf"}, schemas["models.HFCatalogSourceCreate"].Spec.Properties["type"].Spec.Enum)
	preview := doc.Paths.Spec.Paths["/api/v1/settings/model_catalog/source_preview"].Spec.Spec.Post.Spec
	assert.Equal(t, "#/components/schemas/api.CatalogSourcePreviewEnvelope", preview.Responses.Spec.Response["200"].Spec.Spec.Content["application/json"].Spec.Schema.Ref.Ref)
	previewRequestRef := preview.RequestBody.Spec.Spec.Content["application/json"].Spec.Schema.Ref.Ref
	previewRequestSchema := strings.TrimPrefix(previewRequestRef, "#/components/schemas/")
	_, ok := schemas[previewRequestSchema]
	require.True(t, ok, "missing preview request schema: %s", previewRequestSchema)
	assert.Equal(t, "#/components/schemas/models.CatalogSourcePreviewRequest", schemas[previewRequestSchema].Spec.Properties["data"].Ref.Ref)

	createModel := doc.Paths.Spec.Paths["/api/v1/model_registry/{model_registry_id}/registered_models"].Spec.Spec.Post.Spec
	require.NotNil(t, createModel.RequestBody)
	assert.True(t, createModel.RequestBody.Spec.Spec.Required)
	assert.Equal(t, "#/components/schemas/api.RegisteredModelCreateEnvelope", createModel.RequestBody.Spec.Spec.Content["application/json"].Spec.Schema.Ref.Ref)
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
