package repositories

import "github.com/kubeflow/hub/ui/bff/internal/models/healthcheck"

type HealthCheckRepository struct{}

func NewHealthCheckRepository() *HealthCheckRepository {
	return &HealthCheckRepository{}
}

func (r *HealthCheckRepository) HealthCheck(version string) (healthcheck.HealthCheckModel, error) {
	return healthcheck.HealthCheckModel{
		Status: "available",
		SystemInfo: healthcheck.SystemInfo{
			Version: version,
		},
	}, nil
}
