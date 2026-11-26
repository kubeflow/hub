package repositories

import (
	"context"

	k8s "github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SkillCatalogSettingsRepository", func() {
	var (
		repo      *SkillCatalogSettingsRepository
		k8sClient k8s.KubernetesClientInterface
		ctx       context.Context
	)

	BeforeEach(func() {
		repo = NewSkillCatalogSettingsRepository()
		var err error
		k8sClient, err = kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
		Expect(err).NotTo(HaveOccurred())
		ctx = mocks.NewMockSessionContextNoParent()
	})

	findById := func(list *models.SkillCatalogSourceConfigList, id string) *models.SkillCatalogSourceConfig {
		for i := range list.Catalogs {
			if list.Catalogs[i].Id == id {
				return &list.Catalogs[i]
			}
		}
		return nil
	}

	gitSource := func(id, name string) models.SkillCatalogSourceConfigPayload {
		return models.SkillCatalogSourceConfigPayload{
			Id:      id,
			Name:    name,
			Type:    SkillCatalogTypeGit,
			Enabled: skillBoolPtr(true),
			Repositories: []models.SkillRepository{
				{URL: "https://github.com/example/" + id, Refs: []string{"v1.0.0"}},
			},
		}
	}

	Describe("GetAllSkillCatalogSourceConfigs", func() {
		It("returns sources merged from both the default and user managed configmaps", func() {
			list, err := repo.GetAllSkillCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(findById(list, "community_skills")).NotTo(BeNil())
			Expect(findById(list, "custom_skills")).NotTo(BeNil())
		})

		It("marks a source from the default configmap as default", func() {
			list, err := repo.GetAllSkillCatalogSourceConfigs(ctx, k8sClient, "kubeflow")
			Expect(err).NotTo(HaveOccurred())
			Expect(*findById(list, "community_skills").IsDefault).To(BeTrue())
			Expect(*findById(list, "custom_skills").IsDefault).To(BeFalse())
		})

		It("lets a user override win over the default it shadows", func() {
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "bella-namespace", "community_skills",
				models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(false)})
			Expect(err).NotTo(HaveOccurred())

			list, err := repo.GetAllSkillCatalogSourceConfigs(ctx, k8sClient, "bella-namespace")
			Expect(err).NotTo(HaveOccurred())
			merged := findById(list, "community_skills")
			Expect(merged).NotTo(BeNil())
			Expect(*merged.Enabled).To(BeFalse())
			// Fields the override does not carry still come from the default.
			Expect(merged.Name).To(Equal("Community Skills"))
			Expect(*merged.IsDefault).To(BeTrue())
		})
	})

	Describe("GetSkillCatalogSourceConfig", func() {
		It("reads repository metadata from the first repository entry", func() {
			config, err := repo.GetSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_skills")
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Type).To(Equal(SkillCatalogTypeGit))
			Expect(config.Repositories).To(HaveLen(1))
			Expect(config.Repositories[0].URL).To(Equal("https://github.com/example/community-skills"))
			Expect(config.Repositories[0].Refs).To(Equal([]string{"v1.0.0"}))
			Expect(*config.Provider).To(Equal("Community"))
			Expect(*config.Category).To(Equal("development"))
			Expect(*config.TrustTier).To(Equal("communityContributed"))
		})

		It("returns the credentialRef but never an auth token", func() {
			config, err := repo.GetSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_skills")
			Expect(err).NotTo(HaveOccurred())
			Expect(*config.Repositories[0].CredentialRef).To(Equal("custom_skills"))
			Expect(config.Repositories[0].AuthToken).To(BeNil())
		})

		It("returns ErrSkillCatalogSourceNotFound for an unknown id", func() {
			_, err := repo.GetSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist")
			Expect(err).To(MatchError(ErrSkillCatalogSourceNotFound))
		})
	})

	Describe("CreateSkillCatalogSourceConfig", func() {
		It("persists a new source so it can be read back", func() {
			created, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", gitSource("skill_repo_create", "Skill Repo Create"))
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Id).To(Equal("skill_repo_create"))
			Expect(*created.IsDefault).To(BeFalse())

			readBack, err := repo.GetSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_create")
			Expect(err).NotTo(HaveOccurred())
			Expect(readBack.Repositories).To(HaveLen(1))
		})

		It("writes provider, category and trustTier onto the repository entry", func() {
			payload := gitSource("skill_repo_metadata", "Skill Repo Metadata")
			payload.Provider = skillStringPtr("Acme")
			payload.Category = skillStringPtr("security")
			payload.TrustTier = skillStringPtr("partnerVerified")

			created, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(*created.Provider).To(Equal("Acme"))
			Expect(*created.Category).To(Equal("security"))
			Expect(*created.TrustTier).To(Equal("partnerVerified"))
		})

		It("rejects a source that duplicates a default", func() {
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", gitSource("community_skills", "Duplicate"))
			Expect(err).To(MatchError(ErrSkillCatalogSourceAlreadyExist))
		})

		It("rejects a source with no id", func() {
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", gitSource("", "No Id"))
			Expect(err).To(MatchError(ErrSkillCatalogSourceIdRequired))
		})

		It("rejects a non git-skills-plugin type", func() {
			payload := gitSource("skill_repo_bad_type", "Bad Type")
			payload.Type = "yaml"
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
		})

		It("rejects a source with no repositories", func() {
			payload := gitSource("skill_repo_no_repos", "No Repos")
			payload.Repositories = nil
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
		})

		It("rejects a source listing more than one repository", func() {
			payload := gitSource("skill_repo_two_repos", "Two Repos")
			payload.Repositories = []models.SkillRepository{
				{URL: "https://github.com/example/one"},
				{URL: "https://github.com/example/two"},
			}
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
			Expect(err.Error()).To(ContainSubstring("single repository"))
		})

		It("rejects a credentialRef that could escape the mounted secret directory", func() {
			payload := gitSource("skill_repo_bad_ref", "Bad Credential Ref")
			payload.Repositories[0].CredentialRef = skillStringPtr("../../etc/passwd")
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
		})

		It("rejects a trustTier outside the SkillTrustTier enum", func() {
			payload := gitSource("skill_repo_bad_tier", "Bad Tier")
			payload.TrustTier = skillStringPtr("goldPlated")
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
		})
	})

	Describe("UpdateSkillCatalogSourceConfig", func() {
		It("updates a user managed source in place", func() {
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", gitSource("skill_repo_update", "Before"))
			Expect(err).NotTo(HaveOccurred())

			updated, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_update",
				models.SkillCatalogSourceConfigPayload{Name: "After", Enabled: skillBoolPtr(false)})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("After"))
			Expect(*updated.Enabled).To(BeFalse())
		})

		It("clears labels when given an empty but non-nil slice", func() {
			payload := gitSource("skill_repo_labels", "Labels")
			payload.Labels = []string{"Keep"}
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())

			updated, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_labels",
				models.SkillCatalogSourceConfigPayload{Labels: []string{}})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Labels).To(BeEmpty())
		})

		It("leaves labels alone when the payload omits them", func() {
			payload := gitSource("skill_repo_labels_kept", "Labels Kept")
			payload.Labels = []string{"Keep"}
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", payload)
			Expect(err).NotTo(HaveOccurred())

			updated, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_labels_kept",
				models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(false)})
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Labels).To(Equal([]string{"Keep"}))
		})

		It("refuses every field but enabled on a shipped default", func() {
			for _, tc := range []struct {
				field   string
				payload models.SkillCatalogSourceConfigPayload
			}{
				{"name", models.SkillCatalogSourceConfigPayload{Name: "Renamed"}},
				{"labels", models.SkillCatalogSourceConfigPayload{Labels: []string{"Mine"}}},
				{"repositories", models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{{URL: "https://github.com/attacker/evil-skills"}},
				}},
				{"provider", models.SkillCatalogSourceConfigPayload{Provider: skillStringPtr("Attacker")}},
				{"trustTier", models.SkillCatalogSourceConfigPayload{TrustTier: skillStringPtr("platformProvided")}},
			} {
				_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_skills", tc.payload)
				Expect(err).To(MatchError(ErrSkillCatalogCannotChangeDefault), "expected %s to be rejected on a default", tc.field)
			}
		})

		It("still allows toggling enabled on a shipped default", func() {
			updated, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_skills",
				models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(false)})
			Expect(err).NotTo(HaveOccurred())
			Expect(*updated.Enabled).To(BeFalse())
			Expect(updated.Name).To(Equal("Community Skills"))
		})

		It("keeps enforcing the default guard after an override entry already exists", func() {
			// The first toggle writes a user entry, so a later update takes the
			// user-source branch. The guard must still apply there.
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "bella-namespace", "community_skills",
				models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(false)})
			Expect(err).NotTo(HaveOccurred())

			_, err = repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "bella-namespace", "community_skills",
				models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{{URL: "https://github.com/attacker/evil-skills"}},
				})
			Expect(err).To(MatchError(ErrSkillCatalogCannotChangeDefault))
		})

		It("refuses to change the source type", func() {
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_skills",
				models.SkillCatalogSourceConfigPayload{Type: "yaml"})
			Expect(err).To(MatchError(ErrSkillCatalogCannotChangeType))
		})

		It("returns ErrSkillCatalogSourceNotFound for an unknown id", func() {
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist",
				models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(false)})
			Expect(err).To(MatchError(ErrSkillCatalogSourceNotFound))
		})

		It("rejects more than one repository on update too", func() {
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_skills",
				models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{
						{URL: "https://github.com/example/one"},
						{URL: "https://github.com/example/two"},
					},
				})
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
			Expect(err.Error()).To(ContainSubstring("single repository"))
		})

		It("validates repositories on update", func() {
			_, err := repo.UpdateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "custom_skills",
				models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{{URL: ""}},
				})
			Expect(err).To(MatchError(ErrSkillCatalogValidationFailed))
		})
	})

	Describe("DeleteSkillCatalogSourceConfig", func() {
		It("removes a user managed source", func() {
			_, err := repo.CreateSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", gitSource("skill_repo_delete", "Skill Repo Delete"))
			Expect(err).NotTo(HaveOccurred())

			deleted, err := repo.DeleteSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_delete")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Id).To(Equal("skill_repo_delete"))

			_, err = repo.GetSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "skill_repo_delete")
			Expect(err).To(MatchError(ErrSkillCatalogSourceNotFound))
		})

		It("refuses to delete a default source", func() {
			_, err := repo.DeleteSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "community_skills")
			Expect(err).To(MatchError(ErrSkillCatalogCannotDeleteDefault))
		})

		It("returns ErrSkillCatalogSourceNotFound for an unknown id", func() {
			_, err := repo.DeleteSkillCatalogSourceConfig(ctx, k8sClient, "kubeflow", "does_not_exist")
			Expect(err).To(MatchError(ErrSkillCatalogSourceNotFound))
		})
	})
})

func skillBoolPtr(b bool) *bool {
	return &b
}

func skillStringPtr(s string) *string {
	return &s
}
