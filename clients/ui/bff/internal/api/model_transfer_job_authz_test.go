package api

import "testing"

// TestEnsureNamespaceMatch guards the model-transfer-job BOLA fix: the namespace
// a handler operates on (jobNamespace query parameter or request-body namespace)
// must not differ from the namespace authorized by the SubjectAccessReview in
// RequireAccessToMRService.
func TestEnsureNamespaceMatch(t *testing.T) {
	tests := []struct {
		name       string
		authorized string
		operated   string
		wantErr    bool
	}{
		{
			name:       "same namespace is allowed",
			authorized: "kubeflow",
			operated:   "kubeflow",
			wantErr:    false,
		},
		{
			name:       "empty operated namespace defers to existing handling",
			authorized: "kubeflow",
			operated:   "",
			wantErr:    false,
		},
		{
			name:       "cross-namespace operation is rejected",
			authorized: "mr-attacker",
			operated:   "mr-victim",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureNamespaceMatch(tc.authorized, tc.operated)
			if tc.wantErr && err == nil {
				t.Fatalf("ensureNamespaceMatch(%q, %q): expected error, got nil", tc.authorized, tc.operated)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ensureNamespaceMatch(%q, %q): unexpected error %v", tc.authorized, tc.operated, err)
			}
		})
	}
}
