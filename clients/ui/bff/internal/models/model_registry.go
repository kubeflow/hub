package models

type ModelRegistryModel struct {
	// Kubernetes service name identifying the registry.
	Name string `json:"name" example:"model-registry"`
	// Human-readable registry display name.
	DisplayName string `json:"displayName" example:"Team Model Registry"`
	// Description of the registry and its intended use.
	Description string `json:"description" example:"Models maintained by the platform team."`
	// Address of the registry's REST server.
	ServerAddress string `json:"serverAddress" example:"model-registry.example.svc:8080"`
	// Whether the REST server uses HTTPS.
	IsHTTPS bool `json:"isHttps" example:"false"`
	// Whether the service Endpoints include ready addresses.
	IsAvailable bool `json:"isAvailable" example:"true"`
}

// ServiceAuthorizationContext holds the authorization decision context
type ServiceAuthorizationContext struct {
	AllowList           bool
	AllowedServiceNames []string
	Namespace           string
}
