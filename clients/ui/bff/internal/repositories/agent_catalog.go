package repositories

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

const agentPath = "/agents"
const agentFilterOptionPath = "/agents/filter_options"

type AgentCatalogInterface interface {
	GetAllAgents(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.AgentList, error)
	GetAgentsFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error)
	GetAgent(client httpclient.HTTPClientInterface, agentId string) (*models.Agent, error)
}

type AgentCatalog struct {
	AgentCatalogInterface
}

func (a *AgentCatalog) GetAllAgents(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.AgentList, error) {
	responseData, err := client.GET(UrlWithPageParams(agentPath, pageValues))

	if err != nil {
		return nil, fmt.Errorf("error fetching agents list: %w", err)
	}

	var agents models.AgentList

	if err := json.Unmarshal(responseData, &agents); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &agents, nil
}

func (a *AgentCatalog) GetAgentsFilter(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	responseData, err := client.GET(agentFilterOptionPath)

	if err != nil {
		return nil, fmt.Errorf("error fetching agent filter options: %w", err)
	}

	var filters models.FilterOptionsList

	if err := json.Unmarshal(responseData, &filters); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}
	return &filters, nil
}

func (a *AgentCatalog) GetAgent(client httpclient.HTTPClientInterface, agentId string) (*models.Agent, error) {
	path, err := url.JoinPath(agentPath, agentId)

	if err != nil {
		return nil, err
	}

	responseData, err := client.GET(path)

	if err != nil {
		return nil, fmt.Errorf("error fetching agent: %w", err)
	}

	var agent models.Agent

	if err := json.Unmarshal(responseData, &agent); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &agent, nil
}
