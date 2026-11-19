package mocks

import (
	"log/slog"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"

	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
	"github.com/stretchr/testify/mock"
)

type ModelRegistryClientMock struct {
	mock.Mock
}

func NewModelRegistryClient(_ *slog.Logger) (*ModelRegistryClientMock, error) {
	return &ModelRegistryClientMock{}, nil
}

func (m *ModelRegistryClientMock) GetAllRegisteredModels(_ httpclient.HTTPClientInterface, _ url.Values) (*openapiv1.RegisteredModelList, error) {
	mockData := GetRegisteredModelListMock()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) CreateRegisteredModel(_ httpclient.HTTPClientInterface, _ []byte) (*openapiv1.RegisteredModel, error) {
	mockData := GetRegisteredModelMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetRegisteredModel(_ httpclient.HTTPClientInterface, id string) (*openapiv1.RegisteredModel, error) {
	if id == "3" {
		mockData := GetRegisteredModelMocks()[2]
		return &mockData, nil
	}
	if id == "2" {
		mockData := GetRegisteredModelMocks()[1]
		return &mockData, nil
	}
	mockData := GetRegisteredModelMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) UpdateRegisteredModel(_ httpclient.HTTPClientInterface, _ string, _ []byte) (*openapiv1.RegisteredModel, error) {
	mockData := GetRegisteredModelMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetAllModelVersions(_ httpclient.HTTPClientInterface) (*openapiv1.ModelVersionList, error) {
	mockData := GetModelVersionListMock()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetModelVersion(_ httpclient.HTTPClientInterface, id string) (*openapiv1.ModelVersion, error) {
	if id == "4" {
		mockData := GetModelVersionMocks()[3]
		return &mockData, nil
	}

	if id == "3" {
		mockData := GetModelVersionMocks()[2]
		return &mockData, nil
	}

	if id == "2" {
		mockData := GetModelVersionMocks()[1]
		return &mockData, nil
	}

	mockData := GetModelVersionMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) CreateModelVersion(_ httpclient.HTTPClientInterface, _ []byte) (*openapiv1.ModelVersion, error) {
	mockData := GetModelVersionMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) UpdateModelVersion(_ httpclient.HTTPClientInterface, _ string, _ []byte) (*openapiv1.ModelVersion, error) {
	mockData := GetModelVersionMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetAllModelVersionsForRegisteredModel(_ httpclient.HTTPClientInterface, id string, _ url.Values) (*openapiv1.ModelVersionList, error) {
	mockList := GetModelVersionListMock()
	mockData := openapiv1.ModelVersionList{
		Items:         []openapiv1.ModelVersion{},
		NextPageToken: mockList.NextPageToken,
		PageSize:      mockList.PageSize,
		Size:          0,
	}

	for _, mv := range mockList.Items {
		if mv.RegisteredModelId == id {
			mockData.Items = append(mockData.Items, mv)
		}
	}
	mockData.Size = int32(len(mockData.Items))
	return &mockData, nil
}

func (m *ModelRegistryClientMock) CreateModelVersionForRegisteredModel(_ httpclient.HTTPClientInterface, _ string, _ []byte) (*openapiv1.ModelVersion, error) {
	mockData := GetModelVersionMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetModelArtifactsByModelVersion(_ httpclient.HTTPClientInterface, _ string, _ url.Values) (*openapiv1.ModelArtifactList, error) {
	mockData := GetModelArtifactListMock()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) CreateModelArtifactByModelVersion(_ httpclient.HTTPClientInterface, _ string, _ []byte) (*openapiv1.ModelArtifact, error) {
	mockData := GetModelArtifactMocks()[0]
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetAllArtifacts(_ httpclient.HTTPClientInterface, _ url.Values) (*openapiv1.ArtifactList, error) {
	mockData := GenerateMockArtifactList()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) GetArtifact(_ httpclient.HTTPClientInterface, _ string) (*openapiv1.Artifact, error) {
	mockData := GenerateMockArtifact()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) CreateArtifact(_ httpclient.HTTPClientInterface, _ []byte) (*openapiv1.Artifact, error) {
	mockData := GenerateMockArtifact()
	return &mockData, nil
}

func (m *ModelRegistryClientMock) UpdateModelArtifact(_ httpclient.HTTPClientInterface, _ string, _ []byte) (*openapiv1.ModelArtifact, error) {
	mockData := GenerateMockModelArtifact()
	return &mockData, nil
}
