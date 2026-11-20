package mocks

import (
	"io"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
	mock.Mock
}

func (c *MockHTTPClient) GetModelRegistryID() string {
	return "model-registry"
}

func (m *MockHTTPClient) GET(url string) ([]byte, error) {
	args := m.Called(url)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockHTTPClient) GETRaw(url string) (*httpclient.RawResponse, error) {
	args := m.Called(url)
	return args.Get(0).(*httpclient.RawResponse), args.Error(1)
}

func (m *MockHTTPClient) POST(url string, body io.Reader) ([]byte, error) {
	args := m.Called(url, body)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockHTTPClient) POSTWithContentType(url string, body io.Reader, contentType string) ([]byte, error) {
	args := m.Called(url, body, contentType)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockHTTPClient) PATCH(url string, body io.Reader) ([]byte, error) {
	args := m.Called(url, body)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockHTTPClient) DELETE(url string) error {
	args := m.Called(url)
	return args.Error(0)
}
