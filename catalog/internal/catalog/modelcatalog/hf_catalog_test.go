package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	catalogmodels "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	apimodels "github.com/kubeflow/hub/catalog/pkg/openapi"
	models "github.com/kubeflow/hub/internal/platform/db/entity"
)

func TestPopulateFromHFInfo(t *testing.T) {
	tests := []struct {
		name               string
		hfInfo             *hfModelInfo
		sourceId           string
		originalModelName  string
		expectedName       string
		expectedExternalID *string
		expectedSourceID   *string
		expectedProvider   *string
		expectedLicense    *string
		expectedLibrary    *string
		hasReadme          bool
		hasDescription     bool
		hasTasks           bool
		hasCustomProps     bool
		expectedModelType  *string
	}{
		{
			name: "complete model info",
			hfInfo: &hfModelInfo{
				ID:          "test-org/test-model",
				Author:      "test-author",
				Sha:         "abc123",
				CreatedAt:   "2023-01-01T00:00:00Z",
				UpdatedAt:   "2023-01-02T00:00:00Z",
				Downloads:   1000,
				Tags:        []string{"license:mit", "transformers", "pytorch"},
				PipelineTag: "text-generation",
				Task:        "text-generation",
				LibraryName: "transformers",
				Config: &hfConfig{
					Architectures: []string{"GPT2LMHeadModel"},
					ModelType:     "gpt2",
				},
				CardData: &hfCard{
					Data: map[string]any{
						"description": "A test model description",
					},
				},
			},
			sourceId:          "test-source-id",
			originalModelName: "test-org/test-model",
			expectedName:      "test-org/test-model",
			expectedProvider:  new("test-author"),
			expectedLicense:   new("MIT"),
			expectedLibrary:   new("transformers"),
			hasTasks:          true,
			hasCustomProps:    true,
			hasReadme:         false, // No README fetching in unit tests
			expectedModelType: new("generative"),
		},
		{
			name: "model with ModelID fallback",
			hfInfo: &hfModelInfo{
				ModelID: "fallback-model-id",
				Author:  "another-author",
			},
			sourceId:          "source-2",
			originalModelName: "original-name",
			expectedName:      "fallback-model-id",
			expectedProvider:  new("another-author"),
			expectedModelType: new("unknown"),
		},
		{
			name: "model with original name fallback",
			hfInfo: &hfModelInfo{
				Author: "author-3",
			},
			sourceId:          "source-3",
			originalModelName: "fallback-original-name",
			expectedName:      "fallback-original-name",
			expectedProvider:  new("author-3"),
			expectedModelType: new("unknown"),
		},
		{
			name: "model with license in tags",
			hfInfo: &hfModelInfo{
				ID:   "test/licensed-model",
				Tags: []string{"license:apache-2.0", "other-tag"},
			},
			sourceId:          "source-4",
			originalModelName: "test/licensed-model",
			expectedName:      "test/licensed-model",
			expectedLicense:   new("Apache 2.0"),
			hasCustomProps:    true,
			expectedModelType: new("unknown"),
		},
		{
			name: "model with tasks",
			hfInfo: &hfModelInfo{
				ID:          "test/task-model",
				Task:        "text-classification",
				PipelineTag: "sentiment-analysis",
			},
			sourceId:          "source-5",
			originalModelName: "test/task-model",
			expectedName:      "test/task-model",
			hasTasks:          true,
			hasCustomProps:    true,
			expectedModelType: new("predictive"),
		},
		{
			name: "model with description in cardData",
			hfInfo: &hfModelInfo{
				ID: "test/desc-model",
				CardData: &hfCard{
					Data: map[string]any{
						"description": "This is a test description",
					},
				},
			},
			sourceId:          "source-6",
			originalModelName: "test/desc-model",
			expectedName:      "test/desc-model",
			hasDescription:    true,
			expectedModelType: new("unknown"),
		},
		{
			name: "minimal model info",
			hfInfo: &hfModelInfo{
				ID: "minimal/model",
			},
			sourceId:          "source-7",
			originalModelName: "minimal/model",
			expectedName:      "minimal/model",
			hasCustomProps:    true,
			expectedModelType: new("unknown"),
		},
		{
			name: "model with generative task - should classify as generative",
			hfInfo: &hfModelInfo{
				ID:          "test/generative-model",
				Task:        "text-generation",
				PipelineTag: "text-generation",
			},
			sourceId:          "source-8",
			originalModelName: "test/generative-model",
			expectedName:      "test/generative-model",
			hasTasks:          true,
			hasCustomProps:    true,
			expectedModelType: new("generative"),
		},
		{
			name: "model with predictive task - should classify as predictive",
			hfInfo: &hfModelInfo{
				ID:          "test/predictive-model",
				Task:        "text-classification",
				PipelineTag: "tag1",
			},
			sourceId:          "source-9",
			originalModelName: "test/predictive-model",
			expectedName:      "test/predictive-model",
			hasTasks:          true,
			hasCustomProps:    true,
			expectedModelType: new("predictive"),
		},
		{
			name: "model with both generative and predictive tasks - should prioritize generative",
			hfInfo: &hfModelInfo{
				ID:          "test/mixed-model",
				Task:        "text-generation",
				PipelineTag: "text-classification",
			},
			sourceId:          "source-10",
			originalModelName: "test/mixed-model",
			expectedName:      "test/mixed-model",
			hasTasks:          true,
			hasCustomProps:    true,
			expectedModelType: new("generative"),
		},
		{
			name: "model with unknown task - should classify as unknown",
			hfInfo: &hfModelInfo{
				ID:          "test/unknown-model",
				Task:        "unknown-task-type",
				PipelineTag: "another-unknown-task",
			},
			sourceId:          "source-11",
			originalModelName: "test/unknown-model",
			expectedName:      "test/unknown-model",
			hasTasks:          true,
			hasCustomProps:    true,
			expectedModelType: new("unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock provider with HTTP client to avoid nil pointer
			// Note: README fetching will fail, but that's expected in unit tests
			provider := &hfModelProvider{
				sourceId: tt.sourceId,
				client:   &http.Client{},
			}

			// Create hfModel and populate it
			hfm := &hfModel{}
			ctx := context.Background()
			hfm.populateFromHFInfo(ctx, provider, tt.hfInfo, tt.sourceId, tt.originalModelName)

			// Verify name
			if hfm.Name != tt.expectedName {
				t.Errorf("Name = %v, want %v", hfm.Name, tt.expectedName)
			}

			// Verify ExternalID
			if tt.expectedExternalID != nil {
				if hfm.ExternalId == nil || *hfm.ExternalId != *tt.expectedExternalID {
					t.Errorf("ExternalId = %v, want %v", hfm.ExternalId, tt.expectedExternalID)
				}
			} else if tt.hfInfo.ID != "" {
				// If hfInfo has ID, ExternalId should be set
				if hfm.ExternalId == nil || *hfm.ExternalId != tt.hfInfo.ID {
					t.Errorf("ExternalId = %v, want %v", hfm.ExternalId, tt.hfInfo.ID)
				}
			}

			// Verify SourceID
			if tt.expectedSourceID != nil {
				if hfm.SourceId == nil || *hfm.SourceId != *tt.expectedSourceID {
					t.Errorf("SourceId = %v, want %v", hfm.SourceId, tt.expectedSourceID)
				}
			} else if tt.sourceId != "" {
				if hfm.SourceId == nil || *hfm.SourceId != tt.sourceId {
					t.Errorf("SourceId = %v, want %v", hfm.SourceId, tt.sourceId)
				}
			}

			// Verify Provider
			if tt.expectedProvider != nil {
				if hfm.Provider == nil || *hfm.Provider != *tt.expectedProvider {
					t.Errorf("Provider = %v, want %v", hfm.Provider, tt.expectedProvider)
				}
			}

			// Verify License
			if tt.expectedLicense != nil {
				if hfm.License == nil || *hfm.License != *tt.expectedLicense {
					t.Errorf("License = %v, want %v", *hfm.License, *tt.expectedLicense)
				}
			}

			// Verify LibraryName
			if tt.expectedLibrary != nil {
				if hfm.LibraryName == nil || *hfm.LibraryName != *tt.expectedLibrary {
					t.Errorf("LibraryName = %v, want %v", hfm.LibraryName, tt.expectedLibrary)
				}
			}

			// Verify Tasks
			if tt.hasTasks {
				if len(hfm.Tasks) == 0 {
					t.Error("Expected tasks to be set, but got empty slice")
				}
			}
			if !tt.hasTasks {
				if len(hfm.Tasks) != 0 {
					t.Error("Expected tasks to be empty, but got non-empty slice")
				}
			}

			// Verify Description
			if tt.hasDescription {
				if hfm.Description == nil {
					t.Error("Expected description to be set, but got nil")
				}
			}

			// Verify CustomProperties
			if tt.hasCustomProps {
				customProps := hfm.GetCustomProperties()
				if len(customProps) == 0 {
					t.Error("Expected custom properties to be set, but got nil or empty")
				}
				// Verify model_type
				if tt.expectedModelType != nil {
					if modelTypeVal, ok := customProps["model_type"]; !ok {
						t.Error("Expected model_type in custom properties")
					} else if modelTypeVal.MetadataStringValue == nil {
						t.Error("model_type should be a string value")
					} else if modelTypeVal.MetadataStringValue.StringValue != *tt.expectedModelType {
						t.Errorf("Expected model_type to be '%s', but got '%s'", *tt.expectedModelType, modelTypeVal.MetadataStringValue.StringValue)
					}
				}
			}

			// Verify timestamps if present
			if tt.hfInfo.CreatedAt != "" {
				if hfm.CreateTimeSinceEpoch == nil {
					t.Error("Expected CreateTimeSinceEpoch to be set")
				}
			}
			if tt.hfInfo.UpdatedAt != "" {
				if hfm.LastUpdateTimeSinceEpoch == nil {
					t.Error("Expected LastUpdateTimeSinceEpoch to be set")
				}
			}
		})
	}
}

