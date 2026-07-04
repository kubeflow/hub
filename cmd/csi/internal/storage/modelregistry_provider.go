package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	kserve "github.com/kserve/kserve/pkg/agent/storage"
	"github.com/kubeflow/hub/cmd/csi/internal/constants"
	"github.com/kubeflow/hub/pkg/openapi"
)

var (
	_                         kserve.Provider = (*ModelRegistryProvider)(nil)
	ErrInvalidMRURI                           = errors.New("invalid model registry URI, use like model-registry://{dnsName}/{registeredModelName}/{versionName}")
	ErrNoVersionAssociated                    = errors.New("no versions associated to registered model")
	ErrNoArtifactAssociated                   = errors.New("no artifacts associated to model version")
	ErrNoModelArtifact                        = errors.New("no model artifact found for model version")
	ErrModelArtifactEmptyURI                  = errors.New("model artifact has empty URI")
	ErrNoStorageURI                           = errors.New("there is no storageUri supplied")
	ErrNoProtocolInSTorageURI                 = errors.New("there is no protocol specified for the storageUri")
	ErrProtocolNotSupported                   = errors.New("protocol not supported for storageUri")
	ErrFetchingModelVersion                   = errors.New("error fetching model version")
	ErrFetchingModelVersions                  = errors.New("error fetching model versions")
)

type ModelRegistryProvider struct {
	Client    *openapi.APIClient
	Providers map[kserve.Protocol]kserve.Provider
}

func NewModelRegistryProvider(client *openapi.APIClient) (*ModelRegistryProvider, error) {
	return &ModelRegistryProvider{
		Client:    client,
		Providers: map[kserve.Protocol]kserve.Provider{},
	}, nil
}

// storageUri formatted like model-registry://{modelRegistryUrl}/{registeredModelName}/{versionName}
func (p *ModelRegistryProvider) DownloadModel(modelDir string, modelName string, storageUri string) error {
	log.Printf("Download model indexed in model registry: modelName=%s, storageUri=%s, modelDir=%s",
		modelName,
		storageUri,
		modelDir,
	)

	registeredModelName, versionName, err := p.parseModelVersion(storageUri)
	if err != nil {
		return err
	}

	log.Printf("Parsed storageUri=%s as: modelRegistryUrl=%s, registeredModelName=%s, versionName=%v",
		storageUri,
		p.Client.GetConfig().Host,
		registeredModelName,
		safeString(versionName),
	)

	log.Printf("Fetching model: registeredModelName=%s, versionName=%v", registeredModelName, safeString(versionName))

	// Fetch the registered model
	model, _, err := p.Client.ModelRegistryServiceAPI.FindRegisteredModel(context.Background()).Name(registeredModelName).Execute()
	if err != nil {
		return err
	}

	log.Printf("Fetching model version: model=%v", model)

	// Fetch model version by name or latest if not specified
	version, err := p.fetchModelVersion(versionName, registeredModelName, model)
	if err != nil {
		return err
	}

	log.Printf("Fetching model artifacts: version=%v", version)

	artifacts, _, err := p.Client.ModelRegistryServiceAPI.GetModelVersionArtifacts(context.Background(), *version.Id).
		OrderBy(openapi.ORDERBYFIELD_CREATE_TIME).
		SortOrder(openapi.SORTORDER_DESC).
		Execute()
	if err != nil {
		return err
	}

	if artifacts.Size == 0 {
		return fmt.Errorf("%w %s", ErrNoArtifactAssociated, *version.Id)
	}

	modelArtifact := artifacts.Items[0].ModelArtifact
	if modelArtifact == nil {
		return fmt.Errorf("%w %s", ErrNoModelArtifact, *version.Id)
	}

	// Call appropriate kserve provider based on the indexed model artifact URI
	if modelArtifact.Uri == nil {
		return fmt.Errorf("%w %s", ErrModelArtifactEmptyURI, *modelArtifact.Id)
	}

	log.Printf("Extracting protocol from model artifact URI: %s", safeString(modelArtifact.Uri))
	protocol, err := p.extractProtocol(*modelArtifact.Uri)
	if err != nil {
		return err
	}

	log.Printf("Getting KServe provider for protocol: %s", protocol)
	provider, err := kserve.GetProvider(p.Providers, protocol)
	if err != nil {
		return err
	}

	log.Printf("Delegating to KServe provider to download model with: modelDir=%s, storageUri=%s", modelDir, safeString(modelArtifact.Uri))
	return provider.DownloadModel(modelDir, "", *modelArtifact.Uri)
}

