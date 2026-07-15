package main

import (
	"log"
	"os"
	"strings"

	"github.com/kubeflow/hub/cmd/csi/internal/modelregistry"
	"github.com/kubeflow/hub/cmd/csi/internal/storage"
	"github.com/kubeflow/hub/pkg/openapi"
)

const (
	modelRegistryBaseUrlEnv           = "MODEL_REGISTRY_BASE_URL"
	modelRegistrySchemeEnv            = "MODEL_REGISTRY_SCHEME"
	modelRegistryServiceNameEnv       = "MODEL_REGISTRY_SERVICE_NAME"
	modelRegistryClusterDomainEnv     = "MODEL_REGISTRY_CLUSTER_DOMAIN"
	allowedArtifactURIPrefixesEnv     = "ALLOWED_ARTIFACT_URI_PREFIXES"
	modelRegistryBaseUrlDefault       = "localhost:8080"
	modelRegistrySchemeDefault        = "http"
	modelRegistryServiceNameDefault   = "model-registry-service"
	modelRegistryClusterDomainDefault = "svc.cluster.local"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: ./mr-storage-initializer <src-uri> <dest-path>")
	}

	sourceUri := os.Args[1]
	destPath := os.Args[2]

	log.Printf("Initializing, args: src_uri [%s] dest_path[ [%s]\n", sourceUri, destPath)

	baseUrl, ok := os.LookupEnv(modelRegistryBaseUrlEnv)
	if !ok || baseUrl == "" {
		baseUrl = modelRegistryBaseUrlDefault
	}

	scheme, ok := os.LookupEnv(modelRegistrySchemeEnv)
	if !ok || scheme == "" {
		scheme = modelRegistrySchemeDefault
	}

	serviceName, ok := os.LookupEnv(modelRegistryServiceNameEnv)
	if !ok || serviceName == "" {
		serviceName = modelRegistryServiceNameDefault
	}

	clusterDomain, ok := os.LookupEnv(modelRegistryClusterDomainEnv)
	if !ok || clusterDomain == "" {
		clusterDomain = modelRegistryClusterDomainDefault
	}

	var allowedPrefixes []string
	if prefixesStr, ok := os.LookupEnv(allowedArtifactURIPrefixesEnv); ok && prefixesStr != "" {
		for _, p := range strings.Split(prefixesStr, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				allowedPrefixes = append(allowedPrefixes, trimmed)
			}
		}
		log.Printf("Artifact URI allowlist configured with %d prefix(es): %v", len(allowedPrefixes), allowedPrefixes)
	} else {
		log.Printf("No artifact URI allowlist configured (%s not set), all URIs will be accepted", allowedArtifactURIPrefixesEnv)
	}

	cfg := openapi.NewConfiguration()
	cfg.Host = baseUrl
	cfg.Scheme = scheme

	apiClient := modelregistry.NewAPIClient(cfg, sourceUri, serviceName, clusterDomain)

	provider, err := storage.NewModelRegistryProvider(apiClient, allowedPrefixes)
	if err != nil {
		log.Fatalf("Error initiliazing model registry provider: %v", err)
	}

	if err := provider.DownloadModel(destPath, "", sourceUri); err != nil {
		log.Fatalf("Error downloading the model: %s", err.Error())
	}
}
