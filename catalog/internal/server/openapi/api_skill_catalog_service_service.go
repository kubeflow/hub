package openapi

import (
	"context"
	"net/http"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

// SkillCatalogServiceAPIService implements SkillCatalogServiceAPIServicer by
// delegating to the skill catalog DB provider.
type SkillCatalogServiceAPIService struct {
	provider *skillcatalog.DBSkillCatalog
}

var _ SkillCatalogServiceAPIServicer = &SkillCatalogServiceAPIService{}

// NewSkillCatalogServiceAPIService creates a skill catalog API service.
func NewSkillCatalogServiceAPIService(provider *skillcatalog.DBSkillCatalog) SkillCatalogServiceAPIServicer {
	return &SkillCatalogServiceAPIService{provider: provider}
}

// FindSkills lists skills.
func (s *SkillCatalogServiceAPIService) FindSkills(ctx context.Context, name string, q string, source []string, sourceLabel []string, filterQuery string, pageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (ImplResponse, error) {
	pageSizeInt, err := parsePaginationParams(pageSize, nextPageToken)
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	// Clean up an empty source value (comma-split of an absent param yields [""]).
	if len(source) == 1 && source[0] == "" {
		source = nil
	}

	// NOTE: the name, q, and sourceLabel parameters are wired with the query API
	// (SKC-108); they are accepted but not yet applied so the endpoint serves in
	// the meantime.
	params := skillcatalog.ListSkillsParams{
		SourceIDs:     source,
		FilterQuery:   filterQuery,
		PageSize:      pageSizeInt,
		OrderBy:       orderBy,
		SortOrder:     sortOrder,
		NextPageToken: &nextPageToken,
	}

	skills, err := s.provider.ListSkills(ctx, params)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}
	return Response(http.StatusOK, skills), nil
}

// FindSkillsFilterOptions lists the fields and values usable in filterQuery.
func (s *SkillCatalogServiceAPIService) FindSkillsFilterOptions(ctx context.Context) (ImplResponse, error) {
	options, err := s.provider.GetFilterOptions(ctx)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}
	return Response(http.StatusOK, options), nil
}

// GetSkill gets a skill by ID.
func (s *SkillCatalogServiceAPIService) GetSkill(ctx context.Context, id string) (ImplResponse, error) {
	skill, err := s.provider.GetSkill(ctx, id)
	if err != nil {
		return ErrorResponse(api.ErrToStatus(err), err), err
	}
	return Response(http.StatusOK, skill), nil
}
