package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"

	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
)

const artifactPath = "/artifacts"

type ArtifactInterface interface {
	GetAllArtifacts(client httpclient.HTTPClientInterface, pageValues url.Values) (*openapiv1.ArtifactList, error)
	GetArtifact(client httpclient.HTTPClientInterface, id string) (*openapiv1.Artifact, error)
	CreateArtifact(client httpclient.HTTPClientInterface, jsonData []byte) (*openapiv1.Artifact, error)
}

type Artifact struct {
	ArtifactInterface
}

func (a Artifact) GetAllArtifacts(client httpclient.HTTPClientInterface, pageValues url.Values) (*openapiv1.ArtifactList, error) {
	responseData, err := client.GET(UrlWithPageParams(artifactPath, pageValues))
	if err != nil {
		return nil, fmt.Errorf("error fetching artifacts: %w", err)
	}

	var artifacts openapiv1.ArtifactList
	if err := json.Unmarshal(responseData, &artifacts); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &artifacts, nil
}

func (a Artifact) GetArtifact(client httpclient.HTTPClientInterface, id string) (*openapiv1.Artifact, error) {
	path, err := url.JoinPath(artifactPath, id)
	if err != nil {
		return nil, err
	}

	responseData, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching artifacts: %w", err)
	}

	var artifact openapiv1.Artifact
	if err := json.Unmarshal(responseData, &artifact); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &artifact, nil
}

func (a Artifact) CreateArtifact(client httpclient.HTTPClientInterface, jsonData []byte) (*openapiv1.Artifact, error) {
	responseData, err := client.POST(artifactPath, bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, fmt.Errorf("error creating artifact: %w", err)
	}

	var artifact openapiv1.Artifact
	if err := json.Unmarshal(responseData, &artifact); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &artifact, nil
}
