package repositories

import "github.com/kubeflow/hub/ui/bff/internal/models/health_check"

type HealthCheckRepository struct{}

func NewHealthCheckRepository() *HealthCheckRepository {
	return &HealthCheckRepository{}
}

func (r *HealthCheckRepository) HealthCheck(version string) (health_check.HealthCheckModel, error) {
	return health_check.HealthCheckModel{
		Status: "available",
		SystemInfo: health_check.SystemInfo{
			Version: version,
		},
	}, nil
}
