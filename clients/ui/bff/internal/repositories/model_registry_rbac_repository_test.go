package repositories

import (
	"context"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ModelRegistryRBACRepository", func() {
	var (
		repo      *ModelRegistryRBACRepository
		k8sClient k8s.KubernetesClientInterface
		ctx       context.Context
	)

	BeforeEach(func() {
		repo = NewModelRegistryRBACRepository()
		var err error
		k8sClient, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
		Expect(err).NotTo(HaveOccurred())
		ctx = mocks.NewMockSessionContextNoParent()
	})

	// ─── GetRoleBindings ────────────────────────────────────────────────────────

	Describe("GetRoleBindings", func() {
		It("should return model-registry-scoped role bindings from the kubeflow namespace", func() {
			list, err := repo.GetRoleBindings(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Items).NotTo(BeEmpty())

			// Verify every returned binding carries the scoping label.
			for _, rb := range list.Items {
				Expect(rb.Labels).To(HaveKeyWithValue("app.kubernetes.io/part-of", "model-registry"))
			}
		})

		It("should return the pre-seeded mock bindings by name", func() {
			list, err := repo.GetRoleBindings(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())

			names := make([]string, 0, len(list.Items))
			for _, rb := range list.Items {
				names = append(names, rb.Name)
			}
			Expect(names).To(ContainElements("model-registry-user-binding", "model-registry-admin-binding"))
		})

		It("should return an error when namespace is empty", func() {
			_, err := repo.GetRoleBindings(ctx, k8sClient, "")
			Expect(err).To(HaveOccurred())
		})
	})

	// ─── CreateRoleBinding ──────────────────────────────────────────────────────

	Describe("CreateRoleBinding", func() {
		It("should create a new role binding and return it with the scoping label", func() {
			newRB := &models.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-create-binding",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "User",
						Name:     "new-user@example.com",
						APIGroup: "rbac.authorization.k8s.io",
					},
				},
				RoleRef: rbacv1.RoleRef{
					Kind:     "Role",
					Name:     "registry-user-model-registry",
					APIGroup: "rbac.authorization.k8s.io",
				},
			}

			created, err := repo.CreateRoleBinding(ctx, k8sClient, "kubeflow", newRB)
			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Name).To(Equal("test-create-binding"))
			Expect(created.Labels).To(HaveKeyWithValue("app.kubernetes.io/part-of", "model-registry"))
			Expect(created.Subjects).To(HaveLen(1))
			Expect(created.Subjects[0].Name).To(Equal("new-user@example.com"))
		})

		It("should return an error when the role binding name is empty", func() {
			rb := &models.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: ""},
				RoleRef: rbacv1.RoleRef{
					Kind:     "Role",
					Name:     "registry-user-model-registry",
					APIGroup: "rbac.authorization.k8s.io",
				},
			}
			_, err := repo.CreateRoleBinding(ctx, k8sClient, "kubeflow", rb)
			Expect(err).To(MatchError(ErrRoleBindingNameEmpty))
		})

		It("should return an error when payload is nil", func() {
			_, err := repo.CreateRoleBinding(ctx, k8sClient, "kubeflow", nil)
			Expect(err).To(MatchError(ErrRoleBindingNilPayload))
		})
	})

	// ─── PatchRoleBinding ───────────────────────────────────────────────────────

	Describe("PatchRoleBinding", func() {
		It("should update the subjects on an existing role binding", func() {
			updatedRB := &models.RoleBinding{
				Subjects: []rbacv1.Subject{
					{
						Kind:     "User",
						Name:     "updated-user@example.com",
						APIGroup: "rbac.authorization.k8s.io",
					},
				},
			}

			patched, err := repo.PatchRoleBinding(ctx, k8sClient, "kubeflow", "model-registry-user-binding", updatedRB)
			Expect(err).NotTo(HaveOccurred())
			Expect(patched).NotTo(BeNil())
			Expect(patched.Name).To(Equal("model-registry-user-binding"))
		})

		It("should return an error when role binding name is empty", func() {
			_, err := repo.PatchRoleBinding(ctx, k8sClient, "kubeflow", "", &models.RoleBinding{})
			Expect(err).To(MatchError(ErrRoleBindingNameEmpty))
		})

		It("should return an error when payload is nil", func() {
			_, err := repo.PatchRoleBinding(ctx, k8sClient, "kubeflow", "model-registry-user-binding", nil)
			Expect(err).To(MatchError(ErrRoleBindingNilPayload))
		})
	})

	// ─── DeleteRoleBinding ──────────────────────────────────────────────────────

	Describe("DeleteRoleBinding", func() {
		It("should delete an existing role binding successfully", func() {
			// First create a binding to delete so we don't disrupt other tests.
			toDelete := &models.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "binding-to-delete"},
				RoleRef: rbacv1.RoleRef{
					Kind:     "Role",
					Name:     "registry-user-model-registry",
					APIGroup: "rbac.authorization.k8s.io",
				},
			}
			_, err := repo.CreateRoleBinding(ctx, k8sClient, "kubeflow", toDelete)
			Expect(err).NotTo(HaveOccurred())

			err = repo.DeleteRoleBinding(ctx, k8sClient, "kubeflow", "binding-to-delete")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return ErrRoleBindingNotFound for a non-existent binding", func() {
			err := repo.DeleteRoleBinding(ctx, k8sClient, "kubeflow", "does-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role binding not found"))
		})

		It("should return an error when name is empty", func() {
			err := repo.DeleteRoleBinding(ctx, k8sClient, "kubeflow", "")
			Expect(err).To(MatchError(ErrRoleBindingNameEmpty))
		})
	})
})
