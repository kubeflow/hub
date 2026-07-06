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
