package api

import (
	"io"
	"log/slog"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/config"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes/k8mocks"
	"github.com/stretchr/testify/require"
)

func TestNewAppOCIConfigNamespace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mockK8s   bool
		namespace string
		want      string
	}{
		{name: "mock default", mockK8s: true, want: k8mocks.OCIBaseImageConfigNamespace},
		{name: "mock explicit override", mockK8s: true, namespace: " deployment-config ", want: "deployment-config"},
		{name: "non-mock no override"},
		{name: "non-mock explicit override", namespace: " deployment-config ", want: "deployment-config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, err := NewApp(config.EnvConfig{
				AuthMethod:                  config.AuthMethodUser,
				MockK8Client:                tc.mockK8s,
				MockMRClient:                true,
				MockMRCatalogClient:         true,
				OCIBaseImageConfigNamespace: tc.namespace,
			}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, app.Shutdown()) })
			require.Equal(t, tc.want, app.repositories.ModelRegistry.OCIBaseImageConfigNamespace)
		})
	}
}
