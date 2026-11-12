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

type AgentListEnvelope Envelope[*models.AgentList, None]
type AgentFilterOptionsListEnvelope Envelope[*models.FilterOptionsList, None]
type AgentEnvelope Envelope[*models.Agent, None]
type AgentArtifactListEnvelope Envelope[*models.AgentArtifactList, None]

// GetAllAgentsHandler lists catalog agents.
//
//	@Summary		List agents
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			AgentCatalogService
//	@ID				findAgents
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string	true	"Catalog namespace"
//	@Param			pageSize		query		int		false	"Maximum number of results"
//	@Param			orderBy			query		string	false	"Field to order by"
//	@Param			sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param			nextPageToken	query		string	false	"Pagination token"
//	@Param			q				query		string	false	"Search query"
//	@Param			source			query		string	false	"Source IDs to search, comma-separated"
//	@Param			filterQuery		query		string	false	"Agent filter expression"
//	@Success		200				{object}	AgentListEnvelope
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
//	@Description	Not available in Kubeflow integration mode.
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
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			AgentCatalogService
//	@ID				getAgent
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace	query		string	true	"Catalog namespace"
//	@Param			agent_id	path		string	true	"Agent ID"
//	@Success		200			{object}	AgentEnvelope
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
//	@Description	Not available in Kubeflow integration mode.
//	@Tags			AgentCatalogService
//	@ID				getAgentArtifacts
//	@Produce		json
//	@Security		KubeflowUserID
//	@Security		BearerAuth
//	@Param			namespace		query		string	true	"Catalog namespace"
//	@Param			agent_id		path		string	true	"Agent ID"
//	@Param			pageSize		query		int		false	"Maximum number of results"
//	@Param			orderBy			query		string	false	"Field to order by"
//	@Param			sortOrder		query		string	false	"Sort direction"	Enums(ASC,DESC)
//	@Param			nextPageToken	query		string	false	"Pagination token"
//	@Success		200				{object}	AgentArtifactListEnvelope
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
