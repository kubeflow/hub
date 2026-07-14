package api

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/kubeflow/hub/ui/bff/internal/constants"
	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
)

type stubKubernetesClientForAuthz struct {
	k8s.KubernetesClientInterface
	allowedNamespaces map[string]bool
}

func (s *stubKubernetesClientForAuthz) CanListServicesInNamespace(_ context.Context, _ *k8s.RequestIdentity, namespace string) (bool, error) {
	return s.allowedNamespaces[namespace], nil
}

func TestAuthorizeJobNamespace(t *testing.T) {
	app := &App{logger: slog.Default()}
	identity := &k8s.RequestIdentity{UserID: "testuser"}

	tests := []struct {
		name              string
		jobNamespace      string
		allowedNamespaces map[string]bool
		wantErr           bool
	}{
		{
			name:              "empty namespace defers to existing handling",
			jobNamespace:      "",
			allowedNamespaces: map[string]bool{},
			wantErr:           false,
		},
		{
			name:              "user has access to job namespace",
			jobNamespace:      "user-project",
			allowedNamespaces: map[string]bool{"user-project": true},
			wantErr:           false,
		},
		{
			name:              "cross-namespace without access is rejected",
			jobNamespace:      "victim-namespace",
			allowedNamespaces: map[string]bool{"attacker-namespace": true},
			wantErr:           true,
		},
		{
			name:              "same as registry namespace with access is allowed",
			jobNamespace:      "rhoai-model-registries",
			allowedNamespaces: map[string]bool{"rhoai-model-registries": true},
			wantErr:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubKubernetesClientForAuthz{allowedNamespaces: tc.allowedNamespaces}
			ctx := context.WithValue(context.Background(), constants.RequestIdentityKey, identity)

			err := app.authorizeJobNamespace(ctx, client, tc.jobNamespace)
			if tc.wantErr && err == nil {
				t.Fatalf("authorizeJobNamespace(%q): expected error, got nil", tc.jobNamespace)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("authorizeJobNamespace(%q): unexpected error: %v", tc.jobNamespace, err)
			}
		})
	}
}

func TestAuthorizeJobNamespaceMissingIdentity(t *testing.T) {
	app := &App{logger: slog.Default()}
	client := &stubKubernetesClientForAuthz{allowedNamespaces: map[string]bool{"ns": true}}
	ctx := context.Background()

	err := app.authorizeJobNamespace(ctx, client, "ns")
	if err == nil {
		t.Fatal("expected error for missing identity, got nil")
	}
	expected := "missing request identity"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, fmt.Sprint(err))
	}
}
