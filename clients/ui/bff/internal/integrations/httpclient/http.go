package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	helper "github.com/kubeflow/hub/ui/bff/internal/helpers"
)

type HTTPClientInterface interface {
	GET(url string) ([]byte, error)
	POST(url string, body io.Reader) ([]byte, error)
	POSTWithContentType(url string, body io.Reader, contentType string) ([]byte, error)
	PATCH(url string, body io.Reader) ([]byte, error)
}

type HTTPClient struct {
	client  *http.Client
	baseURL string
	logger  *slog.Logger
	Headers http.Header
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HTTPError struct {
	StatusCode int `json:"-"`
	ErrorResponse
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Code, e.Message)
}

func NewHTTPClient(logger *slog.Logger, baseURL string, headers http.Header, insecureSkipVerify bool, rootCAs *x509.CertPool) (HTTPClientInterface, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"h2", "http/1.1"},
	}
	if rootCAs != nil {
		tlsCfg.RootCAs = rootCAs
	}
	return &HTTPClient{
		client: &http.Client{Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig:   tlsCfg,
		}},
		baseURL: baseURL,
		logger:  logger,
		Headers: headers,
	}, nil
}

func (c *HTTPClient) GET(url string) ([]byte, error) {
	requestId := uuid.NewString()

	fullURL := c.baseURL + url
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	c.applyHeaders(req)

	logUpstreamReq(c.logger, requestId, req)

	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	logUpstreamResp(c.logger, requestId, response, body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		if err := json.Unmarshal(body, &errorResponse); err != nil {
			// If we can't unmarshal as JSON, create a generic error response with the raw body
			c.logger.Warn("received non-JSON error response",
				"status_code", response.StatusCode,
				"content_type", response.Header.Get("Content-Type"),
				"body_preview", string(body[:min(len(body), 200)]))

			errorResponse = ErrorResponse{
				Code:    strconv.Itoa(response.StatusCode),
				Message: fmt.Sprintf("HTTP %d: %s", response.StatusCode, string(body)),
			}
		}
		httpError := &HTTPError{
			StatusCode:    response.StatusCode,
			ErrorResponse: errorResponse,
		}
		//Sometimes the code comes empty from model registry API
		//also not all error codes are correctly implemented
		//see https://github.com/kubeflow/hub/issues/95
		if httpError.Code == "" {
			httpError.Code = strconv.Itoa(response.StatusCode)
		}
		return nil, httpError
	}

	return body, nil
}

func (c *HTTPClient) POST(url string, body io.Reader) ([]byte, error) {
	return c.POSTWithContentType(url, body, "application/json")
}

func (c *HTTPClient) POSTWithContentType(url string, body io.Reader, contentType string) ([]byte, error) {
	requestId := uuid.NewString()

	fullURL := c.baseURL + url
	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)

	c.applyHeaders(req)

	logUpstreamReq(c.logger, requestId, req)

	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	logUpstreamResp(c.logger, requestId, response, responseBody)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var errorResponse ErrorResponse
		if err := json.Unmarshal(responseBody, &errorResponse); err != nil {
			// If we can't unmarshal as JSON, create a generic error response with the raw body
			c.logger.Warn("received non-JSON error response",
				"status_code", response.StatusCode,
				"content_type", response.Header.Get("Content-Type"),
				"body_preview", string(responseBody[:min(len(responseBody), 200)]))

			errorResponse = ErrorResponse{
				Code:    strconv.Itoa(response.StatusCode),
				Message: fmt.Sprintf("HTTP %d: %s", response.StatusCode, string(responseBody)),
			}
		}
		httpError := &HTTPError{
			StatusCode:    response.StatusCode,
			ErrorResponse: errorResponse,
		}
		//Sometimes the code comes empty from model registry API
		//also not all error codes are correctly implemented
		//see https://github.com/kubeflow/hub/issues/95
		if httpError.Code == "" {
			httpError.Code = strconv.Itoa(response.StatusCode)
		}
		return nil, httpError
	}

	return responseBody, nil
}

func (c *HTTPClient) PATCH(url string, body io.Reader) ([]byte, error) {
	fullURL := c.baseURL + url
	req, err := http.NewRequest(http.MethodPatch, fullURL, body)
	if err != nil {
		return nil, err
	}

	requestId := uuid.NewString()

	req.Header.Set("Content-Type", "application/json")

	c.applyHeaders(req)

	logUpstreamReq(c.logger, requestId, req)

	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	logUpstreamResp(c.logger, requestId, response, responseBody)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		if err := json.Unmarshal(responseBody, &errorResponse); err != nil {
			// If we can't unmarshal as JSON, create a generic error response with the raw body
			c.logger.Warn("received non-JSON error response",
				"status_code", response.StatusCode,
				"content_type", response.Header.Get("Content-Type"),
				"body_preview", string(responseBody[:min(len(responseBody), 200)]))

			errorResponse = ErrorResponse{
				Code:    strconv.Itoa(response.StatusCode),
				Message: fmt.Sprintf("HTTP %d: %s", response.StatusCode, string(responseBody)),
			}
		}
		httpError := &HTTPError{
			StatusCode:    response.StatusCode,
			ErrorResponse: errorResponse,
		}
		//Sometimes the code comes empty from model registry API
		//also not all error codes are correctly implemented
		//see https://github.com/kubeflow/hub/issues/95
		if httpError.Code == "" {
			httpError.Code = strconv.Itoa(response.StatusCode)
		}
		return nil, httpError
	}
	return responseBody, nil
}

func (c *HTTPClient) applyHeaders(req *http.Request) {
	if c.Headers != nil {
		for key, values := range c.Headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
}

func logUpstreamReq(logger *slog.Logger, reqId string, req *http.Request) {
	logger.Debug("Making upstream HTTP request", slog.String("request_id", reqId), slog.Any("request", helper.RequestLogValuer{Request: req}))
}

func logUpstreamResp(logger *slog.Logger, reqId string, resp *http.Response, body []byte) {
	logger.Debug("Received upstream HTTP response", slog.String("request_id", reqId), slog.Any("response", helper.ResponseLogValuer{Response: resp}))
}