func TestConvertHFModelToRecord(t *testing.T) {
	tests := []struct {
		name              string
		hfInfo            *hfModelInfo
		originalModelName string
		sourceId          string
		verifyFunc        func(t *testing.T, record ModelProviderRecord)
	}{
		{
			name: "complete model conversion",
			hfInfo: &hfModelInfo{
				ID:          "test-org/complete-model",
				Author:      "test-author",
				CreatedAt:   "2023-01-01T00:00:00Z",
				UpdatedAt:   "2023-01-02T00:00:00Z",
				Tags:        []string{"license:mit"},
				LibraryName: "transformers",
				Task:        "text-generation",
				CardData: &hfCard{
					Data: map[string]any{
						"description": "A complete test model",
					},
				},
			},
			originalModelName: "test-org/complete-model",
			sourceId:          "test-source",
			verifyFunc: func(t *testing.T, record ModelProviderRecord) {
				if record.Model == nil {
					t.Fatal("Model should not be nil")
				}
				attrs := record.Model.GetAttributes()
				if attrs == nil {
					t.Fatal("Attributes should not be nil")
				}
				if attrs.Name == nil || *attrs.Name != "test-org/complete-model" {
					t.Errorf("Name = %v, want 'test-org/complete-model'", attrs.Name)
				}
				if attrs.ExternalID == nil || *attrs.ExternalID != "test-org/complete-model" {
					t.Errorf("ExternalID = %v, want 'test-org/complete-model'", attrs.ExternalID)
				}
				if attrs.CreateTimeSinceEpoch == nil {
					t.Error("CreateTimeSinceEpoch should be set")
				}
				if attrs.LastUpdateTimeSinceEpoch == nil {
					t.Error("LastUpdateTimeSinceEpoch should be set")
				}
				if record.Model.GetProperties() == nil || len(*record.Model.GetProperties()) == 0 {
					t.Error("Properties should be set")
				}
				// Should have one artifact with hf:// URI
				if len(record.Artifacts) != 1 {
					t.Errorf("Expected 1 artifact with hf:// URI, got %d", len(record.Artifacts))
				}
				if len(record.Artifacts) == 1 {
					artifact := record.Artifacts[0]
					if artifact.CatalogModelArtifact == nil {
						t.Error("CatalogModelArtifact should not be nil")
					}
					if artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes().URI == nil {
						t.Error("Artifact URI should not be nil")
					} else if !strings.HasPrefix(*artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes().URI, "hf://") {
						t.Errorf("Artifact URI should start with hf://, got %s", *artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes().URI)
					}
				}
			},
		},
		{
			name: "minimal model conversion",
			hfInfo: &hfModelInfo{
				ID: "minimal/model",
			},
			originalModelName: "minimal/model",
			sourceId:          "source-1",
			verifyFunc: func(t *testing.T, record ModelProviderRecord) {
				if record.Model == nil {
					t.Fatal("Model should not be nil")
				}
				attrs := record.Model.GetAttributes()
				if attrs == nil || attrs.Name == nil {
					t.Fatal("Attributes and Name should not be nil")
				}
				if *attrs.Name != "minimal/model" {
					t.Errorf("Name = %v, want 'minimal/model'", attrs.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &hfModelProvider{
				sourceId: tt.sourceId,
			}
			ctx := context.Background()
			record := provider.convertHFModelToRecord(ctx, tt.hfInfo, tt.originalModelName)
			tt.verifyFunc(t, record)
		})
	}
}

func TestConvertHFModelProperties(t *testing.T) {
	tests := []struct {
		name         string
		catalogModel *apimodels.CatalogModel
		wantProps    bool
		wantCustom   bool
		verifyFunc   func(t *testing.T, props []models.Properties, customProps []models.Properties)
	}{
		{
			name: "model with all properties",
			catalogModel: &apimodels.CatalogModel{
				Name:        "test-model",
				Description: new("Test description"),
				Readme:      new("# Test README"),
				Provider:    new("test-provider"),
				License:     new("MIT License"),
				LibraryName: new("transformers"),
				SourceId:    new("source-1"),
				Tasks:       []string{"text-generation"},
			},
			wantProps:  true,
			wantCustom: false,
			verifyFunc: func(t *testing.T, props []models.Properties, customProps []models.Properties) {
				if len(props) == 0 {
					t.Error("Expected properties to be set")
				}
			},
		},
		{
			name: "model with custom properties",
			catalogModel: func() *apimodels.CatalogModel {
				model := &apimodels.CatalogModel{
					Name: "test-model",
				}
				customProps := map[string]apimodels.MetadataValue{
					"hf_tags": {
						MetadataStringValue: &apimodels.MetadataStringValue{
							StringValue: `["tag1","tag2"]`,
						},
					},
				}
				model.SetCustomProperties(customProps)
				return model
			}(),
			wantProps:  false,
			wantCustom: true,
			verifyFunc: func(t *testing.T, props []models.Properties, customProps []models.Properties) {
				if len(customProps) == 0 {
					t.Error("Expected custom properties to be set")
				}
			},
		},
		{
			name: "model with minimal properties",
			catalogModel: &apimodels.CatalogModel{
				Name: "minimal-model",
			},
			wantProps:  false,
			wantCustom: false,
			verifyFunc: func(t *testing.T, props []models.Properties, customProps []models.Properties) {
				if len(props) != 0 {
					t.Errorf("Expected no properties, got %d", len(props))
				}
				if len(customProps) != 0 {
					t.Errorf("Expected no custom properties, got %d", len(customProps))
				}
			},
		},
		{
			name: "model with tasks",
			catalogModel: &apimodels.CatalogModel{
				Name:  "task-model",
				Tasks: []string{"classification", "generation"},
			},
			wantProps: true,
			verifyFunc: func(t *testing.T, props []models.Properties, customProps []models.Properties) {
				// Should have tasks property
				foundTasks := false
				for _, prop := range props {
					if prop.Name == "tasks" {
						foundTasks = true
						break
					}
				}
				if !foundTasks {
					t.Error("Expected tasks property to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props, customProps := convertHFModelProperties(tt.catalogModel)
			if (len(props) > 0) != tt.wantProps {
				t.Errorf("Properties presence = %v, want %v", len(props) > 0, tt.wantProps)
			}
			if (len(customProps) > 0) != tt.wantCustom {
				t.Errorf("Custom properties presence = %v, want %v", len(customProps) > 0, tt.wantCustom)
			}
			if tt.verifyFunc != nil {
				tt.verifyFunc(t, props, customProps)
			}
		})
	}
}

func TestHFModelProviderWithModelFilter(t *testing.T) {
	tests := []struct {
		name           string
		includedModels []string
		excludedModels []string
		modelName      string
		wantAllowed    bool
		description    string
	}{
		{
			name:           "model matches included pattern",
			includedModels: []string{"ibm-granite/*"},
			excludedModels: nil,
			modelName:      "ibm-granite/granite-4.0-h-small",
			wantAllowed:    true,
			description:    "Model matching included pattern should be allowed",
		},
		{
			name:           "model does not match included pattern",
			includedModels: []string{"ibm-granite/*"},
			excludedModels: nil,
			modelName:      "meta-llama/Llama-3.2-1B",
			wantAllowed:    false,
			description:    "Model not matching included pattern should be excluded",
		},
		{
			name:           "model matches excluded pattern",
			includedModels: []string{"ibm-granite/*"},
			excludedModels: []string{"*-beta"},
			modelName:      "ibm-granite/granite-4.0-h-beta",
			wantAllowed:    false,
			description:    "Model matching excluded pattern should be excluded even if it matches included",
		},
		{
			name:           "model matches included but not excluded",
			includedModels: []string{"ibm-granite/*"},
			excludedModels: []string{"*-beta"},
			modelName:      "ibm-granite/granite-4.0-h-small",
			wantAllowed:    true,
			description:    "Model matching included but not excluded should be allowed",
		},
		{
			name:           "case insensitive matching",
			includedModels: []string{"IBM-Granite/*"},
			excludedModels: nil,
			modelName:      "ibm-granite/granite-4.0-h-small",
			wantAllowed:    true,
			description:    "Filtering should be case-insensitive",
		},
		{
			name:           "no included patterns allows all",
			includedModels: nil,
			excludedModels: []string{"*-beta"},
			modelName:      "test/model",
			wantAllowed:    true,
			description:    "No included patterns means all models are allowed (unless excluded)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a filter with the test patterns
			filter, err := NewModelFilter(tt.includedModels, tt.excludedModels)
			if err != nil {
				t.Fatalf("NewModelFilter() error = %v, want nil", err)
			}

			// Create a provider with the filter
			provider := &hfModelProvider{
				filter: filter,
			}

			// Test that the filter works correctly
			got := provider.filter.Allows(tt.modelName)
			if got != tt.wantAllowed {
				t.Errorf("ModelFilter.Allows(%q) = %v, want %v. %s", tt.modelName, got, tt.wantAllowed, tt.description)
			}
		})
	}
}

func TestPopulateFromHFInfoWithCustomProperties(t *testing.T) {
	hfInfo := &hfModelInfo{
		ID:        "test/custom-props-model",
		Sha:       "sha123",
		Downloads: 5000,
		Tags:      []string{"tag1", "tag2", "license:apache-2.0"},
		Config: &hfConfig{
			Architectures: []string{"BertModel", "BertForSequenceClassification"},
			ModelType:     "bert",
		},
	}

	provider := &hfModelProvider{
		sourceId: "test-source",
	}

	hfm := &hfModel{}
	ctx := context.Background()
	hfm.populateFromHFInfo(ctx, provider, hfInfo, "test-source", "test/custom-props-model")

	customProps := hfm.GetCustomProperties()
	if customProps == nil {
		t.Fatal("Custom properties should not be nil")
	}

	// Verify hf_tags
	if tagsVal, ok := customProps["hf_tags"]; !ok {
		t.Error("Expected hf_tags in custom properties")
	} else if tagsVal.MetadataStringValue == nil {
		t.Error("hf_tags should be a string value")
	}

	// Verify hf_architectures
	if archVal, ok := customProps["hf_architectures"]; !ok {
		t.Error("Expected hf_architectures in custom properties")
	} else if archVal.MetadataStringValue == nil {
		t.Error("hf_architectures should be a string value")
	}

	// Verify hf_model_type
	if modelTypeVal, ok := customProps["hf_model_type"]; !ok {
		t.Error("Expected hf_model_type in custom properties")
	} else if modelTypeVal.MetadataStringValue == nil || modelTypeVal.MetadataStringValue.StringValue != "bert" {
		t.Errorf("hf_model_type = %v, want 'bert'", modelTypeVal.MetadataStringValue)
	}
}

func TestParseModelPattern(t *testing.T) {
	tests := []struct {
		pattern      string
		expectedType PatternType
		expectedOrg  string
		expectedPfx  string
	}{
		// Exact patterns
		{"meta-llama/Llama-2-7b-chat", PatternExact, "", ""},
		{"gpt2", PatternExact, "", ""},
		{"openai-community/gpt2", PatternExact, "", ""},

		// Org/* patterns
		{"ibm-granite/*", PatternOrgAll, "ibm-granite", ""},
		{"meta-llama/*", PatternOrgAll, "meta-llama", ""},
		{"openai/*", PatternOrgAll, "openai", ""},

		// Org/prefix* patterns
		{"meta-llama/Llama-2-*", PatternOrgPrefix, "meta-llama", "Llama-2-"},
		{"ibm-granite/granite-3*", PatternOrgPrefix, "ibm-granite", "granite-3"},
		{"mistralai/Mistral-*", PatternOrgPrefix, "mistralai", "Mistral-"},

		// Invalid patterns
		{"*", PatternInvalid, "", ""},
		{"*/*", PatternInvalid, "", ""},
		{"*/something", PatternInvalid, "", ""},
		{"*/prefix*", PatternInvalid, "", ""},
		{"foo*/bar-*", PatternInvalid, "", ""},
		{"foo*", PatternInvalid, "", ""},
		{"foo*/", PatternInvalid, "", ""},
		{"foo*bar/", PatternInvalid, "", ""},
		{"/foo", PatternInvalid, "", ""},
		{"foo/", PatternInvalid, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			pType, org, prefix := parseModelPattern(tt.pattern)
			assert.Equal(t, tt.expectedType, pType, "pattern type mismatch")
			assert.Equal(t, tt.expectedOrg, org, "org mismatch")
			assert.Equal(t, tt.expectedPfx, prefix, "prefix mismatch")
		})
	}
}

func TestParseNextCursor(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "valid next link",
			header:   `<https://huggingface.co/api/models?author=ibm-granite&limit=100&cursor=abc123>; rel="next"`,
			expected: "abc123",
		},
		{
			name:     "next link with other params",
			header:   `<https://huggingface.co/api/models?author=ibm-granite&cursor=xyz789&limit=100>; rel="next"`,
			expected: "xyz789",
		},
		{
			name:     "multiple links",
			header:   `<https://example.com/first>; rel="first", <https://huggingface.co/api/models?cursor=page2>; rel="next"`,
			expected: "page2",
		},
		{
			name:     "no next link",
			header:   `<https://example.com/first>; rel="first"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNextCursor(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListModelsByAuthor(t *testing.T) {
	// Setup mock HF server
	mux := http.NewServeMux()
	callCount := 0

	// Mock /api/whoami-v2 for credential validation
	mux.HandleFunc("/api/whoami-v2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "test-user"})
	})

	// Mock /api/models list endpoint with pagination
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		author := r.URL.Query().Get("author")
		cursor := r.URL.Query().Get("cursor")
		search := r.URL.Query().Get("search")

		if author == "test-org" {
			switch cursor {
			case "":
				// First page - return 100 items to simulate full page (triggers pagination)
				models := make([]map[string]any, 100)
				for i := range 100 {
					models[i] = map[string]any{"id": fmt.Sprintf("test-org/model-%d", i+1)}
				}
				// Add Link header for next page
				w.Header().Set("Link", `<https://huggingface.co/api/models?author=test-org&cursor=page2>; rel="next"`)
				_ = json.NewEncoder(w).Encode(models)
			case "page2":
				// Second page (last) - return fewer than 100 to indicate end
				models := []map[string]any{
					{"id": "test-org/model-101"},
					{"id": "test-org/model-102"},
				}
				_ = json.NewEncoder(w).Encode(models)
			}
		} else if author == "search-org" && search != "" {
			// Search results
			models := []map[string]any{
				{"id": "search-org/" + search + "-match1"},
				{"id": "search-org/" + search + "-match2"},
				{"id": "search-org/other-model"}, // Should be filtered out
			}
			_ = json.NewEncoder(w).Encode(models)
		} else {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("HF_API_KEY", "test-api-key")

	t.Run("lists all models from org with pagination", func(t *testing.T) {
		callCount = 0
		config := &PreviewConfig{
			Type:           "hf",
			Properties:     map[string]any{},
			IncludedModels: []string{"test-org/*"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		models, err := provider.listModelsByAuthor(context.Background(), "test-org", "")
		require.NoError(t, err)

		// Should have 100 from first page + 2 from second page = 102
		assert.Len(t, models, 102)
		assert.Contains(t, models, "test-org/model-1")
		assert.Contains(t, models, "test-org/model-100")
		assert.Contains(t, models, "test-org/model-101")
		assert.Contains(t, models, "test-org/model-102")

		// Should have made 2 API calls (2 pages)
		assert.Equal(t, 2, callCount)
	})

	t.Run("filters by search prefix", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "hf",
			Properties:     map[string]any{},
			IncludedModels: []string{"test-org/*"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		models, err := provider.listModelsByAuthor(context.Background(), "search-org", "prefix")
		require.NoError(t, err)

		// Should only include models starting with "prefix"
		assert.Len(t, models, 2)
		assert.Contains(t, models, "search-org/prefix-match1")
		assert.Contains(t, models, "search-org/prefix-match2")
		// "other-model" should be filtered out
		assert.NotContains(t, models, "search-org/other-model")
	})

	t.Run("respects maxModels limit", func(t *testing.T) {
		callCount = 0
		config := &PreviewConfig{
			Type: "hf",
			Properties: map[string]any{
				"maxModels": 50, // Limit to 50 models
			},
			IncludedModels: []string{"test-org/*"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL
		assert.Equal(t, 50, provider.maxModels)

		models, err := provider.listModelsByAuthor(context.Background(), "test-org", "")
		require.NoError(t, err)

		// Should stop at 50 models (first page has 100, but we limit to 50)
		assert.Len(t, models, 50)

		// Should have only made 1 API call (stopped before second page)
		assert.Equal(t, 1, callCount)
	})

	t.Run("uses default maxModels when not specified", func(t *testing.T) {
		config := &PreviewConfig{
			Type:           "hf",
			Properties:     map[string]any{},
			IncludedModels: []string{"test-org/*"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)

		// Should use default (500)
		assert.Equal(t, 500, provider.maxModels)
	})

	t.Run("maxModels 0 means no limit", func(t *testing.T) {
		callCount = 0
		config := &PreviewConfig{
			Type: "hf",
			Properties: map[string]any{
				"maxModels": 0, // No limit
			},
			IncludedModels: []string{"test-org/*"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL
		assert.Equal(t, 0, provider.maxModels)

		models, err := provider.listModelsByAuthor(context.Background(), "test-org", "")
		require.NoError(t, err)

		// Should get all 102 models (100 from page 1 + 2 from page 2)
		assert.Len(t, models, 102)

		// Should have made 2 API calls
		assert.Equal(t, 2, callCount)
	})
}

func TestFetchModelsForPreviewWithPatterns(t *testing.T) {
	// Setup mock HF server
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami-v2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "test-user"})
	})

	// Mock list API — returns gated field when expand[]=gated is requested
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author")
		if author == "test-org" {
			models := []map[string]any{
				{"id": "test-org/model-a", "gated": false},
				{"id": "test-org/model-b", "gated": "auto"},
			}
			_ = json.NewEncoder(w).Encode(models)
		} else {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	// Mock individual model endpoints
	mux.HandleFunc("/api/models/exact-org/exact-model", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "exact-org/exact-model",
			"gated":    "manual",
			"siblings": []map[string]any{{"rfilename": "README.md"}},
		})
	})
	mux.HandleFunc("/api/models/exact-org/exact-model/auth-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // access granted
	})

	// Mock individual fetch for gated model-b (called during wildcard follow-up)
	mux.HandleFunc("/api/models/test-org/model-b", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "test-org/model-b",
			"gated":    "auto",
			"siblings": []map[string]any{{"rfilename": "config.json"}},
		})
	})
	mux.HandleFunc("/api/models/test-org/model-b/auth-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden) // access NOT granted
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("mixed patterns: org/* and exact", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "hf",
			IncludedModels: []string{
				"test-org/*",            // Should use list API
				"exact-org/exact-model", // Should use direct fetch
			},
			Properties: map[string]any{
				"apiKey": "hf_test-api-key",
			},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		results, err := provider.FetchModelsForPreview(context.Background(), config.IncludedModels)
		require.NoError(t, err)

		assert.Len(t, results, 3)

		// Find results by name
		byName := make(map[string]hfPreviewModelResult)
		for _, r := range results {
			byName[r.Name] = r
		}

		// Wildcard-listed model: public — no gated access info needed
		a := byName["test-org/model-a"]
		assert.Equal(t, "public", a.AccessType)
		assert.Nil(t, a.GatedAccessGranted)

		// Wildcard-listed model: gated_auto — follow-up fetch shows access NOT granted
		b := byName["test-org/model-b"]
		assert.Equal(t, "gated_auto", b.AccessType)
		require.NotNil(t, b.GatedAccessGranted)
		assert.False(t, *b.GatedAccessGranted)

		// Exact model: gated_manual with access granted (has siblings)
		exact := byName["exact-org/exact-model"]
		assert.Equal(t, "gated_manual", exact.AccessType)
		require.NotNil(t, exact.GatedAccessGranted)
		assert.True(t, *exact.GatedAccessGranted)
	})

	t.Run("rejects * wildcard pattern", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "hf",
			IncludedModels: []string{
				"*",
			},
			Properties: map[string]any{},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		_, err = provider.FetchModelsForPreview(context.Background(), config.IncludedModels)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wildcard pattern")
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("rejects */* wildcard pattern", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "hf",
			IncludedModels: []string{
				"*/*",
			},
			Properties: map[string]any{},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		_, err = provider.FetchModelsForPreview(context.Background(), config.IncludedModels)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wildcard pattern")
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("rejects */prefix pattern", func(t *testing.T) {
		config := &PreviewConfig{
			Type: "hf",
			IncludedModels: []string{
				"*/Llama-*",
			},
			Properties: map[string]any{},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		_, err = provider.FetchModelsForPreview(context.Background(), config.IncludedModels)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wildcard pattern")
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("returns error when token is rejected", func(t *testing.T) {
		rejMux := http.NewServeMux()
		rejMux.HandleFunc("/api/whoami-v2", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		rejServer := httptest.NewServer(rejMux)
		defer rejServer.Close()

		config := &PreviewConfig{
			Type:           "hf",
			IncludedModels: []string{"exact-org/exact-model"},
			Properties:     map[string]any{"apiKey": "hf_bad-token"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = rejServer.URL

		_, err = provider.FetchModelsForPreview(context.Background(), config.IncludedModels)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate HuggingFace credentials")
	})
}

func TestPreviewSourceModelsWithHFPatterns(t *testing.T) {
	// Setup mock HF server
	mux := http.NewServeMux()

	mux.HandleFunc("/api/whoami-v2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "test-user"})
	})

	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author")
		if author == "test-org" {
			models := []map[string]any{
				{"id": "test-org/model-stable", "gated": false},
				{"id": "test-org/model-experimental", "gated": false},
				{"id": "test-org/model-draft", "gated": false},
			}
			_ = json.NewEncoder(w).Encode(models)
		} else {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("org/* pattern with excludedModels filter", func(t *testing.T) {
		// Note: We test the filtering logic by calling NewHFPreviewProvider directly
		// rather than PreviewSourceModels, because PreviewSourceModels has SSRF
		// protection that removes custom URLs (which breaks mock server testing).
		// The filtering behavior is the same - we're testing the filter logic here.
		includedModels := []string{"test-org/*"}
		excludedModels := []string{"*-experimental", "*-draft"}

		config := &PreviewConfig{
			Type:           "hf",
			IncludedModels: includedModels,
			ExcludedModels: excludedModels,
			Properties:     map[string]any{"apiKey": "hf_test-api-key"},
		}

		provider, err := NewHFPreviewProvider(config)
		require.NoError(t, err)
		provider.baseURL = server.URL

		results, err := provider.FetchModelsForPreview(context.Background(), includedModels)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Create filter and apply it (same logic as PreviewSourceModels)
		filter, err := NewModelFilter(includedModels, excludedModels)
		require.NoError(t, err)

		var included, excluded []string
		for _, r := range results {
			if filter.Allows(r.Name) {
				included = append(included, r.Name)
			} else {
				excluded = append(excluded, r.Name)
			}
		}

		assert.Len(t, included, 1)
		assert.Contains(t, included, "test-org/model-stable")

		assert.Len(t, excluded, 2)
		assert.Contains(t, excluded, "test-org/model-experimental")
		assert.Contains(t, excluded, "test-org/model-draft")
	})
}

func TestConvertHFModelToRecord_CreatesArtifactWithHFProtocol(t *testing.T) {
	tests := []struct {
		name               string
		hfInfo             *hfModelInfo
		expectedArtifacts  int
		expectedURI        string
		expectedExternalID string
	}{
		{
			name: "creates artifact with hf:// URI for valid model",
			hfInfo: &hfModelInfo{
				ID:        "meta-llama/Llama-2-7b",
				Author:    "meta",
				ModelID:   "meta-llama/Llama-2-7b",
				CreatedAt: "2023-01-01T00:00:00Z",
			},
			expectedArtifacts:  1,
			expectedURI:        "hf://meta-llama/Llama-2-7b",
			expectedExternalID: "meta-llama/Llama-2-7b",
		},
		{
			name: "creates artifact for gated model",
			hfInfo: &hfModelInfo{
				ID:      "ibm-granite/granite-3.0-8b-instruct",
				Author:  "ibm-granite",
				ModelID: "ibm-granite/granite-3.0-8b-instruct",
				Gated:   gatedString("auto"),
			},
			expectedArtifacts:  1,
			expectedURI:        "hf://ibm-granite/granite-3.0-8b-instruct",
			expectedExternalID: "ibm-granite/granite-3.0-8b-instruct",
		},
		{
			name: "no artifact when ExternalId is nil",
			hfInfo: &hfModelInfo{
				ModelID: "test-model",
				Author:  "test-author",
				// ID is empty, so ExternalId will be nil
			},
			expectedArtifacts: 0,
		},
		{
			name: "no artifact when ExternalId is empty string",
			hfInfo: &hfModelInfo{
				ID:      "",
				ModelID: "test-model",
				Author:  "test-author",
			},
			expectedArtifacts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &hfModelProvider{
				client:   &http.Client{},
				sourceId: "test-source",
			}

			ctx := context.Background()
			record := provider.convertHFModelToRecord(ctx, tt.hfInfo, "original-model-name")

			// Check artifact count
			assert.Len(t, record.Artifacts, tt.expectedArtifacts)

			if tt.expectedArtifacts > 0 {
				// Verify artifact structure
				artifact := record.Artifacts[0]
				require.NotNil(t, artifact.CatalogModelArtifact)
				require.NotNil(t, artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes())

				attrs := artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes()

				// Check URI has hf:// prefix
				require.NotNil(t, attrs.URI)
				assert.Equal(t, tt.expectedURI, *attrs.URI)

				// Check artifact type
				require.NotNil(t, attrs.ArtifactType)
				assert.Equal(t, "model-artifact", *attrs.ArtifactType)

				// Check external ID matches model ID
				require.NotNil(t, attrs.ExternalID)
				assert.Equal(t, tt.expectedExternalID, *attrs.ExternalID)

				// Check artifact name is set
				require.NotNil(t, attrs.Name)
				assert.Contains(t, *attrs.Name, "-hf-artifact")
			}
		})
	}
}

func TestConvertHFModelToRecord_ArtifactTimestamps(t *testing.T) {
	hfInfo := &hfModelInfo{
		ID:        "test-org/test-model",
		Author:    "test-author",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-06-01T12:00:00Z",
	}

	provider := &hfModelProvider{
		client:   &http.Client{},
		sourceId: "test-source",
	}

	ctx := context.Background()
	record := provider.convertHFModelToRecord(ctx, hfInfo, "test-org/test-model")

	require.Len(t, record.Artifacts, 1)
	artifact := record.Artifacts[0]
	attrs := artifact.CatalogModelArtifact.(catalogmodels.CatalogModelArtifact).GetAttributes()

	// Verify timestamps are copied from model
	require.NotNil(t, attrs.CreateTimeSinceEpoch)
	require.NotNil(t, attrs.LastUpdateTimeSinceEpoch)

	// Timestamps should match the model's timestamps
	require.NotNil(t, record.Model.GetAttributes().CreateTimeSinceEpoch)
	require.NotNil(t, record.Model.GetAttributes().LastUpdateTimeSinceEpoch)

	assert.Equal(t, *record.Model.GetAttributes().CreateTimeSinceEpoch, *attrs.CreateTimeSinceEpoch)
	assert.Equal(t, *record.Model.GetAttributes().LastUpdateTimeSinceEpoch, *attrs.LastUpdateTimeSinceEpoch)
}

func TestHfModelProvider_Models_WithWildcardPattern(t *testing.T) {
	// Mock server that handles both list API and individual model API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "author=test-org"):
			// Mock response for list API (used by listModelsByAuthor)
			// HF API returns an array of models directly
			models := []map[string]any{
				{"id": "test-org/model-1", "author": "test-org"},
				{"id": "test-org/model-2", "author": "test-org"},
			}
			json.NewEncoder(w).Encode(models)
		case strings.HasPrefix(r.URL.Path, "/api/models/test-org/model-"):
			// Mock response for individual model API (used by fetchModelInfo)
			modelInfo := hfModelInfo{
				ID:     strings.TrimPrefix(r.URL.Path, "/api/models/"),
				Author: "test-org",
			}
			json.NewEncoder(w).Encode(modelInfo)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL:        mockServer.URL,
		includedModels: []string{"test-org/*"},
		filter:         filter,
		client:         &http.Client{},
		syncInterval:   24 * time.Hour,
	}

	// Use a context with cancel to stop the goroutine after receiving records
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := provider.Models(ctx)
	assert.NoError(t, err)

	var records []ModelProviderRecord
	for record := range ch {
		// Skip empty records (batch markers)
		if record.Model == nil {
			// After receiving the empty record marker, we're done with this batch
			cancel()
			continue
		}
		records = append(records, record)
	}

	// Should have expanded wildcard to 2 concrete models
	assert.Len(t, records, 2)
	assert.Contains(t, getModelNames(records), "test-org/model-1")
	assert.Contains(t, getModelNames(records), "test-org/model-2")
}

func getModelNames(records []ModelProviderRecord) []string {
	var names []string
	for _, record := range records {
		if record.Model.GetAttributes() != nil && record.Model.GetAttributes().Name != nil {
			names = append(names, *record.Model.GetAttributes().Name)
		}
	}
	return names
}

func TestHfModelProvider_Models_WithMixedPatterns(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "author=test-org"):
			models := []map[string]any{
				{"id": "test-org/wildcard-model", "author": "test-org"},
			}
			json.NewEncoder(w).Encode(models)
		case strings.Contains(r.URL.Path, "/api/models/exact-model"):
			modelInfo := hfModelInfo{
				ID:     "exact-model",
				Author: "exact-author",
			}
			json.NewEncoder(w).Encode(modelInfo)
		case strings.Contains(r.URL.Path, "/api/models/test-org/wildcard-model"):
			modelInfo := hfModelInfo{
				ID:     "test-org/wildcard-model",
				Author: "test-org",
			}
			json.NewEncoder(w).Encode(modelInfo)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL:        mockServer.URL,
		includedModels: []string{"exact-model", "test-org/*"},
		filter:         filter,
		client:         &http.Client{},
		syncInterval:   24 * time.Hour,
	}

	// Use a context with cancel to stop the goroutine after receiving records
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := provider.Models(ctx)
	assert.NoError(t, err)

	var records []ModelProviderRecord
	for record := range ch {
		// Skip empty records (batch markers)
		if record.Model == nil {
			// After receiving the empty record marker, we're done with this batch
			cancel()
			continue
		}
		records = append(records, record)
	}

	// Should have both exact model and wildcard-expanded model
	assert.Len(t, records, 2)
	modelNames := getModelNames(records)
	assert.Contains(t, modelNames, "exact-model")
	assert.Contains(t, modelNames, "test-org/wildcard-model")
}

func TestHfModelProvider_Models_WithInvalidWildcardPattern(t *testing.T) {
	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL:        "https://huggingface.co",
		includedModels: []string{"*"}, // Invalid global wildcard
		filter:         filter,
		client:         &http.Client{},
	}

	ch, err := provider.Models(context.Background())

	// Should return error for invalid wildcard pattern
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard pattern \"*\" is not supported")
	assert.Nil(t, ch)
}

func TestHfModelProvider_expandModelNames_AllWildcardPatternsFail(t *testing.T) {
	// Test that when ALL wildcard patterns fail, an error is returned
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 401 for unauthorized requests
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer mockServer.Close()

	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL:        mockServer.URL,
		includedModels: []string{"test-org/*"},
		filter:         filter,
		client:         &http.Client{},
		apiKey:         "invalid-key",
	}

	models, err := provider.expandModelNames(context.Background(), []string{"test-org/*"})

	// Should return an error when all wildcard patterns fail (improved error reporting)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no models found")
	assert.Empty(t, models)
}

