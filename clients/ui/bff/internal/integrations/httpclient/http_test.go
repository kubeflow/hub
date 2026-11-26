package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestHTTPClient_GET_Success(t *testing.T) {
	// Setup test server
	expectedResponse := map[string]interface{}{
		"dora": "bella",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(expectedResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Make the request
	response, err := client.GET("/test")

	// Verify successful response
	assert.NoError(t, err)
	assert.NotNil(t, response)

	var actualResponse map[string]interface{}
	err = json.Unmarshal(response, &actualResponse)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
}

func TestHTTPClient_GET_Error(t *testing.T) {
	// Setup test server that returns an error
	errorResponse := ErrorResponse{
		Code:    "500",
		Message: "the server encountered a problem and could not process your request",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		err := json.NewEncoder(w).Encode(errorResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Make the request
	response, err := client.GET("/test")

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, response)

	httpErr, ok := err.(*HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode)
	assert.Equal(t, errorResponse.Code, httpErr.Code)
	assert.Equal(t, errorResponse.Message, httpErr.Message)
}

func TestHTTPClient_POST_Success(t *testing.T) {
	// Setup test server
	requestBody := map[string]string{"dora": "test-model"}
	expectedResponse := map[string]interface{}{
		"id":   "123",
		"name": "dora-model",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var receivedBody map[string]string
		err = json.Unmarshal(body, &receivedBody)
		assert.NoError(t, err)
		assert.Equal(t, requestBody, receivedBody)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(expectedResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Prepare request body
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Make the request
	response, err := client.POST("/test", bytes.NewReader(bodyBytes))

	// Verify response
	assert.NoError(t, err)
	assert.NotNil(t, response)

	var actualResponse map[string]interface{}
	err = json.Unmarshal(response, &actualResponse)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
}

func TestHTTPClient_POST_Error(t *testing.T) {
	// Setup test server that returns an error
	requestBody := map[string]string{"name": "test-model"}
	errorResponse := ErrorResponse{
		Code:    "400",
		Message: "Bad Request parameters",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		err := json.NewEncoder(w).Encode(errorResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Prepare request body
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Make the request
	response, err := client.POST("/test", bytes.NewReader(bodyBytes))

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, response)

	httpErr, ok := err.(*HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
	assert.Equal(t, errorResponse.Code, httpErr.Code)
	assert.Equal(t, errorResponse.Message, httpErr.Message)
}

func TestHTTPClient_PATCH_Success(t *testing.T) {
	// Setup test server
	requestBody := map[string]string{"name": "updated-model"}
	expectedResponse := map[string]interface{}{
		"id":   "123",
		"name": "updated--bella-model",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/test/123", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var receivedBody map[string]string
		err = json.Unmarshal(body, &receivedBody)
		assert.NoError(t, err)
		assert.Equal(t, requestBody, receivedBody)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(expectedResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Prepare request body
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Make the request
	response, err := client.PATCH("/test/123", bytes.NewReader(bodyBytes))

	// Verify response
	assert.NoError(t, err)
	assert.NotNil(t, response)

	var actualResponse map[string]interface{}
	err = json.Unmarshal(response, &actualResponse)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, actualResponse)
}

func TestHTTPClient_PATCH_Error(t *testing.T) {
	// Setup test server that returns an error
	requestBody := map[string]string{"name": "updated-model"}
	errorResponse := ErrorResponse{
		Code:    "404",
		Message: "Resource not found",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		err := json.NewEncoder(w).Encode(errorResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Prepare request body
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Make the request
	response, err := client.PATCH("/test/123", bytes.NewReader(bodyBytes))

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, response)

	httpErr, ok := err.(*HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
	assert.Equal(t, errorResponse.Code, httpErr.Code)
	assert.Equal(t, errorResponse.Message, httpErr.Message)
}

func TestHTTPClient_DELETE_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/sources/test-source/status", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Empty(t, body)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewHTTPClient(setupTestLogger(), server.URL, nil, false, nil)
	require.NoError(t, err)

	err = client.DELETE(context.Background(), "/sources/test-source/status")
	assert.NoError(t, err)
}

func TestHTTPClient_DELETE_Error(t *testing.T) {
	errorResponse := ErrorResponse{
		Code:    "404",
		Message: "Source not found",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/sources/test-source/status", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(errorResponse)
		assert.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewHTTPClient(setupTestLogger(), server.URL, nil, false, nil)
	require.NoError(t, err)

	err = client.DELETE(context.Background(), "/sources/test-source/status")

	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
	assert.Equal(t, errorResponse.Code, httpErr.Code)
	assert.Equal(t, errorResponse.Message, httpErr.Message)
}

func TestHTTPClient_GET_NonJSONError(t *testing.T) {
	// Setup test server that returns a non-JSON error (like HTML)
	htmlErrorResponse := `<!DOCTYPE html>
<html>
<head><title>Connection refused</title></head>
<body><h1>Connection refused</h1></body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, err := w.Write([]byte(htmlErrorResponse))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create http client pointing to test server
	logger := setupTestLogger()
	client, err := NewHTTPClient(logger, server.URL, nil, false, nil)
	require.NoError(t, err)

	// Make the request
	response, err := client.GET("/test")

	// Verify error handling
	assert.Error(t, err)
	assert.Nil(t, response)

	httpErr, ok := err.(*HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, httpErr.StatusCode)
	assert.Equal(t, "502", httpErr.Code)
	assert.Contains(t, httpErr.Message, "Connection refused")
	assert.Contains(t, httpErr.Message, "502")
}
