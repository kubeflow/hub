package skillcatalog

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	skillservice "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/service"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/pkg/api"
)

// DBSkillCatalog serves skill catalog reads from the datastore.
type DBSkillCatalog struct {
	skillRepo models.SkillRepository
}

// NewDBSkillCatalog creates a skill catalog provider backed by the datastore.
func NewDBSkillCatalog(services Services) *DBSkillCatalog {
	return &DBSkillCatalog{
		skillRepo: services.SkillRepository,
	}
}

// ListSkillsParams carries the list/query parameters for skills.
// name/q filtering and sourceLabel resolution are added with the query API (SKC-108).
type ListSkillsParams struct {
	SourceIDs     []string
	FilterQuery   string
	PageSize      int32
	OrderBy       openapi.OrderByField
	SortOrder     openapi.SortOrder
	NextPageToken *string
}

// GetFilterOptions returns the filterable fields for skills. Entity-specific
// options are populated once fields are persisted (SKC-108).
func (d *DBSkillCatalog) GetFilterOptions(_ context.Context) (*openapi.FilterOptionsList, error) {
	options := make(map[string]openapi.FilterOption)
	return &openapi.FilterOptionsList{Filters: &options}, nil
}

// ListSkills returns a paginated list of skills.
func (d *DBSkillCatalog) ListSkills(_ context.Context, params ListSkillsParams) (openapi.SkillList, error) {
	listOptions := models.SkillListOptions{}
	if len(params.SourceIDs) > 0 {
		listOptions.SourceIDs = &params.SourceIDs
	}
	if params.FilterQuery != "" {
		listOptions.FilterQuery = &params.FilterQuery
	}

	orderBy := strings.ToUpper(string(params.OrderBy))
	sortOrder := strings.ToUpper(string(params.SortOrder))
	listOptions.Pagination.PageSize = &params.PageSize
	if orderBy != "" {
		listOptions.Pagination.OrderBy = &orderBy
	}
	if sortOrder != "" {
		listOptions.Pagination.SortOrder = &sortOrder
	}
	if params.NextPageToken != nil {
		listOptions.Pagination.NextPageToken = params.NextPageToken
	}

	result, err := d.skillRepo.List(&listOptions)
	if err != nil {
		return openapi.SkillList{}, err
	}

	items := make([]openapi.Skill, 0, len(result.Items))
	for _, dbSkill := range result.Items {
		items = append(items, *mapDBSkillToAPI(dbSkill))
	}

	list := openapi.SkillList{
		Items:    items,
		Size:     int32(len(items)),
		PageSize: params.PageSize,
	}
	if result.NextPageToken != "" {
		list.NextPageToken = result.NextPageToken
	}
	return list, nil
}

// GetSkill returns a single skill by its datastore ID.
func (d *DBSkillCatalog) GetSkill(_ context.Context, id string) (*openapi.Skill, error) {
	skillID, err := apiutils.ValidateIDAsInt32(id, "skill")
	if err != nil {
		return nil, fmt.Errorf("invalid skill ID '%s': %w", id, api.ErrBadRequest)
	}

	dbSkill, err := d.skillRepo.GetByID(skillID)
	if err != nil {
		if errors.Is(err, skillservice.ErrSkillNotFound) {
			return nil, fmt.Errorf("skill not found with ID %s: %w", id, api.ErrNotFound)
		}
		return nil, fmt.Errorf("error getting skill %s: %w", id, err)
	}

	return mapDBSkillToAPI(dbSkill), nil
}

// mapDBSkillToAPI maps a datastore skill entity to the OpenAPI Skill model.
// Only the identity fields persisted so far are mapped; full field propagation
// through the datastore mappings lands with the query API (SKC-108).
func mapDBSkillToAPI(dbSkill models.Skill) *openapi.Skill {
	res := &openapi.Skill{}

	if id := dbSkill.GetID(); id != nil {
		s := strconv.FormatInt(int64(*id), 10)
		res.Id = &s
	}
	if attrs := dbSkill.GetAttributes(); attrs != nil && attrs.Name != nil {
		res.Name = *attrs.Name
	}
	if props := dbSkill.GetProperties(); props != nil {
		for _, prop := range *props {
			if prop.Name == "source_id" {
				res.SourceId = prop.StringValue
			}
		}
	}

	return res
}
