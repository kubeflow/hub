package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubeflow/hub/ui/bff/internal/constants"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

type (
	AgentListEnvelope              Envelope[*models.AgentList, None]
	AgentFilterOptionsListEnvelope Envelope[*models.FilterOptionsList, None]
	AgentEnvelope                  Envelope[*models.Agent, None]
	AgentArtifactListEnvelope      Envelope[*models.AgentArtifactList, None]
)

// GetAllAgentsHandler lists catalog agents.
//
//	@Summary		List agents
//
//	@Description	Lists agents across configured catalog sources, including their frameworks, deployment environment, image artifacts, and Markdown documentation. Use q for keyword search and filterQuery for structured filtering. Not available in Kubeflow integration mode.
//	@Description	filterQuery supports SQL-like comparisons, LIKE/ILIKE, IN, AND/OR, parentheses, and typed custom-property access. Examples: framework = 'langgraph'; name ILIKE '%research%'. Only the first supplied value of each query parameter is forwarded.
//
//	@Tags			AgentCatalogService
//	@ID				findAgents
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string										true	"Catalog namespace"
//
//	@Param			pageSize		query		int											false	"Maximum number of agents in each page"	minimum(1)												default(10)	example(100)
//	@Param			orderBy			query		string										false	"Field to order by"						Enums(CREATE_TIME,LAST_UPDATE_TIME,ID,NAME)	example(NAME)
//	@Param			sortOrder		query		string										false	"Sort direction; defaults to ascending"	Enums(ASC,DESC)											default(ASC)	example(DESC)
//	@Param			nextPageToken	query		string										false	"Opaque token from the previous response; reuse it to retrieve the next page"
//	@Param			q				query		string										false	"Free-form keyword search"										example(research)
//	@Param			source			query		string										false	"Catalog source ID; only the first supplied value is forwarded"	example(community)
//	@Param			sourceLabel		query		string										false	"Label associated with a catalog source"						example(validated)
//	@Param			filterQuery		query		string										false	"SQL-like expression over agent fields and custom properties"	example(framework='langgraph')
//	@Param 			name query string false "Filter by agent name using SQL LIKE pattern matching" example(%research%)
//	@Success		200				{object}	AgentListEnvelope	"Page of catalog agents"
//
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		404				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/agent_catalog/agents [get]
func (app *App) GetAllAgentsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	agents, err := app.repositories.ModelCatalogClient.GetAllAgents(client, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	agentList := AgentListEnvelope{
		Data: agents,
	}

	err = app.WriteJSON(w, http.StatusOK, agentList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetAgentsFiltersHandler lists agent filter options.
//
//	@Summary		List agent filter options
//	@Description	Returns filterable agent fields, allowed values or numeric ranges, and named query templates for constructing filterQuery. Not available in Kubeflow integration mode.
//	@Tags			AgentCatalogService
//	@ID				getAgentsFilterOptions
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Success		200			{object}	AgentFilterOptionsListEnvelope
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/agent_catalog/agents_filter_options [get]
func (app *App) GetAgentsFiltersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	agentFilterOptions, err := app.repositories.ModelCatalogClient.GetAgentsFilter(client)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	agentFilterOptionsList := AgentFilterOptionsListEnvelope{
		Data: agentFilterOptions,
	}

	err = app.WriteJSON(w, http.StatusOK, agentFilterOptionsList, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetAgentHandler returns a catalog agent.
//
//	@Summary		Get an agent
//
//	@Description	Returns the selected agent's framework, Markdown documentation, deployment environment variables, OCI images, and source repository information. Not available in Kubeflow integration mode.
//
//	@Tags			AgentCatalogService
//	@ID				getAgent
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string							true	"Catalog namespace"
//	@Param			agent_id	path		string							true	"Agent ID"
//
//	@Success		200			{object}	AgentEnvelope	"Agent metadata and deployment information"
//
//	@Failure		400			{object}	ErrorEnvelope
//	@Failure		404			{object}	ErrorEnvelope
//	@Failure		500			{object}	ErrorEnvelope
//	@Router			/api/v1/agent_catalog/agents/{agent_id} [get]
func (app *App) GetAgentHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	agentId := ps.ByName(AgentId)

	if agentId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("agent_id is required"))
		return
	}

	agent, err := app.repositories.ModelCatalogClient.GetAgent(client, agentId)
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	agentEnvelope := AgentEnvelope{
		Data: agent,
	}

	err = app.WriteJSON(w, http.StatusOK, agentEnvelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// GetAgentArtifactsHandler lists artifacts for a catalog agent.
//
//	@Summary		List artifacts for an agent
//
//	@Description	Lists deployment artifacts for an agent. Template artifacts contain the full agent.yaml specification as a JSON-encoded string in content; image artifacts carry a URI. Not available in Kubeflow integration mode.
//
//	@Tags			AgentCatalogService
//	@ID				getAgentArtifacts
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string												true	"Catalog namespace"
//	@Param			agent_id		path		string												true	"Agent ID"
//	@Param			pageSize		query		int													false	"Maximum number of results"
//	@Param			orderBy			query		string												false	"Field to order by"
//	@Param			sortOrder		query		string												false	"Sort direction"	Enums(ASC,DESC)
//	@Param			nextPageToken	query		string												false	"Pagination token"
//	@Param artifactType query string false "Agent artifact category" Enums(template-artifact)
//
//	@Success		200				{object}	AgentArtifactListEnvelope	"Page of agent deployment artifacts"
//
//	@Failure		400				{object}	ErrorEnvelope
//	@Failure		404				{object}	ErrorEnvelope
//	@Failure		500				{object}	ErrorEnvelope
//	@Router			/api/v1/agent_catalog/agents/{agent_id}/artifacts [get]
func (app *App) GetAgentArtifactsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	client, ok := r.Context().Value(constants.ModelCatalogHttpClientKey).(httpclient.HTTPClientInterface)

	if !ok {
		app.serverErrorResponse(w, r, errors.New("catalog REST client not found"))
		return
	}

	agentId := ps.ByName(AgentId)

	if agentId == "" {
		app.badRequestResponse(w, r, fmt.Errorf("agent_id is required"))
		return
	}

	artifacts, err := app.repositories.ModelCatalogClient.GetAgentArtifacts(client, agentId, r.URL.Query())
	if err != nil {
		var httpErr *httpclient.HTTPError
		if errors.As(err, &httpErr) {
			app.errorResponse(w, r, httpErr)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	artifactListEnvelope := AgentArtifactListEnvelope{
		Data: artifacts,
	}

	err = app.WriteJSON(w, http.StatusOK, artifactListEnvelope, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