func TestHfModelProvider_expandModelNames_PartialWildcardFailure(t *testing.T) {
	// Test that when some but not all wildcard patterns fail, operation continues with partial results
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "author=good-org"):
			// Mock successful response for good-org
			models := []map[string]any{
				{"id": "good-org/model-1", "author": "good-org"},
			}
			json.NewEncoder(w).Encode(models)
		case strings.Contains(r.URL.RawQuery, "author=bad-org"):
			// Mock failure for bad-org (unauthorized)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Unauthorized"}`))
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL: mockServer.URL,
		filter:  filter,
		client:  &http.Client{},
	}

	// Test with mixed patterns: one succeeds, one fails
	patterns := []string{"good-org/*", "bad-org/*"}
	models, err := provider.expandModelNames(context.Background(), patterns)

	// Should NOT return error when only some patterns fail
	assert.NoError(t, err)
	// Should return partial results from successful patterns
	assert.Len(t, models, 1)
	assert.Equal(t, "good-org/model-1", models[0])
}

func TestHfModelProvider_expandModelNames_ExactPatternsWithWildcardFailures(t *testing.T) {
	// Test that when wildcard patterns fail but exact patterns exist, operation continues
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// All wildcard requests fail
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer mockServer.Close()

	filter, err := NewModelFilter([]string{}, []string{})
	require.NoError(t, err)

	provider := &hfModelProvider{
		baseURL: mockServer.URL,
		filter:  filter,
		client:  &http.Client{},
	}

	// Test with mixed patterns: exact models and wildcards
	patterns := []string{"exact-org/exact-model", "wildcard-org/*"}
	models, err := provider.expandModelNames(context.Background(), patterns)

	// Should NOT return error when exact patterns exist even if wildcards fail
	assert.NoError(t, err)
	// Should return the exact patterns that don't require API calls
	assert.Len(t, models, 1)
	assert.Equal(t, "exact-org/exact-model", models[0])
}

