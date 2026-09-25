package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/httpclient"

	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
)

const modelArtifactPath = "/model_artifacts"

type ModelArtifactInterface interface {
	UpdateModelArtifact(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapiv1.ModelArtifact, error)
}

type ModelArtifact struct {
	ModelArtifactInterface
}

func (a ModelArtifact) UpdateModelArtifact(client httpclient.HTTPClientInterface, id string, jsonData []byte) (*openapiv1.ModelArtifact, error) {
	path, err := url.JoinPath(modelArtifactPath, id)
	if err != nil {
		return nil, err
	}

	responseData, err := client.PATCH(path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error patching registered model: %w", err)
	}

	var modelArtifact openapiv1.ModelArtifact
	if err := json.Unmarshal(responseData, &modelArtifact); err != nil {
		return nil, fmt.Errorf("error decoding response data: %w", err)
	}

	return &modelArtifact, nil
}
