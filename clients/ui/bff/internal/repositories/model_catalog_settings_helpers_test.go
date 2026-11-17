package repositories

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestCatalogSourceCreateDecoding(t *testing.T) {
	for _, raw := range []string{
		`{"id":"yaml_source","name":"YAML","type":"yaml","yaml":"models: []","enabled":false,"labels":["custom"],"includedModels":["org/*"],"excludedModels":[],"yamlCatalogPath":"models.yaml","allowedOrganization":"ignored"}`,
		`{"id":"hf_source","name":"HF","type":"hf","allowedOrganization":"org","apiKey":"test-token","yaml":"ignored","hasConfiguredApiKey":false,"isDefault":false}`,
		`{"id":"source","name":"Unknown","type":"unsupported"}`,
	} {
		var expected models.CatalogSourceConfigPayload
		require.NoError(t, json.Unmarshal([]byte(raw), &expected))
		actual, err := models.DecodeCatalogSourceCreate([]byte(raw))
		require.NoError(t, err)
		assert.Equal(t, expected, actual, "creation types must preserve existing payload fields")
	}
	for _, raw := range []string{`{`, `{"type":false}`, `{"type":"yaml","yaml":true}`, `{"type":"hf","allowedOrganization":false}`} {
		_, err := models.DecodeCatalogSourceCreate([]byte(raw))
		assert.Error(t, err)
	}
}

func TestConfiguredApiKeyParsingAndMerging(t *testing.T) {
	defaultYAML := `catalogs:
  - id: hf_source
    name: Hugging Face
    type: hf
    properties:
      apiKey: catalog-hf-source-apikey
  - id: yaml_source
    name: YAML
    type: yaml
    properties:
      yamlCatalogPath: models.yaml
`
	defaultList, err := ParseCatalogYaml(defaultYAML, true)
	require.NoError(t, err)
	require.True(t, *defaultList[0].HasConfiguredApiKey)
	require.Nil(t, defaultList[1].HasConfiguredApiKey)
	require.True(t, *FindCatalogSourceById(defaultYAML, "hf_source", true).HasConfiguredApiKey)

	for _, tc := range []struct {
		name       string
		properties string
		wantParsed *bool
		wantMerged bool
	}{
		{"sparse override inherits default", "", nil, true},
		{"empty override clears default", "    properties:\n      apiKey: ''\n", boolPtr(false), false},
		{"nonempty override replaces default", "    properties:\n      apiKey: other-secret\n", boolPtr(true), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			userYAML := "catalogs:\n  - id: hf_source\n    enabled: false\n" + tc.properties
			userList, err := ParseCatalogYaml(userYAML, false)
			require.NoError(t, err)
			assert.Equal(t, tc.wantParsed, userList[0].HasConfiguredApiKey)
			userSingle := FindCatalogSourceById(userYAML, "hf_source", false)
			require.NotNil(t, userSingle)
			assert.Equal(t, tc.wantParsed, userSingle.HasConfiguredApiKey)

			for _, merged := range []models.CatalogSourceConfig{
				mergeCatalogSourceConfigs(defaultList[0], userList[0]),
				mergeCatalogSourceConfigs(*FindCatalogSourceById(defaultYAML, "hf_source", true), *userSingle),
			} {
				stripHuggingFaceApiKeyForAPI(&merged)
				require.NotNil(t, merged.HasConfiguredApiKey)
				assert.Equal(t, tc.wantMerged, *merged.HasConfiguredApiKey)
				assert.Nil(t, merged.ApiKey)
				encoded, err := json.Marshal(merged)
				require.NoError(t, err)
				assert.NotContains(t, string(encoded), "catalog-hf-source-apikey")
				assert.NotContains(t, string(encoded), "other-secret")
				assert.NotContains(t, string(encoded), `"apiKey"`)
			}
		})
	}

	for _, yamlText := range []string{
		"catalogs:\n  - id: hf_source\n    type: hf\n",
		"catalogs:\n  - id: hf_source\n    type: hf\n    properties:\n      apiKey: ''\n",
	} {
		list, err := ParseCatalogYaml(yamlText, false)
		require.NoError(t, err)
		stripHuggingFaceApiKeyForAPI(&list[0])
		require.NotNil(t, list[0].HasConfiguredApiKey)
		assert.False(t, *list[0].HasConfiguredApiKey)
		individual := FindCatalogSourceById(yamlText, "hf_source", false)
		stripHuggingFaceApiKeyForAPI(individual)
		require.NotNil(t, individual.HasConfiguredApiKey)
		assert.False(t, *individual.HasConfiguredApiKey)
	}

	yamlSource := defaultList[1]
	stripHuggingFaceApiKeyForAPI(&yamlSource)
	encoded, err := json.Marshal(yamlSource)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "hasConfiguredApiKey")
}

