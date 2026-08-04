package skillcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	skillservice "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/service"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
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

// emptyFilterMap is the shared empty filter map returned until per-field
// filters are implemented (SKC-108). The map must not be mutated by callers.
var emptyFilterMap = map[string]openapi.FilterOption{}

// GetFilterOptions returns the filterable fields for skills. Entity-specific
// options are populated once fields are persisted (SKC-108).
func (d *DBSkillCatalog) GetFilterOptions(_ context.Context) (*openapi.FilterOptionsList, error) {
	return &openapi.FilterOptionsList{Filters: &emptyFilterMap}, nil
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

// mapDBSkillToAPI maps a datastore skill entity to the OpenAPI Skill model. The
// entity's Name attribute holds the composite identity; the display name and all
// other fields come from properties written by the loader.
func mapDBSkillToAPI(dbSkill models.Skill) *openapi.Skill {
	res := &openapi.Skill{}

	if id := dbSkill.GetID(); id != nil {
		s := strconv.FormatInt(int64(*id), 10)
		res.Id = &s
	}
	if props := dbSkill.GetProperties(); props != nil {
		for i := range *props {
			p := (*props)[i]
			switch p.Name {
			case propSkillName:
				if p.StringValue != nil {
					res.Name = *p.StringValue
				}
			case propDescription:
				res.Description = p.StringValue
			case propRepository:
				res.Repository = p.StringValue
			case propPath:
				res.Path = p.StringValue
			case propSkillVersion:
				res.Version = p.StringValue
			case propResolvedCommit:
				res.ResolvedCommit = p.StringValue
			case propSourceID:
				res.SourceId = p.StringValue
			case propTrustTier:
				if p.StringValue != nil {
					tt := openapi.SkillTrustTier(*p.StringValue)
					res.TrustTier = &tt
				}
			case propProvider:
				res.Provider = p.StringValue
			case propCategory:
				res.Category = p.StringValue
			case propLicense:
				res.License = p.StringValue
			case propAuthor:
				res.Author = p.StringValue
			case propCompatibility:
				res.Compatibility = p.StringValue
			case propReadme:
				res.Readme = p.StringValue
			case propLabels:
				res.Labels = decodeStringSlice(p.StringValue)
			case propAllowedTools:
				res.AllowedTools = decodeStringSlice(p.StringValue)
			case propSupportingFiles:
				res.SupportingFiles = decodeStringSlice(p.StringValue)
			case propBodyLineCount:
				res.BodyLineCount = p.IntValue
			default:
				// Any property that isn't a pre-configured field is surfaced as a
				// custom property rather than dropped.
				addSkillCustomProperty(res, p)
			}
		}
	}

	// Properties flagged as custom (e.g. SKILL.md frontmatter metadata) are always
	// surfaced as customProperties.
	if custom := dbSkill.GetCustomProperties(); custom != nil {
		for i := range *custom {
			addSkillCustomProperty(res, (*custom)[i])
		}
	}

	return res
}

// addSkillCustomProperty adds a datastore property to the Skill's customProperties.
func addSkillCustomProperty(res *openapi.Skill, prop dbmodels.Properties) {
	if res.CustomProperties == nil {
		res.CustomProperties = map[string]openapi.MetadataValue{}
	}
	res.CustomProperties[prop.Name] = dbPropToMetadataValue(prop)
}

// dbPropToMetadataValue converts a datastore property to an OpenAPI MetadataValue,
// mirroring the model/MCP converters.
func dbPropToMetadataValue(prop dbmodels.Properties) openapi.MetadataValue {
	mv := openapi.MetadataValue{}
	switch {
	case prop.StringValue != nil:
		mv.MetadataStringValue = openapi.NewMetadataStringValueWithDefaults()
		mv.MetadataStringValue.StringValue = *prop.StringValue
	case prop.IntValue != nil:
		mv.MetadataIntValue = openapi.NewMetadataIntValueWithDefaults()
		mv.MetadataIntValue.IntValue = fmt.Sprintf("%d", *prop.IntValue)
	case prop.DoubleValue != nil:
		mv.MetadataDoubleValue = openapi.NewMetadataDoubleValueWithDefaults()
		mv.MetadataDoubleValue.DoubleValue = *prop.DoubleValue
	case prop.BoolValue != nil:
		mv.MetadataBoolValue = openapi.NewMetadataBoolValueWithDefaults()
		mv.MetadataBoolValue.BoolValue = *prop.BoolValue
	}
	return mv
}

// decodeStringSlice decodes a JSON string array stored in a string property.
func decodeStringSlice(v *string) []string {
	if v == nil || *v == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(*v), &out); err != nil {
		// The paired writer (addStringSlice) always stores valid JSON; a malformed
		// value means external tampering/migration. Log and treat as empty rather
		// than failing the response.
		glog.Errorf("skill: ignoring malformed JSON array property: %v", err)
		return nil
	}
	return out
}
