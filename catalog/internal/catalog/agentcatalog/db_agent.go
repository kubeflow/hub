package agentcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	agentservice "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/apiutils"
	"github.com/kubeflow/hub/pkg/api"
)

// ListAgentsParams holds parameters for listing agents.
type ListAgentsParams struct {
	Name          string
	Query         string
	SourceIDs     []string
	FilterQuery   string
	PageSize      int32
	OrderBy       openapi.OrderByField
	SortOrder     openapi.SortOrder
	NextPageToken *string
}

// DBAgentCatalog provides database-backed agent catalog operations.
type DBAgentCatalog struct {
	agentRepo                       models.AgentRepository
	agentTemplateArtifactRepo       models.AgentTemplateArtifactRepository
	propertyOptionsRepository       sharedmodels.PropertyOptionsRepository
	sources                         *AgentSourceCollection
}

func NewDBAgentCatalog(services Services, sources *AgentSourceCollection) *DBAgentCatalog {
	return &DBAgentCatalog{
		agentRepo:                       services.AgentRepository,
		agentTemplateArtifactRepo:       services.AgentTemplateArtifactRepository,
		propertyOptionsRepository:       services.PropertyOptionsRepository,
		sources:                         sources,
	}
}

func (d *DBAgentCatalog) GetFilterOptions(ctx context.Context) (*openapi.FilterOptionsList, error) {
	agentTypeID := d.agentRepo.GetTypeID()

	contextProperties, err := d.propertyOptionsRepository.List(sharedmodels.ContextPropertyOptionType, agentTypeID)
	if err != nil {
		return nil, err
	}

	options := make(map[string]openapi.FilterOption, len(contextProperties))

	for _, prop := range contextProperties {
		switch prop.Name {
		case "source_id", "logo", "repositoryUrl", "readme", "artifacts", "env":
			continue
		}

		option := basecatalog.DbPropToAPIOption(prop)
		if option != nil {
			options[prop.FullName("")] = *option
		}
	}

	var namedQueriesPtr *map[string]map[string]openapi.FieldFilter
	if d.sources != nil {
		namedQueriesPtr = basecatalog.ConvertNamedQueries(d.sources.GetNamedQueries(), options)
	}

	return &openapi.FilterOptionsList{
		Filters:      &options,
		NamedQueries: namedQueriesPtr,
	}, nil
}

func (d *DBAgentCatalog) ListAgents(ctx context.Context, params ListAgentsParams) (openapi.AgentList, error) {
	listOptions := models.AgentListOptions{
		FilterQuery: &params.FilterQuery,
	}

	if params.Name != "" {
		listOptions.Name = &params.Name
	}

	if params.Query != "" {
		listOptions.Query = &params.Query
	}

	if len(params.SourceIDs) > 0 {
		listOptions.SourceIDs = &params.SourceIDs
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

	agentsList, err := d.agentRepo.List(&listOptions)
	if err != nil {
		return openapi.AgentList{}, err
	}

	apiAgents := make([]openapi.Agent, 0)
	for _, dbAgent := range agentsList.Items {
		apiAgent := mapDBAgentToAPI(dbAgent)
		apiAgents = append(apiAgents, apiAgent)
	}

	result := openapi.AgentList{
		Items:    apiAgents,
		Size:     int32(len(apiAgents)),
		PageSize: params.PageSize,
	}
	if agentsList.NextPageToken != "" {
		result.NextPageToken = agentsList.NextPageToken
	}

	return result, nil
}

func (d *DBAgentCatalog) GetAgent(ctx context.Context, agentID string) (*openapi.Agent, error) {
	id, err := apiutils.ValidateIDAsInt32(agentID, "agent")
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID '%s': %w", agentID, api.ErrBadRequest)
	}

	dbAgent, err := d.agentRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, agentservice.ErrAgentNotFound) {
			return nil, fmt.Errorf("agent not found with ID %s: %w", agentID, api.ErrNotFound)
		}
		return nil, err
	}

	apiAgent := mapDBAgentToAPI(dbAgent)
	return &apiAgent, nil
}

func (d *DBAgentCatalog) GetAgentArtifacts(ctx context.Context, agentID string, artifactType []openapi.AgentArtifactTypeQueryParam, pageSize int32, orderBy openapi.OrderByField, sortOrder openapi.SortOrder, nextPageToken *string) (openapi.AgentArtifactList, error) {
	id, err := apiutils.ValidateIDAsInt32(agentID, "agent")
	if err != nil {
		return openapi.AgentArtifactList{}, fmt.Errorf("invalid agent ID '%s': %w", agentID, api.ErrBadRequest)
	}

	_, err = d.GetAgent(ctx, agentID)
	if err != nil {
		return openapi.AgentArtifactList{}, err
	}

	wantTemplates := len(artifactType) == 0
	for _, at := range artifactType {
		if at == openapi.AGENTARTIFACTTYPEQUERYPARAM_TEMPLATE_ARTIFACT {
			wantTemplates = true
		}
	}

	var items []openapi.AgentTemplateArtifact
	var responseNextPageToken string

	if wantTemplates && d.agentTemplateArtifactRepo != nil {
		parentID := id
		ob := strings.ToUpper(string(orderBy))
		so := strings.ToUpper(string(sortOrder))
		listOpts := models.AgentTemplateArtifactListOptions{
			ParentResourceID: &parentID,
		}
		listOpts.Pagination.PageSize = &pageSize
		if ob != "" {
			listOpts.Pagination.OrderBy = &ob
		}
		if so != "" {
			listOpts.Pagination.SortOrder = &so
		}
		if nextPageToken != nil {
			listOpts.Pagination.NextPageToken = nextPageToken
		}

		templateList, err := d.agentTemplateArtifactRepo.List(listOpts)
		if err != nil {
			return openapi.AgentArtifactList{}, err
		}
		for _, tmpl := range templateList.Items {
			items = append(items, mapDBTemplateArtifactToAPI(tmpl))
		}
		if templateList.NextPageToken != "" {
			responseNextPageToken = templateList.NextPageToken
		}
	}

	if items == nil {
		items = []openapi.AgentTemplateArtifact{}
	}

	return openapi.AgentArtifactList{
		Items:         items,
		Size:          int32(len(items)),
		PageSize:      pageSize,
		NextPageToken: responseNextPageToken,
	}, nil
}

