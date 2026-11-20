package api

import (
	"net/http"

	"github.com/kubeflow/hub/ui/bff/internal/config"
	"github.com/kubeflow/hub/ui/bff/internal/integrations/kubernetes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestSkillCatalogFilters", func() {
	var requestIdentity kubernetes.RequestIdentity

	BeforeEach(func() {
		requestIdentity = kubernetes.RequestIdentity{UserID: "user@example.com"}
	})

	// Filter options come from the catalog service alone. The BFF used to merge
	// provider/category/trustTier from the source ConfigMaps on top; that offered filter
	// values matching no indexed skill, so a user could select one and get nothing back.
	It("returns only the catalog service's filter options", func() {
		body, rs, err := setupApiTest[SkillFilterOptionsListEnvelope](
			http.MethodGet,
			"/api/v1/skill_catalog/skills_filter_options?namespace=kubeflow",
			nil,
			kubernetesMockedStaticClientFactory,
			requestIdentity,
			"kubeflow",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rs.StatusCode).To(Equal(http.StatusOK))

		// The catalog client mock returns no filters, and the seeded ConfigMap sources
		// carry provider "Community" and trustTier "communityContributed". Neither may
		// appear now that the ConfigMap merge is gone.
		if body.Data != nil && body.Data.Filters != nil {
			Expect(*body.Data.Filters).To(BeEmpty())
		}
	})
})

var _ = Describe("TestSkillMarketplace", func() {
	var requestIdentity kubernetes.RequestIdentity

	BeforeEach(func() {
		requestIdentity = kubernetes.RequestIdentity{UserID: "user@example.com"}
	})

	It("advertises the catalog's in-cluster URL when no override is configured", func() {
		body, rs, err := setupApiTest[SkillMarketplaceEnvelope](
			http.MethodGet,
			"/api/v1/skill_catalog/claude/marketplace.json?namespace=kubeflow",
			nil,
			kubernetesMockedStaticClientFactory,
			requestIdentity,
			"kubeflow",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rs.StatusCode).To(Equal(http.StatusOK))

		// Never this BFF route: it is namespace-scoped and authenticated, which an agent
		// running `/plugin marketplace add` cannot satisfy.
		Expect(body.Metadata.MarketplaceURL).To(Equal(
			"http://model-catalog.kubeflow.svc.cluster.local:8080/api/skill_catalog/v1/claude/marketplace.json"))
		Expect(body.Metadata.External).To(BeFalse())
	})

	It("uses the catalog namespace in dev mode rather than the request namespace", func() {
		body, rs, err := setupApiTestWithConfig[SkillMarketplaceEnvelope](
			http.MethodGet,
			"/api/v1/skill_catalog/claude/marketplace.json?namespace=bella-namespace",
			nil,
			kubernetesMockedStaticClientFactory,
			requestIdentity,
			"bella-namespace",
			func(cfg *config.EnvConfig) {
				cfg.DevMode = true
				cfg.DevModeCatalogNamespace = "kubeflow"
			},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rs.StatusCode).To(Equal(http.StatusOK))
		Expect(body.Metadata.MarketplaceURL).To(ContainSubstring("model-catalog.kubeflow.svc"))
	})

	It("prefers an operator-configured external URL", func() {
		body, rs, err := setupApiTestWithConfig[SkillMarketplaceEnvelope](
			http.MethodGet,
			"/api/v1/skill_catalog/claude/marketplace.json?namespace=kubeflow",
			nil,
			kubernetesMockedStaticClientFactory,
			requestIdentity,
			"kubeflow",
			func(cfg *config.EnvConfig) {
				cfg.SkillCatalogMarketplaceURL = "https://hub.example.com/api/skill_catalog/v1/claude/marketplace.json"
			},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rs.StatusCode).To(Equal(http.StatusOK))
		Expect(body.Metadata.MarketplaceURL).To(Equal(
			"https://hub.example.com/api/skill_catalog/v1/claude/marketplace.json"))
		Expect(body.Metadata.External).To(BeTrue())
	})
})
