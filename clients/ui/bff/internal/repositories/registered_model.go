package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
)

const registeredModelPath = "/registered_models"
const versionsPath = "/versions"

type RegisteredModelInterface interface {
	GetAllRegisteredModels(client httpclient.HTTPClientInterface, pageValues url.Values) (*openapi.RegisteredModelList, error)
	CreateRegisteredModel(client httpclient.HTTPClientInterface, jsonData []byte) (*openapi.RegisteredModel, error)
	GetRegisteredModel(client httpclient.HTTPClientInterface, id string) (*openapi.RegisteredModel, error)
	UpdateRegisteredModel(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.RegisteredModel, error)
	GetAllModelVersionsForRegisteredModel(client httpclient.HTTPClientInterface, id string, pageValues url.Values) (*openapi.ModelVersionList, error)
	CreateModelVersionForRegisteredModel(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelVersion, error)
}

type RegisteredModel struct {
	RegisteredModelInterface
}

func (m RegisteredModel) GetAllRegisteredModels(client httpclient.HTTPClientInterface, pageValues url.Values) (*openapi.RegisteredModelList, error) {
	responseData, err := client.GET(UrlWithPageParams(registeredModelPath, pageValues))

	if err != nil {
		return nil, fmt.Errorf("error fetching registered models: %w", err)
	}

	var modelList openapi.RegisteredModelList
	if err := json.Unmarshal(responseData, &modelList); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &modelList, nil
}

func (m RegisteredModel) CreateRegisteredModel(client httpclient.HTTPClientInterface, jsonData []byte) (*openapi.RegisteredModel, error) {
	responseData, err := client.POST(registeredModelPath, bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, fmt.Errorf("error posting registered model: %w", err)
	}

	var model openapi.RegisteredModel
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (m RegisteredModel) GetRegisteredModel(client httpclient.HTTPClientInterface, id string) (*openapi.RegisteredModel, error) {
	path, err := url.JoinPath(registeredModelPath, id)
	if err != nil {
		return nil, err
	}
	responseData, err := client.GET(path)

	if err != nil {
		return nil, fmt.Errorf("error fetching registered model: %w", err)
	}

	var model openapi.RegisteredModel
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (m RegisteredModel) UpdateRegisteredModel(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.RegisteredModel, error) {
	path, err := url.JoinPath(registeredModelPath, id)

	if err != nil {
		return nil, err
	}

	responseData, err := client.PATCH(path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error patching registered model: %w", err)
	}

	var model openapi.RegisteredModel
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (m RegisteredModel) GetAllModelVersionsForRegisteredModel(client httpclient.HTTPClientInterface, id string, pageValues url.Values) (*openapi.ModelVersionList, error) {
	path, err := url.JoinPath(registeredModelPath, id, versionsPath)

	if err != nil {
		return nil, err
	}

	responseData, err := client.GET(UrlWithPageParams(path, pageValues))

	if err != nil {
		return nil, fmt.Errorf("error fetching model versions: %w", err)
	}

	var model openapi.ModelVersionList
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (m RegisteredModel) CreateModelVersionForRegisteredModel(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelVersion, error) {
	path, err := url.JoinPath(registeredModelPath, id, versionsPath)

	if err != nil {
		return nil, err
	}

	responseData, err := client.POST(path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error posting model version: %w", err)
	}

	var model openapi.ModelVersion
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}
