package repositories

import (
	"context"
	"errors"
	"fmt"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrRoleBindingNotFound   = errors.New("role binding not found")
	ErrRoleBindingNameEmpty  = errors.New("role binding name cannot be empty")
	ErrRoleBindingNilPayload = errors.New("role binding payload cannot be nil")
)

// ModelRegistryRBACRepository handles Kubernetes RoleBinding CRUD operations
// scoped to model-registry resources (filtered by app.kubernetes.io/part-of=model-registry).
type ModelRegistryRBACRepository struct{}

// NewModelRegistryRBACRepository creates a new ModelRegistryRBACRepository.
func NewModelRegistryRBACRepository() *ModelRegistryRBACRepository {
	return &ModelRegistryRBACRepository{}
}

// GetRoleBindings returns all model-registry RoleBindings in the given namespace,
// mapped to the BFF models.RoleBinding type.
func (r *ModelRegistryRBACRepository) GetRoleBindings(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
) (models.RoleBindingList, error) {
	if namespace == "" {
		return models.RoleBindingList{}, fmt.Errorf("namespace cannot be empty")
	}

	k8sBindings, err := client.ListModelRegistryRoleBindings(ctx, namespace)
	if err != nil {
		return models.RoleBindingList{}, fmt.Errorf("error fetching role bindings: %w", err)
	}

	items := make([]models.RoleBinding, 0, len(k8sBindings))
	for i := range k8sBindings {
		items = append(items, toModelRoleBinding(&k8sBindings[i]))
	}

	return models.RoleBindingList{Items: items}, nil
}

// CreateRoleBinding creates a new Kubernetes RoleBinding in the given namespace.
// It automatically adds the app.kubernetes.io/part-of=model-registry label so the binding
// is included in future list calls.
func (r *ModelRegistryRBACRepository) CreateRoleBinding(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	rb *models.RoleBinding,
) (*models.RoleBinding, error) {
	if rb == nil {
		return nil, ErrRoleBindingNilPayload
	}
	if rb.Name == "" {
		return nil, ErrRoleBindingNameEmpty
	}

	k8sRB := toK8sRoleBinding(rb, namespace)

	created, err := client.CreateModelRegistryRoleBinding(ctx, namespace, k8sRB)
	if err != nil {
		return nil, fmt.Errorf("error creating role binding %q: %w", rb.Name, err)
	}

	result := toModelRoleBinding(created)
	return &result, nil
}

// PatchRoleBinding updates the subjects of an existing RoleBinding identified by name
// in the given namespace. Only the Subjects field is modified; roleRef is left untouched.
func (r *ModelRegistryRBACRepository) PatchRoleBinding(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	name string,
	rb *models.RoleBinding,
) (*models.RoleBinding, error) {
	if name == "" {
		return nil, ErrRoleBindingNameEmpty
	}
	if rb == nil {
		return nil, ErrRoleBindingNilPayload
	}

	// Build a minimal k8s RoleBinding with only the subjects to patch.
	patch := toK8sRoleBinding(rb, namespace)
	patch.Name = name

	patched, err := client.PatchModelRegistryRoleBinding(ctx, namespace, name, patch)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrRoleBindingNotFound, name)
		}
		return nil, fmt.Errorf("error patching role binding %q: %w", name, err)
	}

	result := toModelRoleBinding(patched)
	return &result, nil
}

// DeleteRoleBinding deletes a RoleBinding by name from the given namespace.
// Returns ErrRoleBindingNotFound if the binding does not exist.
func (r *ModelRegistryRBACRepository) DeleteRoleBinding(
	ctx context.Context,
	client k8s.KubernetesClientInterface,
	namespace string,
	name string,
) error {
	if name == "" {
		return ErrRoleBindingNameEmpty
	}

	err := client.DeleteModelRegistryRoleBinding(ctx, namespace, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("%w: %s", ErrRoleBindingNotFound, name)
		}
		return fmt.Errorf("error deleting role binding %q: %w", name, err)
	}

	return nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// toModelRoleBinding converts a Kubernetes RoleBinding into the BFF model.
func toModelRoleBinding(rb *rbacv1.RoleBinding) models.RoleBinding {
	return models.RoleBinding{
		ObjectMeta: rb.ObjectMeta,
		Subjects:   rb.Subjects,
		RoleRef:    rb.RoleRef,
	}
}

// toK8sRoleBinding converts a BFF models.RoleBinding into a Kubernetes RoleBinding,
// ensuring the mandatory model-registry label is always set.
func toK8sRoleBinding(rb *models.RoleBinding, namespace string) *rbacv1.RoleBinding {
	labels := make(map[string]string)
	// Preserve any labels already on the model (e.g., app.kubernetes.io/name).
	for k, v := range rb.Labels {
		labels[k] = v
	}
	// Ensure the scoping label that ListModelRegistryRoleBindings uses is always present.
	labels["app.kubernetes.io/part-of"] = "model-registry"

	subjects := rb.Subjects
	if subjects == nil {
		subjects = []rbacv1.Subject{}
	}

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rb.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Subjects: subjects,
		RoleRef:  rb.RoleRef,
	}
}
