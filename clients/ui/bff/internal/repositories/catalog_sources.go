package repositories

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/models"
)

const sourcesPath = "/sources"
const filterOptionPath = "/models/filter_options"
const labelsPath = "/labels"

type CatalogSourcesInterface interface {
	GetAllCatalogSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogSourceList, error)
	GetCatalogSourceModel(client httpclient.HTTPClientInterface, sourceId string, modelName string) (*models.CatalogModel, error)
	GetCatalogSourceModelArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error)
	GetCatalogFilterOptions(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error)
	GetCatalogModelPerformanceArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error)
	GetCatalogModelSecurityArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error)
	GetCatalogLabels(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogLabelList, error)
}

type CatalogSources struct {
	CatalogSourcesInterface
}

func (a CatalogSources) GetAllCatalogSources(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogSourceList, error) {
	responseData, err := client.GET(UrlWithPageParams(sourcesPath, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching sourcesPath: %w", err)
	}

	var sources models.CatalogSourceList

	if err := json.Unmarshal(responseData, &sources); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &sources, nil
}

func (a CatalogSources) GetCatalogSourceModel(client httpclient.HTTPClientInterface, sourceId string, modelName string) (*models.CatalogModel, error) {
	path, err := url.JoinPath(sourcesPath, sourceId, "models", modelName)

	if err != nil {
		return nil, err
	}
	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching sourcesPath: %w", err)
	}

	var catalogModel models.CatalogModel

	if err := json.Unmarshal(responseData, &catalogModel); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &catalogModel, nil
}

func (a CatalogSources) GetCatalogSourceModelArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	path, err := url.JoinPath(sourcesPath, sourceId, "models", modelName, "artifacts")
	if err != nil {
		return nil, err
	}
	responseData, err := client.GET(UrlWithPageParams(path, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching sourcesPath: %w", err)
	}

	var catalogModelArtifacts models.CatalogModelArtifactList

	if err := json.Unmarshal(responseData, &catalogModelArtifacts); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}
	return &catalogModelArtifacts, nil
}

func (a CatalogSources) GetCatalogFilterOptions(client httpclient.HTTPClientInterface) (*models.FilterOptionsList, error) {
	responseData, err := client.GET(filterOptionPath)

	if err != nil {
		return nil, fmt.Errorf("error fetching filterOptionPath: %w", err)
	}

	var sources models.FilterOptionsList

	if err := json.Unmarshal(responseData, &sources); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &sources, nil
}

func (a CatalogSources) GetCatalogModelPerformanceArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	path, err := url.JoinPath(sourcesPath, sourceId, "models", modelName, "artifacts", "performance")
	if err != nil {
		return nil, err
	}
	responseData, err := client.GET(UrlWithPageParams(path, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching sourcesPath: %w", err)
	}

	var catalogModelArtifacts models.CatalogModelArtifactList

	if err := json.Unmarshal(responseData, &catalogModelArtifacts); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}
	return &catalogModelArtifacts, nil
}

func (a CatalogSources) GetCatalogModelSecurityArtifacts(client httpclient.HTTPClientInterface, sourceId string, modelName string, pageValues url.Values) (*models.CatalogModelArtifactList, error) {
	path, err := url.JoinPath(sourcesPath, sourceId, "models", modelName, "artifacts")
	if err != nil {
		return nil, err
	}

	filteredValues := FilterPageValues(pageValues)
	filteredValues.Set("filterQuery", `metricsType.string_value="security-metrics"`)

	responseData, err := client.GET(UrlWithParams(path, filteredValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching security artifacts: %w", err)
	}

	var catalogModelArtifacts models.CatalogModelArtifactList

	if err := json.Unmarshal(responseData, &catalogModelArtifacts); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}
	return &catalogModelArtifacts, nil
}

func (a CatalogSources) GetCatalogLabels(client httpclient.HTTPClientInterface, pageValues url.Values) (*models.CatalogLabelList, error) {
	responseData, err := client.GET(UrlWithPageParams(labelsPath, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching labels: %w", err)
	}

	var labels models.CatalogLabelList

	if err := json.Unmarshal(responseData, &labels); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &labels, nil
}
