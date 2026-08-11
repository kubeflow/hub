package openapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

// SkillCatalogServiceAPIService implements SkillCatalogServiceAPIServicer by
// delegating to the skill catalog DB provider.
type SkillCatalogServiceAPIService struct {
	provider *skillcatalog.DBSkillCatalog
	sources  *skillcatalog.SkillSourceCollection
}

var _ SkillCatalogServiceAPIServicer = &SkillCatalogServiceAPIService{}

// NewSkillCatalogServiceAPIService creates a skill catalog API service.
func NewSkillCatalogServiceAPIService(provider *skillcatalog.DBSkillCatalog, sources *skillcatalog.SkillSourceCollection) SkillCatalogServiceAPIServicer {
	return &SkillCatalogServiceAPIService{provider: provider, sources: sources}
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
	if len(sourceLabel) == 1 && sourceLabel[0] == "" {
		sourceLabel = nil
	}

	if len(source) > 0 && len(sourceLabel) > 0 {
		err := fmt.Errorf("source and sourceLabel cannot be used together")
		return ErrorResponse(http.StatusBadRequest, err), err
	}

	sourceIDs := source
	if len(sourceIDs) == 0 && len(sourceLabel) > 0 && s.sources != nil {
		matched := s.sources.ByLabel(sourceLabel)
		if len(matched) == 0 {
			return Response(http.StatusOK, model.SkillList{
				Items:    []model.Skill{},
				PageSize: pageSizeInt,
			}), nil
		}
		sourceIDs = make([]string, len(matched))
		for i, src := range matched {
			sourceIDs[i] = src.ID
		}
	}

	params := skillcatalog.ListSkillsParams{
		Name:          name,
		Query:         q,
		SourceIDs:     sourceIDs,
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