func TestConfiguredApiKeyMetadataIsNotPersisted(t *testing.T) {
	payload := models.CatalogSourceConfigPayload{
		Id: "hf_source", Name: "Hugging Face", Type: CatalogTypeHuggingFace,
		HasConfiguredApiKey: boolPtr(true),
	}
	entry := ConvertSourceConfigToYamlEntry(payload, "", "")
	encoded, err := yaml.Marshal(entry)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "hasConfiguredApiKey")

	updated, err := UpdateCatalogSourceInYAML(
		"catalogs:\n  - id: hf_source\n    name: Hugging Face\n    type: hf\n",
		"hf_source", payload, "", "", false,
	)
	require.NoError(t, err)
	assert.NotContains(t, updated, "hasConfiguredApiKey")
}

func TestIsHuggingFaceApiKeySecretName(t *testing.T) {
	assert.True(t, isHuggingFaceApiKeySecretName("catalog-my-source-apikey"))
	assert.False(t, isHuggingFaceApiKeySecretName("hf_abc123"))
	assert.False(t, isHuggingFaceApiKeySecretName(""))
}

func TestIsRawHuggingFaceApiKey(t *testing.T) {
	assert.True(t, isRawHuggingFaceApiKey("hf_abc123"))
	assert.False(t, isRawHuggingFaceApiKey("catalog-my-source-apikey"))
	assert.False(t, isRawHuggingFaceApiKey("hugging-face-source-secret"))
}

func TestClassifyHuggingFaceApiKeyValue(t *testing.T) {
	raw, secret := classifyHuggingFaceApiKeyValue("hf_abc123")
	assert.Equal(t, "hf_abc123", raw)
	assert.Empty(t, secret)

	raw, secret = classifyHuggingFaceApiKeyValue("catalog-my-source-apikey")
	assert.Empty(t, raw)
	assert.Equal(t, "catalog-my-source-apikey", secret)
}

func TestHuggingFaceApiKeyFromSecret(t *testing.T) {
	assert.Empty(t, huggingFaceApiKeyFromSecret(nil))

	secret := &corev1.Secret{
		StringData: map[string]string{ApiKey: "hf_from_string_data"},
	}
	assert.Equal(t, "hf_from_string_data", huggingFaceApiKeyFromSecret(secret))

	secret = &corev1.Secret{
		Data: map[string][]byte{ApiKey: []byte("hf_from_data")},
	}
	assert.Equal(t, "hf_from_data", huggingFaceApiKeyFromSecret(secret))
}

func TestHuggingFaceSecretNameForCatalogId(t *testing.T) {
	assert.Equal(t, "catalog-my-source-apikey", huggingFaceSecretNameForCatalogId("my_source"))
}

func TestHfSourceLabelValue(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
		// validLabel is false only for the documented degenerate case (an ID
		// that normalizes to all underscores), where hfSourceLabelValue falls
		// back to a value that is not a valid Kubernetes label value.
		validLabel bool
	}{
		{"lowercase with underscore", "my_source", "MY_SOURCE", true},
		{"already valid", "abc_123", "ABC_123", true},
		{"digits", "abc123", "ABC123", true},
		{"leading underscore", "_test", "TEST", true},
		{"trailing underscore", "my_source_", "MY_SOURCE", true},
		{"leading and trailing underscores", "_my_source_", "MY_SOURCE", true},
		{"all underscores", "___", "___", false},
		{"longer than 63 chars", strings.Repeat("a", 70), strings.Repeat("A", 63), true},
		{"truncation lands on underscore", strings.Repeat("a", 62) + "_bbb", strings.Repeat("A", 62), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hfSourceLabelValue(tt.id)
			assert.Equal(t, tt.want, got)
			errs := validation.IsValidLabelValue(got)
			if tt.validLabel {
				assert.Empty(t, errs, "expected %q to be a valid label value", got)
			} else {
				assert.NotEmpty(t, errs, "expected %q to be an invalid label value", got)
			}
		})
	}
}

func TestValidateCatalogId(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{"valid with underscore", "my_source", nil},
		{"valid alphanumeric with underscore", "abc_123", nil},
		{"empty", "", ErrCatalogSourceIdRequired},
		{"uppercase rejected by regex", "MySource", nil},
		{"hyphen rejected by regex", "my-source", nil},
		{"all underscores", "___", ErrCatalogIdInvalid},
		{"single underscore", "_", ErrCatalogIdInvalid},
		{"too long", strings.Repeat("a", 239), ErrCatalogIDTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCatalogId(tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else if tt.name == "uppercase rejected by regex" || tt.name == "hyphen rejected by regex" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
