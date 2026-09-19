package v1

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

func TestOrderBySpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    orderBySpec
		orderBy string
		wantOK  bool
	}{
		{"empty is allowed", modelOrderBy, "", true},
		{"model standard field", modelOrderBy, "NAME", true},
		{"model recommended", modelOrderBy, "RECOMMENDED", true},
		{"model accuracy alias", modelOrderBy, "ACCURACY", true},
		{"model property", modelOrderBy, "provider.string_value", true},
		{"model artifact property", modelOrderBy, "artifacts.accuracy.double_value", true},
		{"model artifact property with dashes", modelOrderBy, "artifacts.ttft-p90.double_value", true},
		{"model unknown field", modelOrderBy, "BOGUS", false},
		{"model lower-case field is not honoured", modelOrderBy, "name", false},
		{"model unknown value column", modelOrderBy, "accuracy.float_value", false},
		{"model empty property name", modelOrderBy, ".double_value", false},
		{"model four segments", modelOrderBy, "artifacts.a.b.double_value", false},
		{"model injection-shaped", modelOrderBy, "id; DROP TABLE x", false},
		{"enum field", enumOrderBy, "CREATE_TIME", true},
		{"enum accepts every declared value", enumOrderBy, "RECOMMENDED", true},
		{"enum rejects property", enumOrderBy, "provider.string_value", false},
		{"enum rejects accuracy alias", enumOrderBy, "ACCURACY", false},
		{"enum rejects unknown", enumOrderBy, "BOGUS", false},
		{"case-insensitive accepts lower case", caseInsensitiveOrderBy, "name", true},
		{"case-insensitive rejects unknown", caseInsensitiveOrderBy, "bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.validate(tt.orderBy)
			if tt.wantOK {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, api.ErrBadRequest), "expected ErrBadRequest, got %v", err)
			assert.Contains(t, err.Error(), tt.orderBy)
		})
	}
}

func TestFindModelsOrderByValidation(t *testing.T) {
	svc := newTestServiceWithSources(map[string]*model.CatalogModel{
		"modelA": {Name: "modelA", SourceId: new("source1")},
	})

	for _, tt := range []struct {
		orderBy    string
		wantStatus int
	}{
		{"NAME", http.StatusOK},
		{"provider.string_value", http.StatusOK},
		{"artifacts.accuracy.double_value", http.StatusOK},
		{"ACCURACY", http.StatusOK},
		{"BOGUS", http.StatusBadRequest},
		{"accuracy.float_value", http.StatusBadRequest},
	} {
		t.Run(tt.orderBy, func(t *testing.T) {
			resp, _ := svc.FindModels(context.Background(), 0, "", "", "", "", []string{"source1"}, "", nil, "", "10", model.OrderByField(tt.orderBy), model.SORTORDER_ASC, "")
			assert.Equal(t, tt.wantStatus, resp.Code)
		})
	}
}

func TestFindMCPServersRejectsUnknownOrderBy(t *testing.T) {
	svc := NewMCPCatalogServiceAPIService(&mockMCPProvider{}, nil)

	resp, err := svc.FindMCPServers(context.Background(), "", "", nil, "", "", false, 0, "10", "BOGUS", model.SORTORDER_ASC, "")
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	resp, err = svc.FindMCPServerTools(context.Background(), "server-1", "", "10", "provider.string_value", model.SORTORDER_ASC, "")
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestFindSkillsRejectsUnknownOrderBy(t *testing.T) {
	svc := NewSkillCatalogServiceAPIService(nil, nil)

	resp, err := svc.FindSkills(context.Background(), "", "", nil, nil, "", "10", "BOGUS", model.SORTORDER_ASC, "")
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
