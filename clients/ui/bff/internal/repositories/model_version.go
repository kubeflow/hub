package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
)

const modelVersionPath = "/model_versions"
const artifactsByModelVersionPath = "/artifacts"

type ModelVersionInterface interface {
	GetAllModelVersions(client httpclient.HTTPClientInterface) (*openapi.ModelVersionList, error)
	GetModelVersion(client httpclient.HTTPClientInterface, id string) (*openapi.ModelVersion, error)
	CreateModelVersion(client httpclient.HTTPClientInterface, jsonData []byte) (*openapi.ModelVersion, error)
	UpdateModelVersion(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelVersion, error)
	GetModelArtifactsByModelVersion(client httpclient.HTTPClientInterface, id string, pageValues url.Values) (*openapi.ModelArtifactList, error)
	CreateModelArtifactByModelVersion(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelArtifact, error)
}

type ModelVersion struct {
	ModelVersionInterface
}

func (v ModelVersion) GetAllModelVersions(client httpclient.HTTPClientInterface) (*openapi.ModelVersionList, error) {
	response, err := client.GET(modelVersionPath)

	if err != nil {
		return nil, fmt.Errorf("error fetching model versions: %w", err)
	}

	var models openapi.ModelVersionList
	if err := json.Unmarshal(response, &models); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &models, nil
}

func (v ModelVersion) GetModelVersion(client httpclient.HTTPClientInterface, id string) (*openapi.ModelVersion, error) {
	path, err := url.JoinPath(modelVersionPath, id)
	if err != nil {
		return nil, err
	}

	response, err := client.GET(path)

	if err != nil {
		return nil, fmt.Errorf("error fetching model version: %w", err)
	}

	var model openapi.ModelVersion
	if err := json.Unmarshal(response, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (v ModelVersion) CreateModelVersion(client httpclient.HTTPClientInterface, jsonData []byte) (*openapi.ModelVersion, error) {
	responseData, err := client.POST(modelVersionPath, bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, fmt.Errorf("error posting registered model: %w", err)
	}

	var model openapi.ModelVersion
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (v ModelVersion) UpdateModelVersion(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelVersion, error) {
	path, err := url.JoinPath(modelVersionPath, id)

	if err != nil {
		return nil, err
	}

	responseData, err := client.PATCH(path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error patching ModelVersion: %w", err)
	}

	var model openapi.ModelVersion
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (v ModelVersion) GetModelArtifactsByModelVersion(client httpclient.HTTPClientInterface, id string, pageValues url.Values) (*openapi.ModelArtifactList, error) {
	path, err := url.JoinPath(modelVersionPath, id, artifactsByModelVersionPath)

	if err != nil {
		return nil, err
	}

	responseData, err := client.GET(UrlWithPageParams(path, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching model version artifacts: %w", err)
	}

	var model openapi.ModelArtifactList
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}

func (v ModelVersion) CreateModelArtifactByModelVersion(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapi.ModelArtifact, error) {
	path, err := url.JoinPath(modelVersionPath, id, artifactsByModelVersionPath)
	if err != nil {
		return nil, err
	}

	responseData, err := client.POST(path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error posting model artifact: %w", err)
	}

	var model openapi.ModelArtifact
	if err := json.Unmarshal(responseData, &model); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &model, nil
}
