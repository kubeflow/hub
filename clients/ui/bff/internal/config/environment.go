package config

import (
	"fmt"
	"log/slog"
	"strings"
)

const (
	// AuthMethodInternal uses the credentials of the running backend.
	// If running inside the cluster, it uses the pod's service account.
	// If running locally (e.g. for development), it uses the current user's kubeconfig context.
	// This is the default authentication method.
	// This uses kubeflow-userid header to carry the user identity.
	AuthMethodInternal = "internal"

	// AuthMethodUser uses a user-provided Bearer token for authentication.
	AuthMethodUser = "user_token"

	// DefaultAuthTokenHeader is the standard header for Bearer token auth.
	DefaultAuthTokenHeader = "Authorization"

	// DefaultAuthTokenPrefix is the prefix used in the Authorization header.
	// note: the space here is intentional, as the prefix is "Bearer " (with a space).
	DefaultAuthTokenPrefix = "Bearer "
)

// DeploymentMode represents the deployment mode enum
type DeploymentMode string

const (
	// DeploymentModeKubeflow represents the Kubeflow integration mode
	DeploymentModeKubeflow DeploymentMode = "kubeflow"
	// DeploymentModeFederated represents the federated platform mode
	DeploymentModeFederated DeploymentMode = "federated"
	// DeploymentModeStandalone represents the standalone mode
	DeploymentModeStandalone DeploymentMode = "standalone"
)

// String implements the fmt.Stringer interface
func (d DeploymentMode) String() string {
	return string(d)
}

// Set implements the flag.Value interface
func (d *DeploymentMode) Set(value string) error {
	switch strings.ToLower(value) {
	case "kubeflow":
		*d = DeploymentModeKubeflow
	case "federated":
		*d = DeploymentModeFederated
	case "standalone":
		*d = DeploymentModeStandalone
	default:
		return fmt.Errorf("invalid deployment mode: %s (must be kubeflow, federated, or standalone)", value)
	}
	return nil
}

// IsKubeflowMode returns true if the deployment mode is Kubeflow
func (d DeploymentMode) IsKubeflowMode() bool {
	return d == DeploymentModeKubeflow
}

// IsStandaloneMode returns true if the deployment mode is standalone
func (d DeploymentMode) IsStandaloneMode() bool {
	return d == DeploymentModeStandalone
}

// IsFederatedMode returns true if the deployment mode is federated
func (d DeploymentMode) IsFederatedMode() bool {
	return d == DeploymentModeFederated
}

type EnvConfig struct {
	Port                     int
	MockK8Client             bool
	MockMRClient             bool
	MockMRCatalogClient      bool
	DevMode                  bool
	DeploymentMode           DeploymentMode
	DevModeModelRegistryPort int
	DevModeCatalogPort       int
	DevModeCatalogNamespace  string
	// SkillCatalogGitCredentialsSecret is the single Secret mounted into the catalog
	// pod at /etc/skill-catalog/git-credentials. Each skill source's git token is
	// stored under a key named after the source id, and that key name is what the
	// source's credentialRef points at — the catalog service resolves credentialRef
	// as a filename in that mount, not as a Secret name.
	SkillCatalogGitCredentialsSecret string
	// SkillCatalogMarketplaceURL overrides the marketplace.json URL shown in the skill
	// install instructions. Leave empty for the default, which is the catalog service's
	// in-cluster address — reachable by agents running in the cluster, which is the
	// primary consumer. Set it to an externally reachable URL when the catalog has been
	// put behind a Route or Ingress, so the command also works from outside the cluster.
	SkillCatalogMarketplaceURL string
	StaticAssetsDir            string
	LogLevel                   slog.Level
	AllowedOrigins             []string
	// BundlePaths is a list of filesystem paths to PEM-encoded CA bundle files.
	// If provided, the application will attempt to load these files and add the
	// certificates to the HTTP client's Root CAs for outbound TLS connections.
	// Missing or unreadable files are ignored.
	BundlePaths []string

	// ─── AUTH ───────────────────────────────────────────────────
	// Specifies the authentication method used by the server.
	// Valid values: "internal" or "user_token"
	AuthMethod string

	// Header used to extract the authentication token.
	// Default is "Authorization" and can be overridden via CLI/env for proxy integration scenarios.
	AuthTokenHeader string

	// Optional prefix to strip from the token header value.
	// Default is "Bearer ", can be set to empty if the token is sent without a prefix.
	AuthTokenPrefix string

	// ─── TLS ────────────────────────────────────────────────────
	// TLS verification settings for HTTP client connections to Model Registry
	// InsecureSkipVerify when true, skips TLS certificate verification (useful for development/local setups)
	// Default is false (secure) for production environments
	InsecureSkipVerify bool

	// ─── DEPRECATED ─────────────────────────────────────────────
	// The following fields are deprecated and maintained for backward compatibility
	// Use DeploymentMode instead
	StandaloneMode    bool
	FederatedPlatform bool
}
