package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	"github.com/kubeflow/hub/ui/bff/internal/mocks"
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestSkillCatalogSettings", func() {
	var requestIdentity kubernetes.RequestIdentity

	BeforeEach(func() {
		requestIdentity = kubernetes.RequestIdentity{
			UserID: "user@example.com",
		}
	})

	newGitSource := func(id, name string) *models.SkillCatalogSourceConfigPayload {
		return &models.SkillCatalogSourceConfigPayload{
			Id:      id,
			Name:    name,
			Type:    "git-skills-plugin",
			Enabled: skillHandlerBoolPtr(true),
			Repositories: []models.SkillRepository{
				{URL: "https://github.com/example/" + id, Refs: []string{"v1.0.0"}},
			},
		}
	}

	Context("fetching skill catalog source configs", func() {
		It("GET ALL returns 200 and includes both default and user managed sources", func() {
			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigListEnvelope](
				http.MethodGet,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))

			ids := make([]string, 0, len(body.Data.Catalogs))
			for _, c := range body.Data.Catalogs {
				ids = append(ids, c.Id)
			}
			Expect(ids).To(ContainElements("community_skills", "custom_skills"))
		})

		It("GET SINGLE returns 200 and never leaks the auth token", func() {
			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodGet,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(body.Data.Type).To(Equal("git-skills-plugin"))
			Expect(*body.Data.IsDefault).To(BeTrue())
			Expect(body.Data.Repositories).To(HaveLen(1))
			Expect(body.Data.Repositories[0].AuthToken).To(BeNil())
			Expect(*body.Data.TrustTier).To(Equal("communityContributed"))
		})

		It("GET returns 404 for non-existent source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodGet,
				"/api/v1/settings/skill_catalog/source_configs/does_not_exist?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("GET returns 400 when namespace is missing", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodGet,
				"/api/v1/settings/skill_catalog/source_configs",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("creating a skill source config", func() {
		It("POST returns 201 on success", func() {
			payload := SkillCatalogSourcePayloadEnvelope{Data: newGitSource("skill_handler_test_create", "Skill Handler Test")}
			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(body.Data.Id).To(Equal("skill_handler_test_create"))
			Expect(rs.Header.Get("Location")).To(ContainSubstring("skill_handler_test_create"))
		})

		It("POST returns 400 when id is missing", func() {
			source := newGitSource("", "No Id")
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for a duplicate source", func() {
			payload := SkillCatalogSourcePayloadEnvelope{Data: newGitSource("custom_skills", "Duplicate")}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for an unsupported catalog type", func() {
			source := newGitSource("skill_bad_type", "Bad Type")
			source.Type = "yaml"
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when no repository is supplied", func() {
			source := newGitSource("skill_no_repos", "No Repos")
			source.Repositories = nil
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 for an invalid trustTier", func() {
			source := newGitSource("skill_bad_tier", "Bad Tier")
			source.TrustTier = skillHandlerStringPtr("goldPlated")
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("POST returns 400 when data is missing from the envelope", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				SkillCatalogSourcePayloadEnvelope{},
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("credential syncing on create", func() {
		It("POST moves the auth token into a credentialRef and never echoes it back", func() {
			source := newGitSource("skill_with_token", "Skill With Token")
			source.Repositories[0].AuthToken = skillHandlerStringPtr("ghp_secret_value")
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}

			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(body.Data.Repositories).To(HaveLen(1))
			Expect(body.Data.Repositories[0].AuthToken).To(BeNil())
			// The catalog service resolves credentialRef as a key inside the shared
			// git-credentials Secret, so it must be the source id.
			Expect(body.Data.Repositories[0].CredentialRef).NotTo(BeNil())
			Expect(*body.Data.Repositories[0].CredentialRef).To(Equal("skill_with_token"))
		})

		It("POST returns 400 when the payload lists more than one repository", func() {
			source := newGitSource("skill_two_tokens", "Two Tokens")
			source.Repositories = []models.SkillRepository{
				{URL: "https://github.com/example/one", AuthToken: skillHandlerStringPtr("token-one")},
				{URL: "https://github.com/example/two", AuthToken: skillHandlerStringPtr("token-two")},
			}
			payload := SkillCatalogSourcePayloadEnvelope{Data: source}

			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Context("credential writes are gated on the source being persisted", func() {
		readCredential := func(key string) string {
			c, err := kubernetesMockedStaticClientFactory.GetClient(mocks.NewMockSessionContextNoParent())
			Expect(err).NotTo(HaveOccurred())
			secret, err := c.GetSecret(mocks.NewMockSessionContextNoParent(), "kubeflow", "skill-catalog-git-credentials")
			if err != nil {
				return ""
			}
			if v, ok := secret.Data[key]; ok {
				return string(v)
			}
			return secret.StringData[key]
		}

		withToken := func(id, name, token string) SkillCatalogSourcePayloadEnvelope {
			source := newGitSource(id, name)
			source.Repositories[0].AuthToken = skillHandlerStringPtr(token)
			return SkillCatalogSourcePayloadEnvelope{Data: source}
		}

		// The settings form derives a source id from the source name, so re-using a name
		// produces a colliding id. The Secret key is that same id, so a create that the
		// BFF rejects as a duplicate must not overwrite the existing source's token —
		// the admin saw the request fail and reasonably assumes nothing changed.
		It("a rejected duplicate create leaves the existing source's token intact", func() {
			_, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				withToken("acme_skills", "Acme Skills", "REAL-TOKEN"),
				kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(readCredential("acme_skills")).To(Equal("REAL-TOKEN"))

			_, rs, err = setupApiTest[Envelope[any, any]](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				withToken("acme_skills", "Acme Skills", "WRONG-TOKEN"),
				kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(readCredential("acme_skills")).To(Equal("REAL-TOKEN"),
				"a rejected create must not overwrite the colliding source's credential")
		})

		It("a rejected edit of a read-only default stores no credential", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{
						{URL: "https://github.com/example/community-skills", AuthToken: skillHandlerStringPtr("SNEAKY")},
					},
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				payload, kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
			Expect(readCredential("community_skills")).To(BeEmpty())
		})

		It("stores the credential when the source is actually created", func() {
			_, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				withToken("beta_skills", "Beta Skills", "BETA-TOKEN"),
				kubernetesMockedStaticClientFactory, requestIdentity, "kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusCreated))
			Expect(readCredential("beta_skills")).To(Equal("BETA-TOKEN"))
		})
	})

	Context("patching a skill source config", func() {
		It("PATCH returns 200 on success", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{Enabled: skillHandlerBoolPtr(false)},
			}
			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/custom_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(*body.Data.Enabled).To(BeFalse())
		})

		It("PATCH returns 404 for a non-existent source", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{Enabled: skillHandlerBoolPtr(false)},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/does_not_exist?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("PATCH returns 403 when changing type", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{Type: "yaml"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/custom_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("PATCH returns 400 for an invalid credentialRef", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{
						{URL: "https://github.com/example/custom-skills", CredentialRef: skillHandlerStringPtr("../escape")},
					},
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/custom_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("PATCH returns 403 when changing a forbidden field on a default", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{Name: "Renamed Default"},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("PATCH returns 403 when repointing a default's repositories", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{
					Repositories: []models.SkillRepository{
						{URL: "https://github.com/attacker/evil-skills"},
					},
				},
			}
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("PATCH toggles enabled on a default source via an override entry", func() {
			payload := SkillCatalogSourcePayloadEnvelope{
				Data: &models.SkillCatalogSourceConfigPayload{Enabled: skillHandlerBoolPtr(false)},
			}
			body, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPatch,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				payload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))
			Expect(*body.Data.Enabled).To(BeFalse())
			// The default's own fields survive the override.
			Expect(body.Data.Name).To(Equal("Community Skills"))
		})
	})

	Context("deleting a skill source config", func() {
		It("DELETE returns 200 on success", func() {
			createPayload := SkillCatalogSourcePayloadEnvelope{Data: newGitSource("skill_delete_handler_test", "Skill Delete Test")}
			_, _, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodPost,
				"/api/v1/settings/skill_catalog/source_configs?namespace=kubeflow",
				createPayload,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())

			_, rs, err := setupApiTest[SkillCatalogSettingsSourceConfigEnvelope](
				http.MethodDelete,
				"/api/v1/settings/skill_catalog/source_configs/skill_delete_handler_test?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusOK))

			_, rs, err = setupApiTest[Envelope[any, any]](
				http.MethodGet,
				"/api/v1/settings/skill_catalog/source_configs/skill_delete_handler_test?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("DELETE returns 404 for a non-existent source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodDelete,
				"/api/v1/settings/skill_catalog/source_configs/does_not_exist?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("DELETE returns 403 for a default source", func() {
			_, rs, err := setupApiTest[Envelope[any, any]](
				http.MethodDelete,
				"/api/v1/settings/skill_catalog/source_configs/community_skills?namespace=kubeflow",
				nil,
				kubernetesMockedStaticClientFactory,
				requestIdentity,
				"kubeflow",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(rs.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})

func skillHandlerBoolPtr(b bool) *bool {
	return &b
}

func skillHandlerStringPtr(s string) *string {
	return &s
}