func TestClassifyModelTypeFromTasks(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []string
		expected ModelType
	}{
		{
			name:     "generative task - text-generation",
			tasks:    []string{"text-generation"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "generative task - summarization",
			tasks:    []string{"summarization"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "generative task - translation",
			tasks:    []string{"translation"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "generative task - text-to-image",
			tasks:    []string{"text-to-image"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "generative task - text-to-speech",
			tasks:    []string{"text-to-speech"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "predictive task - text-classification",
			tasks:    []string{"text-classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "predictive task - image-classification",
			tasks:    []string{"image-classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "predictive task - question-answering",
			tasks:    []string{"question-answering"},
			expected: ModelTypePredictive,
		},
		{
			name:     "predictive task - object-detection",
			tasks:    []string{"object-detection"},
			expected: ModelTypePredictive,
		},
		{
			name:     "predictive task - feature-extraction",
			tasks:    []string{"feature-extraction"},
			expected: ModelTypePredictive,
		},
		{
			name:     "multiple generative tasks",
			tasks:    []string{"text-generation", "summarization"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "multiple predictive tasks",
			tasks:    []string{"text-classification", "image-classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "generative takes priority over predictive",
			tasks:    []string{"text-generation", "text-classification"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "unknown task",
			tasks:    []string{"unknown-task"},
			expected: ModelTypeUnknown,
		},
		{
			name:     "empty tasks",
			tasks:    []string{},
			expected: ModelTypeUnknown,
		},
		{
			name:     "case insensitive - GENERATIVE",
			tasks:    []string{"TEXT-GENERATION"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "case insensitive - Predictive",
			tasks:    []string{"Text-Classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "task with whitespace",
			tasks:    []string{"  text-generation  "},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - summarization",
			tasks:    []string{"summarization"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - translation",
			tasks:    []string{"translation"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - text-to-image",
			tasks:    []string{"text-to-image"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - unconditional-image-generation",
			tasks:    []string{"unconditional-image-generation"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - image-to-image",
			tasks:    []string{"image-to-image"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all generative task types - audio-to-audio",
			tasks:    []string{"audio-to-audio"},
			expected: ModelTypeGenerative,
		},
		{
			name:     "all predictive task types - zero-shot-classification",
			tasks:    []string{"zero-shot-classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - audio-classification",
			tasks:    []string{"audio-classification"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - document-question-answering",
			tasks:    []string{"document-question-answering"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - image-segmentation",
			tasks:    []string{"image-segmentation"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - keypoint-detection",
			tasks:    []string{"keypoint-detection"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - image-feature-extraction",
			tasks:    []string{"image-feature-extraction"},
			expected: ModelTypePredictive,
		},
		{
			name:     "all predictive task types - fill-mask",
			tasks:    []string{"fill-mask"},
			expected: ModelTypePredictive,
		},
		{
			name:     "unknown task with recognized task - should classify based on recognized",
			tasks:    []string{"unknown-task", "text-generation"},
			expected: ModelTypeGenerative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyModelTypeFromTasks(tt.tasks)
			if result != tt.expected {
				t.Errorf("classifyModelTypeFromTasks(%v) = %q, want %q", tt.tasks, result, tt.expected)
			}
		})
	}
}

func TestNewHFPreviewProvider_RejectsCustomURL(t *testing.T) {
	config := &PreviewConfig{
		Type: "hf",
		Properties: map[string]any{
			"url":    "http://attacker.example.com",
			"apiKey": "hf_test123",
		},
		IncludedModels: []string{"test-org/model-1"},
	}

	provider, err := NewHFPreviewProvider(config)
	require.NoError(t, err)
	assert.Equal(t, defaultHuggingFaceURL, provider.baseURL)

	_, exists := config.Properties["url"]
	assert.False(t, exists, "url property should be deleted from config")
}

func TestNewHFPreviewProvider_UsesDirectApiKey(t *testing.T) {
	config := &PreviewConfig{
		Type: "hf",
		Properties: map[string]any{
			"apiKey": "hf_direct_key",
		},
		IncludedModels: []string{"test-org/model-1"},
	}

	provider, err := NewHFPreviewProvider(config)
	require.NoError(t, err)
	assert.Equal(t, "hf_direct_key", provider.apiKey,
		"should use the API key supplied directly in config properties")

	_, exists := config.Properties["apiKey"]
	assert.False(t, exists, "apiKey should be deleted from properties after extraction")
}

func TestNewHFPreviewProvider_IgnoresServerEnvVars(t *testing.T) {
	t.Setenv("HF_API_KEY", "hf_server_key")
	t.Setenv("HF_API_KEY_ORG1", "hf_org1_key")

	config := &PreviewConfig{
		Type:           "hf",
		Properties:     map[string]any{},
		IncludedModels: []string{"test-org/model-1"},
	}

	provider, err := NewHFPreviewProvider(config)
	require.NoError(t, err)
	assert.Empty(t, provider.apiKey,
		"preview should not fall back to server-side env vars — only the caller-provided key is used")
}

func TestNewHFPreviewProvider_IgnoresApiKeyEnvVarProperty(t *testing.T) {
	t.Setenv("HF_API_KEY", "hf_server_key")
	t.Setenv("PGPASSWORD", "db_secret")

	config := &PreviewConfig{
		Type: "hf",
		Properties: map[string]any{
			"apiKeyEnvVar": "PGPASSWORD",
		},
		IncludedModels: []string{"test-org/model-1"},
	}

	provider, err := NewHFPreviewProvider(config)
	require.NoError(t, err)
	assert.Empty(t, provider.apiKey,
		"apiKeyEnvVar property should be ignored for preview — only direct apiKey is accepted")
}

func TestNewHFPreviewProvider_RejectsInvalidKeyPrefix(t *testing.T) {
	config := &PreviewConfig{
		Type: "hf",
		Properties: map[string]any{
			"apiKey": "bad-prefix-key",
		},
		IncludedModels: []string{"test-org/model-1"},
	}

	_, err := NewHFPreviewProvider(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hf_")
}

func TestNewHFPreviewProvider_WorksWithoutApiKey(t *testing.T) {
	config := &PreviewConfig{
		Type:           "hf",
		Properties:     map[string]any{},
		IncludedModels: []string{"test-org/model-1"},
	}

	provider, err := NewHFPreviewProvider(config)
	require.NoError(t, err)
	assert.Empty(t, provider.apiKey,
		"should succeed without an API key — public models are still accessible")
}

// TestNewHFModelProvider_SanitizesSecurityProperties verifies that the full catalog code path
// (newHFModelProvider) removes forbidden properties before use, matching the preview path.
// sanitizeHFProperties is called before p.Models(), so the assertions hold even if Models()
// returns an error (e.g. network failure reaching huggingface.co in a test environment).
func TestNewHFModelProvider_SanitizesSecurityProperties(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := &basecatalog.ModelSource{
		CatalogSource: apimodels.CatalogSource{
			Id:             "test-source",
			IncludedModels: []string{"test-org/test-model"},
		},
		Properties: map[string]any{
			"url":          "http://attacker.example.com",
			"apiKeyEnvVar": "PGPASSWORD",
		},
	}

	ch, _ := newHFModelProvider(ctx, source, "") // ignore error: network may fail in test env
	cancel()
	if ch != nil {
		go func() { //nolint:revive
			for range ch {
			}
		}()
	}

	_, urlExists := source.Properties[urlKey]
	assert.False(t, urlExists, "url property must be removed to prevent SSRF")

	_, envVarExists := source.Properties[apiKeyEnvVarKey]
	assert.False(t, envVarExists, "apiKeyEnvVar property must be removed to prevent env var oracle")
}

func TestEnvVarSuffix(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		want     string
	}{
		{"lowercase with dash", "my-source", "MY_SOURCE"},
		{"mixed case", "MySource", "MYSOURCE"},
		{"dots and slashes", "org.name/sub", "ORG_NAME_SUB"},
		{"already valid", "ABC_123", "ABC_123"},
		{"digits", "abc123", "ABC123"},
		{"empty", "", ""},
		{"leading underscore", "_test", "TEST"},
		{"trailing slash", "org/", "ORG"},
		{"leading and trailing special chars", "-my-source-", "MY_SOURCE"},
		{"all underscores", "___", "___"},
		{"longer than 63 chars", strings.Repeat("a", 70), strings.Repeat("A", 63)},
		{"truncation lands on underscore", strings.Repeat("a", 62) + "_bbb", strings.Repeat("A", 62)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, envVarSuffix(tt.sourceID))
		})
	}
}

func TestResolveHFAPIKey(t *testing.T) {
	t.Run("property env var takes precedence", func(t *testing.T) {
		t.Setenv("HF_API_KEY_ORG1", "hf_org1")
		t.Setenv("HF_API_KEY_MY_SOURCE", "hf_derived")
		t.Setenv("HF_API_KEY", "hf_global")
		assert.Equal(t, "hf_org1", resolveHFAPIKey("my-source", "HF_API_KEY_ORG1"))
	})

	t.Run("falls back to source-specific var", func(t *testing.T) {
		t.Setenv("HF_API_KEY_MY_SOURCE", "hf_derived")
		t.Setenv("HF_API_KEY", "hf_global")
		assert.Equal(t, "hf_derived", resolveHFAPIKey("my-source", ""))
	})

	t.Run("source id is sanitized when building the derived var", func(t *testing.T) {
		t.Setenv("HF_API_KEY_ORG_NAME_SUB", "hf_sanitized")
		t.Setenv("HF_API_KEY", "hf_global")
		assert.Equal(t, "hf_sanitized", resolveHFAPIKey("org.name/sub", ""))
	})

	t.Run("falls back to global var", func(t *testing.T) {
		t.Setenv("HF_API_KEY_MY_SOURCE", "")
		t.Setenv("HF_API_KEY", "hf_global")
		assert.Equal(t, "hf_global", resolveHFAPIKey("my-source", ""))
	})

	t.Run("property var empty falls through to derived", func(t *testing.T) {
		t.Setenv("HF_API_KEY_ORG1", "")
		t.Setenv("HF_API_KEY_MY_SOURCE", "hf_derived")
		t.Setenv("HF_API_KEY", "hf_global")
		assert.Equal(t, "hf_derived", resolveHFAPIKey("my-source", "HF_API_KEY_ORG1"))
	})

	t.Run("returns empty when nothing is set", func(t *testing.T) {
		t.Setenv("HF_API_KEY_MY_SOURCE", "")
		t.Setenv("HF_API_KEY", "")
		assert.Equal(t, "", resolveHFAPIKey("my-source", ""))
	})
}

func TestDeriveHFAccessType(t *testing.T) {
	tests := []struct {
		name     string
		hfInfo   *hfModelInfo
		expected string
	}{
		{
			name:     "public model (not private, not gated)",
			hfInfo:   &hfModelInfo{Private: false, Gated: gatedString("false")},
			expected: "public",
		},
		{
			name:     "public model (empty gated)",
			hfInfo:   &hfModelInfo{Private: false, Gated: gatedString("")},
			expected: "public",
		},
		{
			name:     "private model",
			hfInfo:   &hfModelInfo{Private: true, Gated: gatedString("false")},
			expected: "private",
		},
		{
			name:     "gated auto model",
			hfInfo:   &hfModelInfo{Private: false, Gated: gatedString("auto")},
			expected: "gated_auto",
		},
		{
			name:     "gated manual model",
			hfInfo:   &hfModelInfo{Private: false, Gated: gatedString("manual")},
			expected: "gated_manual",
		},
		{
			name:     "gated boolean true (legacy format)",
			hfInfo:   &hfModelInfo{Private: false, Gated: gatedString("true")},
			expected: "gated_auto",
		},
		{
			name:     "private takes precedence over gated",
			hfInfo:   &hfModelInfo{Private: true, Gated: gatedString("auto")},
			expected: "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveHFAccessType(tt.hfInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPopulateFromHFInfo_AccessTypeProperties(t *testing.T) {
	ctx := context.Background()

	// Mock auth-check endpoint: 200 for granted models, 403 for others.
	grantedModels := map[string]bool{
		"meta-llama/Llama-3-8B": true,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/auth-check") {
			modelID := strings.TrimPrefix(r.URL.Path, "/api/models/")
			modelID = strings.TrimSuffix(modelID, "/auth-check")
			if grantedModels[modelID] {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider := &hfModelProvider{
		sourceId: "test-source",
		client:   &http.Client{},
		baseURL:  server.URL,
		apiKey:   "hf_test-key",
	}

	tests := []struct {
		name                       string
		hfInfo                     *hfModelInfo
		expectedAccessType         string
		expectGatedAccessGranted   bool
		expectedGatedAccessGranted string
	}{
		{
			name: "public model has hf_access_type=public, no hf_gated_access_granted",
			hfInfo: &hfModelInfo{
				ID:      "test-org/public-model",
				Private: false,
				Gated:   gatedString("false"),
			},
			expectedAccessType:       "public",
			expectGatedAccessGranted: false,
		},
		{
			name: "private model has hf_access_type=private, no hf_gated_access_granted",
			hfInfo: &hfModelInfo{
				ID:       "my-org/private-llm",
				Private:  true,
				Gated:    gatedString("false"),
				Siblings: []hfFile{{RFileName: "config.json"}},
			},
			expectedAccessType:       "private",
			expectGatedAccessGranted: false,
		},
		{
			name: "gated auto model with access granted (auth-check 200)",
			hfInfo: &hfModelInfo{
				ID:       "meta-llama/Llama-3-8B",
				Private:  false,
				Gated:    gatedString("auto"),
				Siblings: []hfFile{{RFileName: "config.json"}, {RFileName: "model.safetensors"}},
			},
			expectedAccessType:         "gated_auto",
			expectGatedAccessGranted:   true,
			expectedGatedAccessGranted: "true",
		},
		{
			name: "gated manual model without access (auth-check 403)",
			hfInfo: &hfModelInfo{
				ID:       "org/not-granted-model",
				Private:  false,
				Gated:    gatedString("manual"),
				Siblings: []hfFile{{RFileName: "config.json"}},
			},
			expectedAccessType:         "gated_manual",
			expectGatedAccessGranted:   true,
			expectedGatedAccessGranted: "false",
		},
		{
			name: "gated boolean true without access (auth-check 403)",
			hfInfo: &hfModelInfo{
				ID:       "org/gated-model",
				Private:  false,
				Gated:    gatedString("true"),
				Siblings: []hfFile{{RFileName: "config.json"}},
			},
			expectedAccessType:         "gated_auto",
			expectGatedAccessGranted:   true,
			expectedGatedAccessGranted: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hfm := &hfModel{}
			hfm.populateFromHFInfo(ctx, provider, tt.hfInfo, "test-source", tt.hfInfo.ID)

			customProps := hfm.GetCustomProperties()

			// Check hf_access_type
			accessTypeProp, ok := customProps["hf_access_type"]
			require.True(t, ok, "hf_access_type should be present")
			require.NotNil(t, accessTypeProp.MetadataStringValue)
			assert.Equal(t, tt.expectedAccessType, accessTypeProp.MetadataStringValue.StringValue)

			// Check hf_gated_access_granted
			gatedProp, hasGatedProp := customProps["hf_gated_access_granted"]
			assert.Equal(t, tt.expectGatedAccessGranted, hasGatedProp,
				"hf_gated_access_granted presence mismatch")
			if tt.expectGatedAccessGranted {
				require.NotNil(t, gatedProp.MetadataStringValue)
				assert.Equal(t, tt.expectedGatedAccessGranted,
					gatedProp.MetadataStringValue.StringValue)
			}
		})
	}
}

func TestCheckGatedAccess(t *testing.T) {
	mux := http.NewServeMux()

	// Simulate HF auth-check: 200 for granted, 401 for unauthenticated, 403 for denied.
	mux.HandleFunc("/api/models/org/granted-model/auth-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/models/org/denied-model/auth-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/api/models/org/unauthed-model/auth-check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("returns true when auth-check returns 200", func(t *testing.T) {
		p := &hfModelProvider{client: &http.Client{}, baseURL: server.URL, apiKey: "hf_key"}
		assert.True(t, p.checkGatedAccess(context.Background(), "org/granted-model"))
	})

	t.Run("returns false when auth-check returns 403", func(t *testing.T) {
		p := &hfModelProvider{client: &http.Client{}, baseURL: server.URL, apiKey: "hf_key"}
		assert.False(t, p.checkGatedAccess(context.Background(), "org/denied-model"))
	})

	t.Run("returns false when auth-check returns 401", func(t *testing.T) {
		p := &hfModelProvider{client: &http.Client{}, baseURL: server.URL, apiKey: "hf_key"}
		assert.False(t, p.checkGatedAccess(context.Background(), "org/unauthed-model"))
	})

	t.Run("returns false when no API key is configured", func(t *testing.T) {
		p := &hfModelProvider{client: &http.Client{}, baseURL: server.URL, apiKey: ""}
		assert.False(t, p.checkGatedAccess(context.Background(), "org/granted-model"))
	})
}

func TestSetSourceCredentialStatus(t *testing.T) {
	tests := []struct {
		name              string
		hasApiKey         bool
		authenticated     *bool
		hfUsername        string
		expectHasApiKey   bool
		expectAuthSet     bool
		expectAuthValue   bool
		expectUsername    string
		expectUsernameNil bool
	}{
		{
			name:              "no API key",
			hasApiKey:         false,
			authenticated:     nil,
			hfUsername:        "",
			expectHasApiKey:   false,
			expectAuthSet:     false,
			expectUsernameNil: true,
		},
		{
			name:            "API key with successful auth and username",
			hasApiKey:       true,
			authenticated:   new(true),
			hfUsername:      "jdoe",
			expectHasApiKey: true,
			expectAuthSet:   true,
			expectAuthValue: true,
			expectUsername:  "jdoe",
		},
		{
			name:              "API key with failed auth",
			hasApiKey:         true,
			authenticated:     new(false),
			hfUsername:        "",
			expectHasApiKey:   true,
			expectAuthSet:     true,
			expectAuthValue:   false,
			expectUsernameNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &basecatalog.ModelSource{
				CatalogSource: apimodels.CatalogSource{
					Id:   "test-source",
					Name: "Test Source",
				},
			}

			setSourceCredentialStatus(source, tt.hasApiKey, tt.authenticated, tt.hfUsername)

			assert.Equal(t, tt.expectHasApiKey, source.GetHasApiKey())
			if tt.expectAuthSet {
				assert.True(t, source.HasAuthenticated())
				assert.Equal(t, tt.expectAuthValue, source.GetAuthenticated())
			} else {
				val, isSet := source.GetAuthenticatedOk()
				assert.True(t, isSet, "authenticated should be explicitly set (to nil)")
				assert.Nil(t, val, "authenticated value should be nil (no token)")
			}
			if tt.expectUsernameNil {
				assert.Empty(t, source.GetHfUsername(), "hfUsername should be nil/empty")
			} else {
				assert.Equal(t, tt.expectUsername, source.GetHfUsername())
			}
		})
	}
}