func mapDBTemplateArtifactToAPI(dbArtifact models.AgentTemplateArtifact) openapi.AgentTemplateArtifact {
	tmpl := openapi.AgentTemplateArtifact{
		ArtifactType: "template-artifact",
	}

	if dbArtifact.GetID() != nil {
		idStr := strconv.FormatInt(int64(*dbArtifact.GetID()), 10)
		tmpl.Id = &idStr
	}

	attrs := dbArtifact.GetAttributes()
	if attrs != nil {
		if attrs.Name != nil {
			tmpl.Name = attrs.Name
		}
		if attrs.Content != nil {
			tmpl.Content = *attrs.Content
		}
		if attrs.CreateTimeSinceEpoch != nil {
			ts := strconv.FormatInt(*attrs.CreateTimeSinceEpoch, 10)
			tmpl.CreateTimeSinceEpoch = &ts
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			ts := strconv.FormatInt(*attrs.LastUpdateTimeSinceEpoch, 10)
			tmpl.LastUpdateTimeSinceEpoch = &ts
		}
		tmpl.ExternalId = attrs.ExternalID
	}

	return tmpl
}

func displayNameFromStoredName(storedName string) string {
	if _, after, ok := strings.Cut(storedName, ":"); ok {
		return after
	}
	return storedName
}

func mapDBAgentToAPI(dbAgent models.Agent) openapi.Agent {
	res := openapi.Agent{}

	if dbAgent.GetID() != nil {
		idStr := strconv.FormatInt(int64(*dbAgent.GetID()), 10)
		res.Id = &idStr
	}

	attrs := dbAgent.GetAttributes()
	if attrs != nil {
		if attrs.Name != nil {
			res.Name = displayNameFromStoredName(*attrs.Name)
		}
		if attrs.CreateTimeSinceEpoch != nil {
			ts := strconv.FormatInt(*attrs.CreateTimeSinceEpoch, 10)
			res.CreateTimeSinceEpoch = &ts
		}
		if attrs.LastUpdateTimeSinceEpoch != nil {
			ts := strconv.FormatInt(*attrs.LastUpdateTimeSinceEpoch, 10)
			res.LastUpdateTimeSinceEpoch = &ts
		}
		res.ExternalId = attrs.ExternalID
	}

	if dbAgent.GetProperties() != nil {
		for _, prop := range *dbAgent.GetProperties() {
			if prop.StringValue == nil {
				continue
			}
			switch prop.Name {
			case "source_id":
				res.SourceId = prop.StringValue
			case "displayName":
				res.DisplayName = prop.StringValue
			case "description":
				res.Description = prop.StringValue
			case "readme":
				res.Readme = prop.StringValue
			case "framework":
				res.Framework = prop.StringValue
			case "logo":
				res.Logo = prop.StringValue
			case "repositoryUrl":
				res.RepositoryUrl = prop.StringValue
			case "labels":
				var labels []string
				if err := json.Unmarshal([]byte(*prop.StringValue), &labels); err == nil {
					res.Labels = labels
				}
			case "env":
				var envVars []openapi.AgentEnvVar
				if err := json.Unmarshal([]byte(*prop.StringValue), &envVars); err == nil {
					res.Env = envVars
				}
			case "artifacts":
				var artifacts []openapi.AgentImageArtifact
				if err := json.Unmarshal([]byte(*prop.StringValue), &artifacts); err == nil {
					res.Artifacts = artifacts
				}
			}
		}
	}

	if dbAgent.GetCustomProperties() != nil {
		customProps := make(map[string]openapi.MetadataValue)
		for _, prop := range *dbAgent.GetCustomProperties() {
			mv := openapi.MetadataValue{}
			if prop.StringValue != nil {
				mv.MetadataStringValue = openapi.NewMetadataStringValueWithDefaults()
				mv.MetadataStringValue.StringValue = *prop.StringValue
			} else if prop.IntValue != nil {
				mv.MetadataIntValue = openapi.NewMetadataIntValueWithDefaults()
				mv.MetadataIntValue.IntValue = fmt.Sprintf("%d", *prop.IntValue)
			} else if prop.DoubleValue != nil {
				mv.MetadataDoubleValue = openapi.NewMetadataDoubleValueWithDefaults()
				mv.MetadataDoubleValue.DoubleValue = *prop.DoubleValue
			} else if prop.BoolValue != nil {
				mv.MetadataBoolValue = openapi.NewMetadataBoolValueWithDefaults()
				mv.MetadataBoolValue.BoolValue = *prop.BoolValue
			}
			customProps[prop.Name] = mv
		}
		if len(customProps) > 0 {
			res.CustomProperties = customProps
		}
	}

	return res
}
