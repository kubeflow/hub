package openapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

// fakeSkillPreviewer is a stand-in for the skill plugin's previewer, returning
// canned results so the dispatch and response shaping can be tested without git.
type fakeSkillPreviewer struct {
	results []model.AssetPreviewResult
	err     error
}

func (f fakeSkillPreviewer) PreviewSkillSource(_ context.Context, _ []byte) ([]model.AssetPreviewResult, error) {
	return f.results, f.err
}

func skillPreviewService(p SkillSourcePreviewer) ModelCatalogServiceAPIServicer {
	// The preview path uses only the skill previewer; the other dependencies are
	// unused here and left nil.
	return NewModelCatalogServiceAPIService(nil, nil, nil, nil, nil, nil, WithSkillPreviewer(p))
}

func TestPreviewCatalogSource_SkillsDispatch(t *testing.T) {
	fake := fakeSkillPreviewer{results: []model.AssetPreviewResult{
		{Name: "deploy", Included: true},
		{Name: "rollout", Included: true},
		{Name: "scratch-draft", Included: false},
	}}
	service := skillPreviewService(fake)

	configFile := writeTempYAML(t, "config", `
assetType: skills
type: git-skills-plugin
properties:
  repositories:
    - url: https://example.com/a.git
      refs: ["v1.0"]
`)

	resp, err := service.PreviewCatalogSource(context.Background(), configFile, "100", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)

	assetResp, ok := resp.Body.(model.AssetSourcePreviewResponse)
	require.True(t, ok, "expected AssetSourcePreviewResponse, got %T", resp.Body)
	assert.Equal(t, model.CATALOGASSETTYPE_SKILLS, assetResp.AssetType)
	assert.Len(t, assetResp.Items, 3)
	assert.Equal(t, int32(3), assetResp.Summary.TotalAssets)
	assert.Equal(t, int32(2), assetResp.Summary.IncludedAssets)
	assert.Equal(t, int32(1), assetResp.Summary.ExcludedAssets)
}

func TestPreviewCatalogSource_SkillsFilterStatusExcluded(t *testing.T) {
	fake := fakeSkillPreviewer{results: []model.AssetPreviewResult{
		{Name: "deploy", Included: true},
		{Name: "scratch-draft", Included: false},
	}}
	service := skillPreviewService(fake)

	configFile := writeTempYAML(t, "config", "assetType: skills\ntype: git-skills-plugin\nproperties:\n  repositories:\n    - url: https://example.com/a.git\n      refs: [\"v1.0\"]\n")

	resp, err := service.PreviewCatalogSource(context.Background(), configFile, "100", "", "excluded", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Code)

	assetResp := resp.Body.(model.AssetSourcePreviewResponse)
	require.Len(t, assetResp.Items, 1)
	assert.Equal(t, "scratch-draft", assetResp.Items[0].Name)
}

func TestPreviewCatalogSource_SkillsPreviewerUnavailable(t *testing.T) {
	// No WithSkillPreviewer option: an assetType: skills preview should report 501.
	service := NewModelCatalogServiceAPIService(nil, nil, nil, nil, nil, nil)

	configFile := writeTempYAML(t, "config", "assetType: skills\ntype: git-skills-plugin\n")

	resp, err := service.PreviewCatalogSource(context.Background(), configFile, "100", "", "", nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, resp.Code)
}

func TestPreviewCatalogSource_SkillsPreviewError(t *testing.T) {
	service := skillPreviewService(fakeSkillPreviewer{err: errors.New("clone failed")})

	configFile := writeTempYAML(t, "config", "assetType: skills\ntype: git-skills-plugin\n")

	resp, err := service.PreviewCatalogSource(context.Background(), configFile, "100", "", "", nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}