// Possible URIs:
// (1) model-registry://{modelName}
// (2) model-registry://{modelName}/{modelVersion}
// (3) model-registry://{modelRegistryUrl}/{modelName}
// (4) model-registry://{modelRegistryUrl}/{modelName}/{modelVersion}
func (p *ModelRegistryProvider) parseModelVersion(storageUri string) (string, *string, error) {
	var versionName *string

	// Strip query parameters
	storageUri, _, _ = strings.Cut(storageUri, "?")

	// Parse the URI to retrieve the needed information to query model registry (modelArtifact)
	mrUri := strings.TrimPrefix(storageUri, string(constants.MR))

	tokens := strings.SplitN(mrUri, "/", 3)

	if len(tokens) == 0 || len(tokens) > 3 {
		return "", nil, ErrInvalidMRURI
	}

	// Check if the first token is the host and remove it so that we reduce cases (3) and (4) to (1) and (2)
	if len(tokens) >= 2 && p.Client.GetConfig().Host == tokens[0] {
		tokens = tokens[1:]
	}

	registeredModelName := tokens[0]
	if registeredModelName == "" {
		return "", nil, ErrInvalidMRURI
	}

	if len(tokens) == 2 {
		if tokens[1] == "" {
			return "", nil, ErrInvalidMRURI
		}
		versionName = &tokens[1]
	}

	return registeredModelName, versionName, nil
}

func (p *ModelRegistryProvider) fetchModelVersion(
	versionName *string,
	registeredModelName string,
	model *openapi.RegisteredModel,
) (*openapi.ModelVersion, error) {
	if versionName != nil {
		version, _, err := p.Client.ModelRegistryServiceAPI.
			FindModelVersion(context.Background()).
			Name(*versionName).
			ParentResourceId(*model.Id).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFetchingModelVersion, err)
		}

		return version, nil
	}

	versions, _, err := p.Client.ModelRegistryServiceAPI.GetRegisteredModelVersions(context.Background(), *model.Id).
		// OrderBy(openapi.ORDERBYFIELD_CREATE_TIME). not supported
		SortOrder(openapi.SORTORDER_DESC).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingModelVersions, err)
	}

	if versions.Size == 0 {
		return nil, fmt.Errorf("%w %s", ErrNoVersionAssociated, registeredModelName)
	}

	return &versions.Items[0], nil
}

func (*ModelRegistryProvider) extractProtocol(storageURI string) (kserve.Protocol, error) {
	if storageURI == "" {
		return "", ErrNoStorageURI
	}

	if !regexp.MustCompile(`\w+?://`).MatchString(storageURI) {
		return "", ErrNoProtocolInSTorageURI
	}

	for _, prefix := range kserve.SupportedProtocols {
		if strings.HasPrefix(storageURI, string(prefix)) {
			return prefix, nil
		}
	}

	return "", ErrProtocolNotSupported
}

// UploadObject cannot be implemented for the model-registry provider because:
//  1. Unlike DownloadModel which receives a model-registry URI to resolve, UploadObject
//     receives raw bucket/key parameters with no model-registry context
//  2. There's no way to determine which underlying KServe provider to delegate to
func (p *ModelRegistryProvider) UploadObject(bucket string, key string, object []byte) error {
	return fmt.Errorf("uploading objects is currently not supported when using the model-registry protocol")
}

// safeString returns a string representation of a string pointer.
// Useful for logging or printing values that may be nil. For Zero values, you can refer to ZeroIfNil instead.
// Returns "<nil>" if the input is nil
func safeString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
