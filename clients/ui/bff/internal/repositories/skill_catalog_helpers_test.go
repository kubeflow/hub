package repositories

import (
	"github.com/kubeflow/hub/ui/bff/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("skill catalog helpers", func() {

	ptr := func(s string) *string { return &s }

	Describe("validateSkillCatalogId", func() {
		It("accepts an id with only lowercase letters, numbers, and underscores", func() {
			Expect(validateSkillCatalogId("my_source_1")).NotTo(HaveOccurred())
		})

		It("accepts a hyphenated id, unlike the shared model/mcp catalog validator", func() {
			Expect(validateSkillCatalogId("my-source")).NotTo(HaveOccurred())
		})

		It("rejects an empty id", func() {
			Expect(validateSkillCatalogId("")).To(HaveOccurred())
		})

		It("rejects an id with disallowed characters", func() {
			err := validateSkillCatalogId("my source!")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("lowercase letters, numbers, hyphens, and underscores"))
		})
	})

	Describe("validateSourceMetadataFields", func() {
		It("accepts all three fields absent (nil)", func() {
			Expect(validateSourceMetadataFields(nil, nil, nil)).To(Succeed())
		})
		It("accepts all three fields populated with valid values", func() {
			Expect(validateSourceMetadataFields(
				ptr("Development"), ptr("Matt"), ptr("communityContributed"),
			)).To(Succeed())
		})

		Context("category", func() {
			It("rejects an empty-string category", func() {
				Expect(validateSourceMetadataFields(ptr(""), nil, nil)).
					To(MatchError(ContainSubstring("category must not be an empty string")))
			})
			It("accepts a non-empty category", func() {
				Expect(validateSourceMetadataFields(ptr("productivity"), nil, nil)).To(Succeed())
			})
		})

		Context("provider", func() {
			It("rejects an empty-string provider", func() {
				Expect(validateSourceMetadataFields(nil, ptr(""), nil)).
					To(MatchError(ContainSubstring("provider must not be an empty string")))
			})
			It("accepts a non-empty provider", func() {
				Expect(validateSourceMetadataFields(nil, ptr("Red Hat"), nil)).To(Succeed())
			})
		})

		Context("trustTier", func() {
			It("rejects 'Community'", func() {
				Expect(validateSourceMetadataFields(nil, nil, ptr("Community"))).
					To(MatchError(ContainSubstring("invalid trustTier")))
			})
			It("accepts all valid enum values", func() {
				for tier := range validSkillTrustTiers {
					t := tier
					Expect(validateSourceMetadataFields(nil, nil, &t)).To(Succeed())
				}
			})
		})
	})

	Describe("validateSkillCatalogSourceConfigPayload (create)", func() {
		basePayload := func() models.SkillCatalogSourceConfig {
			return models.SkillCatalogSourceConfig{
				Id:           "test-source",
				Name:         "Test Source",
				Type:         SkillCatalogTypeGit,
				Repositories: []models.SkillRepository{{URL: "https://github.com/example/skills.git"}},
			}
		}

		It("accepts a fully valid payload with all three metadata fields", func() {
			p := basePayload()
			p.Category = ptr("Development")
			p.Provider = ptr("Matt")
			p.TrustTier = ptr("communityContributed")
			Expect(validateSkillCatalogSourceConfigPayload(p)).To(Succeed())
		})
		It("rejects an empty category", func() {
			p := basePayload()
			p.Category = ptr("")
			Expect(validateSkillCatalogSourceConfigPayload(p)).
				To(MatchError(ContainSubstring("category must not be an empty string")))
		})
		It("rejects an empty provider", func() {
			p := basePayload()
			p.Provider = ptr("")
			Expect(validateSkillCatalogSourceConfigPayload(p)).
				To(MatchError(ContainSubstring("provider must not be an empty string")))
		})
		It("rejects an invalid trustTier", func() {
			p := basePayload()
			p.TrustTier = ptr("Community")
			Expect(validateSkillCatalogSourceConfigPayload(p)).
				To(MatchError(ContainSubstring("invalid trustTier")))
		})
	})

	Describe("validateSkillCatalogUpdatePayload (patch)", func() {
		It("accepts an empty payload (toggle-only)", func() {
			Expect(validateSkillCatalogUpdatePayload(models.SkillCatalogSourceConfig{})).To(Succeed())
		})
		It("accepts all three metadata fields set to valid values", func() {
			p := models.SkillCatalogSourceConfig{
				Category:  ptr("Development"),
				Provider:  ptr("Matt"),
				TrustTier: ptr("communityContributed"),
			}
			Expect(validateSkillCatalogUpdatePayload(p)).To(Succeed())
		})
		It("rejects an empty-string category in a patch", func() {
			p := models.SkillCatalogSourceConfig{Category: ptr("")}
			Expect(validateSkillCatalogUpdatePayload(p)).
				To(MatchError(ContainSubstring("category must not be an empty string")))
		})
		It("rejects an empty-string provider in a patch", func() {
			p := models.SkillCatalogSourceConfig{Provider: ptr("")}
			Expect(validateSkillCatalogUpdatePayload(p)).
				To(MatchError(ContainSubstring("provider must not be an empty string")))
		})
		It("rejects an invalid trustTier in a patch", func() {
			p := models.SkillCatalogSourceConfig{TrustTier: ptr("Community")}
			Expect(validateSkillCatalogUpdatePayload(p)).
				To(MatchError(ContainSubstring("invalid trustTier")))
		})
	})

	Describe("ParseSkillCatalogYaml", func() {
		It("reads category/provider/trustTier from repository level (canonical)", func() {
			raw := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      repositories:
        - url: https://github.com/example/skills.git
          provider: Matt
          category: Development
          trustTier: communityContributed
`
			configs, err := ParseSkillCatalogYaml(raw, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs).To(HaveLen(1))
			Expect(configs[0].Category).To(HaveValue(Equal("Development")))
			Expect(configs[0].Provider).To(HaveValue(Equal("Matt")))
			Expect(configs[0].TrustTier).To(HaveValue(Equal("communityContributed")))
		})

		It("falls back to properties level for hand-written ConfigMaps", func() {
			raw := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      category: Development
      provider: Matt
      trustTier: communityContributed
      repositories:
        - url: https://github.com/example/skills.git
`
			configs, err := ParseSkillCatalogYaml(raw, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs).To(HaveLen(1))
			Expect(configs[0].Category).To(HaveValue(Equal("Development")))
			Expect(configs[0].Provider).To(HaveValue(Equal("Matt")))
			Expect(configs[0].TrustTier).To(HaveValue(Equal("communityContributed")))
		})

		It("prefers repository-level values over properties-level when both exist", func() {
			raw := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      category: WrongLevel
      provider: WrongLevel
      trustTier: platformProvided
      repositories:
        - url: https://github.com/example/skills.git
          category: Development
          provider: Matt
          trustTier: communityContributed
`
			configs, err := ParseSkillCatalogYaml(raw, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Category).To(HaveValue(Equal("Development")))
			Expect(configs[0].Provider).To(HaveValue(Equal("Matt")))
			Expect(configs[0].TrustTier).To(HaveValue(Equal("communityContributed")))
		})

		It("returns an error instead of silently dropping repositories when the shape is malformed", func() {
			raw := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      repositories: "not-a-list-of-repos"
`
			configs, err := ParseSkillCatalogYaml(raw, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test-source"))
			Expect(configs).To(BeNil())
		})
	})

	Describe("ConvertSkillSourceConfigToYamlEntry", func() {
		It("writes all three fields inside each repository entry, not at properties level", func() {
			payload := models.SkillCatalogSourceConfig{
				Id:        "test-source",
				Name:      "Test Source",
				Type:      SkillCatalogTypeGit,
				Provider:  ptr("Matt"),
				Category:  ptr("Development"),
				TrustTier: ptr("communityContributed"),
				Repositories: []models.SkillRepository{
					{URL: "https://github.com/example/skills.git"},
				},
			}
			entry := ConvertSkillSourceConfigToYamlEntry(payload)

			props, ok := entry["properties"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(props).NotTo(HaveKey("category"), "category must not leak to properties level")
			Expect(props).NotTo(HaveKey("provider"), "provider must not leak to properties level")
			Expect(props).NotTo(HaveKey("trustTier"), "trustTier must not leak to properties level")

			repos, ok := props["repositories"].([]map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]).To(HaveKeyWithValue("category", "Development"))
			Expect(repos[0]).To(HaveKeyWithValue("provider", "Matt"))
			Expect(repos[0]).To(HaveKeyWithValue("trustTier", "communityContributed"))
		})
	})

	Describe("UpdateSkillCatalogSourceInYAML", func() {
		It("migrates all three fields from properties level to repository level on update", func() {
			existing := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      category: Development
      provider: Matt
      trustTier: communityContributed
      repositories:
        - url: https://github.com/example/skills.git
`
			payload := models.SkillCatalogSourceConfig{
				Category:     ptr("Development"),
				Provider:     ptr("Matt"),
				TrustTier:    ptr("communityContributed"),
				Repositories: []models.SkillRepository{{URL: "https://github.com/example/skills.git"}},
			}
			updated, err := UpdateSkillCatalogSourceInYAML(existing, "test-source", payload)
			Expect(err).NotTo(HaveOccurred())

			// Re-parse and verify all three landed in the repo, not at properties level
			configs, err := ParseSkillCatalogYaml(updated, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Category).To(HaveValue(Equal("Development")))
			Expect(configs[0].Provider).To(HaveValue(Equal("Matt")))
			Expect(configs[0].TrustTier).To(HaveValue(Equal("communityContributed")))

			// Also confirm the fields sit under the repository and not on properties
			// itself. A substring check cannot tell the two levels apart — the same
			// "category: Development" text appears either way — so inspect the structure.
			var doc struct {
				Catalogs []struct {
					Properties map[string]interface{} `yaml:"properties"`
				} `yaml:"skill_catalogs"`
			}
			Expect(yaml.Unmarshal([]byte(updated), &doc)).To(Succeed())
			Expect(doc.Catalogs).To(HaveLen(1))
			props := doc.Catalogs[0].Properties
			Expect(props).NotTo(HaveKey("category"), "category must not be at properties level")
			Expect(props).NotTo(HaveKey("provider"), "provider must not be at properties level")
			Expect(props).NotTo(HaveKey("trustTier"), "trustTier must not be at properties level")
			Expect(props).To(HaveKey("repositories"))
		})
	})

	Describe("buildSkillRepoYamlEntries", func() {
		// backendRepoKeys mirrors SkillRepository in
		// catalog/internal/catalog/skillcatalog/source_schema.go. That struct is decoded
		// with DisallowUnknownFields, so emitting a key absent from this set does not get
		// ignored — it fails the entire source at load time. Adding a key here without a
		// matching backend field is a breaking change.
		backendRepoKeys := []string{
			"url", "canonicalUrl", "refs", "scanPaths", "credentialRef",
			"trustTier", "provider", "category", "labels",
			"includedSkills", "excludedSkills", "skillOverrides",
		}

		It("emits only keys the catalog service declares", func() {
			payload := models.SkillCatalogSourceConfig{
				Id:        "test-source",
				Provider:  ptr("Matt"),
				Category:  ptr("Development"),
				TrustTier: ptr("communityContributed"),
				Repositories: []models.SkillRepository{{
					URL:            "https://github.com/example/skills.git",
					CanonicalURL:   ptr("https://github.com/upstream/skills.git"),
					Refs:           []string{"v1.2.3"},
					ScanPaths:      []string{"skills/"},
					CredentialRef:  ptr("test-source"),
					IncludedSkills: []string{"a*"},
					ExcludedSkills: []string{"*-draft"},
				}},
			}

			repos := buildSkillRepoYamlEntries(payload)
			Expect(repos).To(HaveLen(1))
			for key := range repos[0] {
				Expect(backendRepoKeys).To(ContainElement(key),
					"key %q is not declared by the backend SkillRepository struct and will fail "+
						"its DisallowUnknownFields decode, breaking the whole source", key)
			}
		})

		It("writes the credential as credentialRef and never leaks a token", func() {
			payload := models.SkillCatalogSourceConfig{
				Id: "test-source",
				Repositories: []models.SkillRepository{{
					URL:           "https://github.com/example/skills.git",
					CredentialRef: ptr("test-source"),
					AuthToken:     ptr("ghp_supersecret"),
				}},
			}

			repos := buildSkillRepoYamlEntries(payload)
			Expect(repos[0]).To(HaveKeyWithValue("credentialRef", "test-source"))
			Expect(repos[0]).NotTo(HaveKey("authToken"), "the raw token must never reach the ConfigMap")
			Expect(repos[0]).NotTo(HaveKey("authSecretName"), "authSecretName is not a backend field")
		})

		It("preserves every repository field, not just url and refs", func() {
			// Regression guard: the default-source override path previously emitted only
			// url and refs, silently dropping the rest when a default source was edited.
			payload := models.SkillCatalogSourceConfig{
				Id:       "test-source",
				Provider: ptr("Matt"),
				Repositories: []models.SkillRepository{{
					URL:            "https://github.com/example/skills.git",
					Refs:           []string{"v1.2.3"},
					ScanPaths:      []string{"skills/"},
					IncludedSkills: []string{"a*"},
					ExcludedSkills: []string{"*-draft"},
					CredentialRef:  ptr("test-source"),
				}},
			}

			repos := buildSkillRepoYamlEntries(payload)
			Expect(repos[0]).To(HaveKeyWithValue("scanPaths", []string{"skills/"}))
			Expect(repos[0]).To(HaveKeyWithValue("includedSkills", []string{"a*"}))
			Expect(repos[0]).To(HaveKeyWithValue("excludedSkills", []string{"*-draft"}))
			Expect(repos[0]).To(HaveKeyWithValue("credentialRef", "test-source"))
			Expect(repos[0]).To(HaveKeyWithValue("provider", "Matt"))
		})
	})

	Describe("repository labels", func() {
		// Repo labels are stamped onto every skill the repository yields
		// (catalog/internal/catalog/skillcatalog/entity_builder.go). They are a
		// different field from the source-level labels that drive sourceLabel
		// grouping, and both have to survive a round trip independently.
		It("round-trips repository labels through write then read", func() {
			payload := models.SkillCatalogSourceConfig{
				Id:     "test-source",
				Name:   "Test Source",
				Type:   SkillCatalogTypeGit,
				Labels: []string{"source-level"},
				Repositories: []models.SkillRepository{{
					URL:    "https://github.com/example/skills.git",
					Labels: []string{"community", "typescript"},
				}},
			}
			yamlStr, err := AppendSkillCatalogSourceToYaml("", ConvertSkillSourceConfigToYamlEntry(payload))
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(yamlStr, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Repositories[0].Labels).To(Equal([]string{"community", "typescript"}))
			Expect(configs[0].Labels).To(Equal([]string{"source-level"}),
				"source-level labels must not be overwritten by repository labels")
		})

		It("writes repository labels inside the repository entry, not at properties level", func() {
			payload := models.SkillCatalogSourceConfig{
				Id:   "test-source",
				Type: SkillCatalogTypeGit,
				Repositories: []models.SkillRepository{{
					URL:    "https://github.com/example/skills.git",
					Labels: []string{"community"},
				}},
			}
			entry := ConvertSkillSourceConfigToYamlEntry(payload)
			props, ok := entry["properties"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(props).NotTo(HaveKey("labels"))

			repos, ok := props["repositories"].([]map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(repos[0]).To(HaveKeyWithValue("labels", []string{"community"}))
		})

		It("clears repository labels when an update omits them", func() {
			existing := `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    properties:
      repositories:
        - url: https://github.com/example/skills.git
          labels: [community]
`
			payload := models.SkillCatalogSourceConfig{
				Repositories: []models.SkillRepository{{URL: "https://github.com/example/skills.git"}},
			}
			updated, err := UpdateSkillCatalogSourceInYAML(existing, "test-source", payload)
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(updated, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Repositories[0].Labels).To(BeEmpty())
		})
	})

	Describe("source label updates", func() {
		const existing = `
skill_catalogs:
  - id: test-source
    name: Test Source
    type: git-skills-plugin
    labels: [alpha, beta]
    properties:
      repositories:
        - url: https://github.com/example/skills.git
`

		It("clears source labels when given an empty (non-nil) list", func() {
			// Regression guard: the old `len(payload.Labels) > 0` check made removing
			// every label a no-op, which also meant the ConfigMap came back
			// byte-identical and the catalog never reloaded.
			payload := models.SkillCatalogSourceConfig{Labels: []string{}}
			updated, err := UpdateSkillCatalogSourceInYAML(existing, "test-source", payload)
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(updated, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Labels).To(BeEmpty())
		})

		It("leaves source labels alone when they are not sent at all", func() {
			// A toggle-only update must not wipe labels it was never given.
			enabled := false
			payload := models.SkillCatalogSourceConfig{Enabled: &enabled}
			updated, err := UpdateSkillCatalogSourceInYAML(existing, "test-source", payload)
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(updated, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Labels).To(Equal([]string{"alpha", "beta"}))
			Expect(configs[0].Enabled).To(HaveValue(BeFalse()))
		})

		It("replaces source labels when given a new list", func() {
			payload := models.SkillCatalogSourceConfig{Labels: []string{"gamma"}}
			updated, err := UpdateSkillCatalogSourceInYAML(existing, "test-source", payload)
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(updated, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Labels).To(Equal([]string{"gamma"}))
		})
	})

	Describe("default-source label overrides", func() {
		It("keeps an empty label list distinguishable from an absent one after a YAML round trip", func() {
			// The default-override path writes `labels: []` to mean "clear the
			// default's labels". That only works if the value survives as a non-nil
			// empty slice, because mergeSkillCatalogSourceConfigs branches on nil.
			yamlStr, err := AppendSkillCatalogSourceToYaml("", map[string]interface{}{
				"id":     "test-source",
				"labels": []string{},
			})
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(yamlStr, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Labels).NotTo(BeNil(), "an explicit clear must not decode as nil")
			Expect(configs[0].Labels).To(BeEmpty())

			merged := mergeSkillCatalogSourceConfigs(
				models.SkillCatalogSourceConfig{Id: "test-source", Labels: []string{"shipped"}},
				configs[0],
			)
			Expect(merged.Labels).To(BeEmpty())
		})
	})

	Describe("mergeSkillCatalogSourceConfigs", func() {
		It("lets a user override clear a default source's labels", func() {
			merged := mergeSkillCatalogSourceConfigs(
				models.SkillCatalogSourceConfig{Id: "s", Labels: []string{"shipped"}},
				models.SkillCatalogSourceConfig{Id: "s", Labels: []string{}},
			)
			Expect(merged.Labels).To(BeEmpty())
		})

		It("keeps the default's labels when the override does not mention them", func() {
			merged := mergeSkillCatalogSourceConfigs(
				models.SkillCatalogSourceConfig{Id: "s", Labels: []string{"shipped"}},
				models.SkillCatalogSourceConfig{Id: "s"},
			)
			Expect(merged.Labels).To(Equal([]string{"shipped"}))
		})
	})

	Describe("validateSkillRepositories", func() {
		It("accepts a credentialRef that is a plain filename", func() {
			Expect(validateSkillRepositories([]models.SkillRepository{
				{URL: "https://github.com/example/skills.git", CredentialRef: ptr("matt-pocock-skills")},
			})).To(Succeed())
		})
		It("rejects a credentialRef containing a path separator", func() {
			Expect(validateSkillRepositories([]models.SkillRepository{
				{URL: "https://github.com/example/skills.git", CredentialRef: ptr("../../etc/passwd")},
			})).To(MatchError(ContainSubstring("must be a plain filename")))
		})
		It("still requires a repository url", func() {
			Expect(validateSkillRepositories([]models.SkillRepository{{URL: ""}})).
				To(MatchError(ContainSubstring("url is required")))
		})
	})

	Describe("credentialRef round-trip", func() {
		It("survives write then read", func() {
			payload := models.SkillCatalogSourceConfig{
				Id:   "test-source",
				Name: "Test Source",
				Type: SkillCatalogTypeGit,
				Repositories: []models.SkillRepository{{
					URL:           "https://github.com/example/skills.git",
					CredentialRef: ptr("test-source"),
				}},
			}
			yamlStr, err := AppendSkillCatalogSourceToYaml("", ConvertSkillSourceConfigToYamlEntry(payload))
			Expect(err).NotTo(HaveOccurred())

			configs, err := ParseSkillCatalogYaml(yamlStr, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(configs[0].Repositories[0].CredentialRef).To(HaveValue(Equal("test-source")))
		})
	})
})

var _ = Describe("skillOverrides round trip", func() {
	// The settings UI has no control for skillOverrides, so an edit sends back whatever
	// the previous read returned. The write path rebuilds properties.repositories from
	// scratch, so anything it fails to emit is deleted rather than left alone.
	const sourceYAML = `
skill_catalogs:
  - id: acme_skills
    name: Acme Skills
    type: git-skills-plugin
    enabled: true
    properties:
      repositories:
        - url: https://github.com/acme/skills.git
          category: DevOps
          skillOverrides:
            - name: deploy
              category: SRE
              labels: [ops, oncall]
            - name: lint
              category: QA
`

	It("reads skillOverrides off a repository entry", func() {
		parsed, err := ParseSkillCatalogYaml(sourceYAML, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsed).To(HaveLen(1))
		Expect(parsed[0].Repositories).To(HaveLen(1))

		overrides := parsed[0].Repositories[0].SkillOverrides
		Expect(overrides).To(HaveLen(2))
		Expect(overrides[0].Name).To(Equal("deploy"))
		Expect(*overrides[0].Category).To(Equal("SRE"))
		Expect(overrides[0].Labels).To(Equal([]string{"ops", "oncall"}))
		Expect(overrides[1].Name).To(Equal("lint"))
		Expect(overrides[1].Labels).To(BeEmpty())
	})

	It("preserves them through a save that does not touch them", func() {
		parsed, err := ParseSkillCatalogYaml(sourceYAML, false)
		Expect(err).NotTo(HaveOccurred())

		// Feed the parsed config straight back, the way an unrelated UI edit would.
		updated, err := UpdateSkillCatalogSourceInYAML(sourceYAML, "acme_skills", parsed[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(ContainSubstring("skillOverrides"))

		reparsed, err := ParseSkillCatalogYaml(updated, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(reparsed).To(HaveLen(1))
		Expect(reparsed[0].Repositories[0].SkillOverrides).
			To(Equal(parsed[0].Repositories[0].SkillOverrides))
	})

	It("omits the key entirely when a source has no overrides", func() {
		payload := models.SkillCatalogSourceConfigPayload{
			Id:   "plain_source",
			Name: "Plain",
			Type: SkillCatalogTypeGit,
			Repositories: []models.SkillRepository{
				{URL: "https://github.com/acme/plain.git"},
			},
		}
		entry := ConvertSkillSourceConfigToYamlEntry(payload)
		out, err := AppendSkillCatalogSourceToYaml("", entry)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).NotTo(ContainSubstring("skillOverrides"))
	})
})

var _ = Describe("default source override round trip", func() {
	// Toggling a shipped default off writes an override entry carrying only {id, enabled}.
	// Toggling it back on then takes the user-source update path. That path must not emit
	// `properties: {}`: MergeCommonSourceFields replaces the base source's Properties
	// whenever the override's is non-nil, so an empty map strips the default's
	// repositories and the source stops loading entirely.
	It("does not write an empty properties map when the override has none", func() {
		existing := "skill_catalogs:\n    - enabled: false\n      id: community_skills\n"

		updated, err := UpdateSkillCatalogSourceInYAML(existing, "community_skills",
			models.SkillCatalogSourceConfigPayload{Enabled: skillBoolPtr(true)})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(ContainSubstring("properties"))

		reparsed, err := ParseSkillCatalogYaml(updated, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(reparsed).To(HaveLen(1))
		Expect(*reparsed[0].Enabled).To(BeTrue())
	})

	It("still writes properties when the update carries repositories", func() {
		existing := "skill_catalogs:\n    - enabled: true\n      id: custom\n"

		updated, err := UpdateSkillCatalogSourceInYAML(existing, "custom",
			models.SkillCatalogSourceConfigPayload{
				Repositories: []models.SkillRepository{{URL: "https://github.com/example/skills"}},
			})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(ContainSubstring("repositories"))
	})
})
