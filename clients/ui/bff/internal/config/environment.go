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
	OCIBaseImageConfigNamespace string

	Port                     int
	MockK8Client             bool
	MockMRClient             bool
	MockMRCatalogClient      bool
	DevMode                  bool
	DeploymentMode           DeploymentMode
	DevModeModelRegistryPort int
	DevModeCatalogPort       int
	StaticAssetsDir          string
	LogLevel                 slog.Level
	AllowedOrigins           []string
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
