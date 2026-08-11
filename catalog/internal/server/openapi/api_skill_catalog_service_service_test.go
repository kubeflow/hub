package openapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSkillsRejectsSourceAndSourceLabelTogether(t *testing.T) {
	service := NewSkillCatalogServiceAPIService(nil, nil)

	resp, err := service.FindSkills(context.Background(), "", "", []string{"src-a"}, []string{"official"}, "", "", "", "", "")

	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, err.Error(), "source and sourceLabel cannot be used together")
}
