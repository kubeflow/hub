package repositories

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAllAgents_Success(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	responseJSON := `{"size": 2, "pageSize": 10, "nextPageToken": "", "items": [{"id": "1", "name": "agent-1"}, {"id": "2", "name": "agent-2"}]}`
	mockClient.On("GET", "/agents").Return([]byte(responseJSON), nil)

	repo := AgentCatalog{}
	pageValues := url.Values{}
	result, err := repo.GetAllAgents(mockClient, pageValues)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(2), result.Size)
	assert.Equal(t, int32(10), result.PageSize)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "1", result.Items[0].ID)
	assert.Equal(t, "agent-1", result.Items[0].Name)
	mockClient.AssertExpectations(t)
}

func TestGetAllAgents_WithQueryParams(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	responseJSON := `{"size": 1, "pageSize": 10, "nextPageToken": "", "items": [{"id": "1", "name": "code-review-agent"}]}`
	mockClient.On("GET", mock.MatchedBy(func(path string) bool {
		return strings.HasPrefix(path, "/agents?") &&
			strings.Contains(path, "name=code")
	})).Return([]byte(responseJSON), nil)

	repo := AgentCatalog{}
	pageValues := url.Values{}
	pageValues.Set("name", "code")
	result, err := repo.GetAllAgents(mockClient, pageValues)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "code-review-agent", result.Items[0].Name)
	mockClient.AssertExpectations(t)
}

func TestGetAllAgents_ClientError(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	expectedErr := &httpclient.HTTPError{
		StatusCode: 500,
		ErrorResponse: httpclient.ErrorResponse{
			Code:    "500",
			Message: "internal server error",
		},
	}
	mockClient.On("GET", "/agents").Return([]byte(nil), expectedErr)

	repo := AgentCatalog{}
	result, err := repo.GetAllAgents(mockClient, url.Values{})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	mockClient.AssertExpectations(t)
}

func TestGetAllAgents_InvalidJSON(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	mockClient.On("GET", "/agents").Return([]byte("not valid json"), nil)

	repo := AgentCatalog{}
	result, err := repo.GetAllAgents(mockClient, url.Values{})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding")
	mockClient.AssertExpectations(t)
}

func TestGetAgentsFilter_Success(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	responseJSON := `{"filters": {}, "namedQueries": {}}`
	mockClient.On("GET", "/agents/filter_options").Return([]byte(responseJSON), nil)

	repo := AgentCatalog{}
	result, err := repo.GetAgentsFilter(mockClient)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockClient.AssertExpectations(t)
}

func TestGetAgentsFilter_ClientError(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	expectedErr := errors.New("network failure")
	mockClient.On("GET", "/agents/filter_options").Return([]byte(nil), expectedErr)

	repo := AgentCatalog{}
	result, err := repo.GetAgentsFilter(mockClient)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching")
	mockClient.AssertExpectations(t)
}

func TestGetAgent_Success(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	responseJSON := `{"id": "1", "name": "code-review-agent"}`
	mockClient.On("GET", "/agents/1").Return([]byte(responseJSON), nil)

	repo := AgentCatalog{}
	result, err := repo.GetAgent(mockClient, "1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "code-review-agent", result.Name)
	mockClient.AssertExpectations(t)
}

func TestGetAgent_ClientError(t *testing.T) {
	mockClient := &mocks.MockHTTPClient{}
	expectedErr := &httpclient.HTTPError{StatusCode: 404}
	mockClient.On("GET", "/agents/missing").Return([]byte(nil), expectedErr)

	repo := AgentCatalog{}
	result, err := repo.GetAgent(mockClient, "missing")

	assert.Nil(t, result)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}
