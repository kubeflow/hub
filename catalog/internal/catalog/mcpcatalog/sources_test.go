package mcpcatalog

import (
	"reflect"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPSourceCollection_MergeOverride(t *testing.T) {
	tests := []struct {
		name          string
		originOrder   []string
		mergeSequence []struct {
			origin  string
			sources map[string]basecatalog.MCPSource
		}
		expectedSources map[string]basecatalog.MCPSource
	}{
		{
			name:        "later origin overrides earlier origin",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"community"},
							Properties: map[string]any{
								"url": "https://community.example.com/github",
							},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server Enterprise",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"enterprise"},
							Properties: map[string]any{
								"url": "https://org.example.com/github",
							},
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server Enterprise",
					Type:    "sse",
					Enabled: new(true),
					Labels:  []string{"enterprise"},
					Properties: map[string]any{
						"url": "https://org.example.com/github",
					},
				},
			},
		},
		{
			name:        "single origin no merge needed",
			originOrder: []string{"community.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_slack": {
							ID:      "mcp_slack",
							Name:    "Slack MCP Server",
							Type:    "stdio",
							Enabled: new(true),
							Labels:  []string{"community"},
							Properties: map[string]any{
								"command": "slack-mcp",
							},
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_slack": {
					ID:      "mcp_slack",
					Name:    "Slack MCP Server",
					Type:    "stdio",
					Enabled: new(true),
					Labels:  []string{"community"},
					Properties: map[string]any{
						"command": "slack-mcp",
					},
				},
			},
		},
		{
			name:        "multiple sources only one overridden",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"community"},
						},
						"mcp_slack": {
							ID:      "mcp_slack",
							Name:    "Slack MCP Server",
							Type:    "stdio",
							Enabled: new(true),
							Labels:  []string{"community"},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server Enterprise",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"enterprise"},
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server Enterprise",
					Type:    "sse",
					Enabled: new(true),
					Labels:  []string{"enterprise"},
				},
				"mcp_slack": {
					ID:      "mcp_slack",
					Name:    "Slack MCP Server",
					Type:    "stdio",
					Enabled: new(true),
					Labels:  []string{"community"},
				},
			},
		},
		{
			name:        "three-origin cascading override community to org to team",
			originOrder: []string{"community.yaml", "org.yaml", "team.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "Community GitHub",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"community"},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "Org GitHub",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"org"},
						},
					},
				},
				{
					origin: "team.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "Team GitHub",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"team"},
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "Team GitHub",
					Type:    "sse",
					Enabled: new(true),
					Labels:  []string{"team"},
				},
			},
		},
		{
			name:        "override disables an enabled source",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(false), // Disable it
							Labels:  []string{},
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server",
					Type:    "sse",
					Enabled: new(false),
					Labels:  []string{},
				},
			},
		},
		{
			name:        "sparse override: only id and enabled set, all other fields inherited",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(false), // Disabled in base
							Labels:  []string{"community", "certified"},
							Properties: map[string]any{
								"url": "https://community.example.com/github",
							},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Enabled: new(true), // Only enable it
							// Name, Type, Labels, Properties all empty/nil -> inherited
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server",                // Inherited
					Type:    "sse",                              // Inherited
					Enabled: new(true),                          // Overridden
					Labels:  []string{"community", "certified"}, // Inherited
					Properties: map[string]any{ // Inherited
						"url": "https://community.example.com/github",
					},
				},
			},
		},
		{
			name:        "sparse override: only labels changed",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"community"},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:     "mcp_github",
							Labels: []string{"enterprise", "validated"}, // Override only labels
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server",                 // Inherited
					Type:    "sse",                               // Inherited
					Enabled: new(true),                           // Inherited
					Labels:  []string{"enterprise", "validated"}, // Overridden
				},
			},
		},
		{
			name:        "sparse override: empty slice clears labels",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Labels:  []string{"community", "public"},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:     "mcp_github",
							Labels: []string{}, // Explicitly clear labels
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server", // Inherited
					Type:    "sse",               // Inherited
					Enabled: new(true),           // Inherited
					Labels:  []string{},          // Overridden to empty
				},
			},
		},
		{
			name:        "defaults applied: nil Enabled defaults to true, nil Labels defaults to empty slice, nil AssetType defaults to mcp_servers",
			originOrder: []string{"community.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:   "mcp_github",
							Name: "GitHub MCP Server",
							Type: "sse",
							// Enabled, Labels, and AssetType are nil - defaults should be applied
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:        "mcp_github",
					Name:      "GitHub MCP Server",
					Type:      "sse",
					Enabled:   new(true),                                // Default applied
					Labels:    []string{},                               // Default applied
					AssetType: model.CATALOGASSETTYPE_MCP_SERVERS.Ptr(), // Default applied
				},
			},
		},
		{
			name:        "type and properties inherited when not overridden",
			originOrder: []string{"community.yaml", "org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(false),
							Labels:  []string{"community"},
							Properties: map[string]any{
								"url":     "https://example.com/mcp",
								"timeout": 30,
							},
						},
					},
				},
				{
					origin: "org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Enabled: new(true), // Only enable it; Type and Properties not set
						},
					},
				},
			},
			expectedSources: map[string]basecatalog.MCPSource{
				"mcp_github": {
					ID:      "mcp_github",
					Name:    "GitHub MCP Server",   // Inherited
					Type:    "sse",                 // Inherited
					Enabled: new(true),             // Overridden
					Labels:  []string{"community"}, // Inherited
					Properties: map[string]any{ // Inherited
						"url":     "https://example.com/mcp",
						"timeout": 30,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msc := NewMCPSourceCollection(tt.originOrder...)

			for _, merge := range tt.mergeSequence {
				err := msc.Merge(merge.origin, merge.sources)
				if err != nil {
					t.Fatalf("Merge(%s) failed: %v", merge.origin, err)
				}
			}

			result := msc.AllSources()

			if len(result) != len(tt.expectedSources) {
				t.Errorf("AllSources() returned %d sources, want %d", len(result), len(tt.expectedSources))
			}

			for id, expected := range tt.expectedSources {
				got, ok := result[id]
				if !ok {
					t.Errorf("source %s not found in result", id)
					continue
				}
				if got.ID != expected.ID {
					t.Errorf("source %s: ID = %s, want %s", id, got.ID, expected.ID)
				}
				if got.Name != expected.Name {
					t.Errorf("source %s: Name = %s, want %s", id, got.Name, expected.Name)
				}
				if got.Type != expected.Type {
					t.Errorf("source %s: Type = %s, want %s", id, got.Type, expected.Type)
				}
				if got.Enabled == nil || expected.Enabled == nil {
					if got.Enabled != expected.Enabled {
						t.Errorf("source %s: Enabled = %v, want %v", id, got.Enabled, expected.Enabled)
					}
				} else if *got.Enabled != *expected.Enabled {
					t.Errorf("source %s: Enabled = %v, want %v", id, *got.Enabled, *expected.Enabled)
				}
				if !reflect.DeepEqual(got.Labels, expected.Labels) {
					t.Errorf("source %s: Labels = %v, want %v", id, got.Labels, expected.Labels)
				}
				if !reflect.DeepEqual(got.Properties, expected.Properties) {
					t.Errorf("source %s: Properties = %v, want %v", id, got.Properties, expected.Properties)
				}
				if got.AssetType == nil {
					t.Errorf("source %s: AssetType is nil, expected mcp_servers default", id)
				} else if *got.AssetType != model.CATALOGASSETTYPE_MCP_SERVERS {
					t.Errorf("source %s: AssetType = %s, want %s", id, *got.AssetType, model.CATALOGASSETTYPE_MCP_SERVERS)
				}
			}
		})
	}
}

func TestMCPSourceCollection_MergeOverride_Origin(t *testing.T) {
	tests := []struct {
		name          string
		originOrder   []string
		mergeSequence []struct {
			origin  string
			sources map[string]basecatalog.MCPSource
		}
		expectedOrigins map[string]string // sourceId -> expected origin
	}{
		{
			name:        "origin preserved from base when Properties NOT overridden",
			originOrder: []string{"/config/community.yaml", "/config/org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "/config/community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Properties: map[string]any{
								"url": "https://community.example.com/github",
							},
							Origin: "/config/community.yaml",
						},
					},
				},
				{
					origin: "/config/org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_github": {
							ID:      "mcp_github",
							Name:    "GitHub MCP Server Custom",
							Enabled: new(true),
							// Properties not set -> Origin should stay with base
							Origin: "/config/org.yaml",
						},
					},
				},
			},
			expectedOrigins: map[string]string{
				"mcp_github": "/config/community.yaml", // Base origin preserved
			},
		},
		{
			name:        "origin changes when Properties ARE overridden",
			originOrder: []string{"/config/community.yaml", "/config/org.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "/config/community.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_local": {
							ID:      "mcp_local",
							Name:    "Local MCP Server",
							Type:    "stdio",
							Enabled: new(true),
							Properties: map[string]any{
								"command": "community-mcp",
							},
							Origin: "/config/community.yaml",
						},
					},
				},
				{
					origin: "/config/org.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_local": {
							ID:      "mcp_local",
							Enabled: new(true),
							// Override Properties -> Origin should change
							Properties: map[string]any{
								"command": "org-mcp",
							},
							Origin: "/config/org.yaml",
						},
					},
				},
			},
			expectedOrigins: map[string]string{
				"mcp_local": "/config/org.yaml", // Origin changed because Properties overridden
			},
		},
		{
			name:        "multiple sources from different origins keep their own Origins",
			originOrder: []string{"/admin/sources.yaml", "/user/sources.yaml"},
			mergeSequence: []struct {
				origin  string
				sources map[string]basecatalog.MCPSource
			}{
				{
					origin: "/admin/sources.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_admin": {
							ID:      "mcp_admin",
							Name:    "Admin MCP Server",
							Type:    "sse",
							Enabled: new(true),
							Properties: map[string]any{
								"url": "https://admin.example.com/mcp",
							},
							Origin: "/admin/sources.yaml",
						},
					},
				},
				{
					origin: "/user/sources.yaml",
					sources: map[string]basecatalog.MCPSource{
						"mcp_user": {
							ID:      "mcp_user",
							Name:    "User MCP Server",
							Type:    "stdio",
							Enabled: new(true),
							Properties: map[string]any{
								"command": "user-mcp",
							},
							Origin: "/user/sources.yaml",
						},
					},
				},
			},
			expectedOrigins: map[string]string{
				"mcp_admin": "/admin/sources.yaml",
				"mcp_user":  "/user/sources.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msc := NewMCPSourceCollection(tt.originOrder...)

			for _, merge := range tt.mergeSequence {
				err := msc.Merge(merge.origin, merge.sources)
				if err != nil {
					t.Fatalf("Merge(%s) failed: %v", merge.origin, err)
				}
			}

			sources := msc.AllSources()

			for id, expectedOrigin := range tt.expectedOrigins {
				source, ok := sources[id]
				if !ok {
					t.Errorf("source %s not found in AllSources()", id)
					continue
				}
				if source.Origin != expectedOrigin {
					t.Errorf("source %s: Origin = %s, want %s", id, source.Origin, expectedOrigin)
				}
			}
		})
	}
}

func TestMCPSourceCollection_MergeOverride_DynamicOrigin(t *testing.T) {
	// Test that origins not in the initial originOrder are appended dynamically
	// when Merge() is called with an unknown origin.
	msc := NewMCPSourceCollection("community.yaml")

	err := msc.Merge("community.yaml", map[string]basecatalog.MCPSource{
		"mcp_github": {ID: "mcp_github", Name: "Community", Enabled: new(true), Labels: []string{}},
	})
	if err != nil {
		t.Fatalf("Merge(community.yaml) failed: %v", err)
	}

	// Dynamic origin not in initial order - should be appended and take precedence
	err = msc.Merge("extra.yaml", map[string]basecatalog.MCPSource{
		"mcp_github": {ID: "mcp_github", Name: "Extra Override", Enabled: new(true), Labels: []string{}},
	})
	if err != nil {
		t.Fatalf("Merge(extra.yaml) failed: %v", err)
	}

	result := msc.AllSources()
	source, ok := result["mcp_github"]
	if !ok {
		t.Fatal("AllSources() should return mcp_github")
	}

	if source.Name != "Extra Override" {
		t.Errorf("dynamically added origin should override earlier origins, got Name = %s, want 'Extra Override'", source.Name)
	}
}

func TestMCPSourceCollection_MergeWithNamedQueries(t *testing.T) {
	t.Run("stores and retrieves named queries", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Name: "Source 1", Type: "yaml"},
		}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"production_ready": {
				"verified": {Operator: "=", Value: true},
			},
		}

		err := coll.MergeWithNamedQueries("origin1", sources, queries)
		require.NoError(t, err)

		result := coll.GetNamedQueries()
		require.Len(t, result, 1)
		assert.Equal(t, "=", result["production_ready"]["verified"].Operator)
		assert.Equal(t, true, result["production_ready"]["verified"].Value)
	})

	t.Run("later origin overrides field-level within a query", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}

		queriesA := map[string]map[string]basecatalog.FieldFilter{
			"quality": {
				"rating":   {Operator: ">=", Value: 3},
				"verified": {Operator: "=", Value: false},
			},
		}
		queriesB := map[string]map[string]basecatalog.FieldFilter{
			"quality": {
				"verified": {Operator: "=", Value: true}, // overrides origin A
			},
		}

		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, queriesA))
		require.NoError(t, coll.MergeWithNamedQueries("originB", sources, queriesB))

		result := coll.GetNamedQueries()
		// "rating" from A is still present
		assert.Equal(t, ">=", result["quality"]["rating"].Operator)
		assert.Equal(t, 3, result["quality"]["rating"].Value)
		// "verified" from B overrides A
		assert.Equal(t, true, result["quality"]["verified"].Value)
	})

	t.Run("multiple distinct queries accumulated across origins", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}

		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, map[string]map[string]basecatalog.FieldFilter{
			"queryA": {"field": {Operator: "=", Value: "a"}},
		}))
		require.NoError(t, coll.MergeWithNamedQueries("originB", sources, map[string]map[string]basecatalog.FieldFilter{
			"queryB": {"field": {Operator: "=", Value: "b"}},
		}))

		result := coll.GetNamedQueries()
		assert.Len(t, result, 2)
		assert.Contains(t, result, "queryA")
		assert.Contains(t, result, "queryB")
	})

	t.Run("sources are still merged correctly alongside named queries", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Name: "Source 1", Type: "yaml"},
		}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"test_query": {"field": {Operator: "=", Value: "val"}},
		}

		err := coll.MergeWithNamedQueries("origin1", sources, queries)
		require.NoError(t, err)

		all := coll.AllSources()
		assert.Contains(t, all, "src1")
	})
}

func TestMCPSourceCollection_GetNamedQueries_DeepCopy(t *testing.T) {
	coll := NewMCPSourceCollection()
	queries := map[string]map[string]basecatalog.FieldFilter{
		"my_query": {"field": {Operator: "=", Value: "original"}},
	}
	require.NoError(t, coll.MergeWithNamedQueries("origin", map[string]basecatalog.MCPSource{}, queries))

	// Mutate the returned map
	result := coll.GetNamedQueries()
	result["my_query"]["field"] = basecatalog.FieldFilter{Operator: "!=", Value: "mutated"}
	delete(result, "my_query")

	// Internal state must be unchanged
	result2 := coll.GetNamedQueries()
	require.Len(t, result2, 1)
	assert.Equal(t, "=", result2["my_query"]["field"].Operator)
	assert.Equal(t, "original", result2["my_query"]["field"].Value)
}

func TestMCPSourceCollection_GetNamedQuery(t *testing.T) {
	coll := NewMCPSourceCollection()
	queries := map[string]map[string]basecatalog.FieldFilter{
		"verified_only": {"verified": {Operator: "=", Value: true}},
	}
	require.NoError(t, coll.MergeWithNamedQueries("origin", map[string]basecatalog.MCPSource{}, queries))

	t.Run("returns filters for known query", func(t *testing.T) {
		filters, ok := coll.GetNamedQuery("verified_only")
		assert.True(t, ok)
		require.Len(t, filters, 1)
		assert.Equal(t, "=", filters["verified"].Operator)
	})

	t.Run("returns false for unknown query", func(t *testing.T) {
		filters, ok := coll.GetNamedQuery("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, filters)
	})

	t.Run("returned map is a copy and does not affect internal state", func(t *testing.T) {
		filters, ok := coll.GetNamedQuery("verified_only")
		require.True(t, ok)
		filters["verified"] = basecatalog.FieldFilter{Operator: "!=", Value: false}

		filters2, ok2 := coll.GetNamedQuery("verified_only")
		require.True(t, ok2)
		assert.Equal(t, "=", filters2["verified"].Operator)
	})
}

func TestMCPSourceCollection_GetNamedQuery_SliceDeepCopy(t *testing.T) {
	coll := NewMCPSourceCollection()
	queries := map[string]map[string]basecatalog.FieldFilter{
		"in_query": {"status": {Operator: "IN", Value: []any{"active", "beta"}}},
	}
	require.NoError(t, coll.MergeWithNamedQueries("origin", map[string]basecatalog.MCPSource{}, queries))

	// Mutate the slice returned by GetNamedQuery
	filters, ok := coll.GetNamedQuery("in_query")
	require.True(t, ok)
	sliceVal := filters["status"].Value.([]any)
	sliceVal[0] = "mutated"

	// Internal state must be unchanged
	filters2, ok2 := coll.GetNamedQuery("in_query")
	require.True(t, ok2)
	assert.Equal(t, "active", filters2["status"].Value.([]any)[0])
}

func TestMCPSourceCollection_MergeWithNamedQueries_InputMutationIsolation(t *testing.T) {
	t.Run("mutating input map after MergeWithNamedQueries does not affect stored state", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"my_query": {
				"status": {Operator: "=", Value: "active"},
			},
		}

		require.NoError(t, coll.MergeWithNamedQueries("origin", sources, queries))

		// Mutate the input maps after merge
		queries["my_query"]["status"] = basecatalog.FieldFilter{Operator: "!=", Value: "mutated"}
		queries["injected"] = map[string]basecatalog.FieldFilter{
			"field": {Operator: "=", Value: "injected"},
		}

		// Internal state must be unchanged
		result := coll.GetNamedQueries()
		require.Len(t, result, 1)
		assert.NotContains(t, result, "injected")
		assert.Equal(t, "=", result["my_query"]["status"].Operator)
		assert.Equal(t, "active", result["my_query"]["status"].Value)
	})

	t.Run("mutating input slice Value after MergeWithNamedQueries does not affect stored state", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}
		sliceVal := []any{"active", "beta"}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"in_query": {
				"status": {Operator: "IN", Value: sliceVal},
			},
		}

		require.NoError(t, coll.MergeWithNamedQueries("origin", sources, queries))

		// Mutate the original slice after merge
		sliceVal[0] = "mutated"

		// Internal state must be unchanged
		result := coll.GetNamedQueries()
		assert.Equal(t, "active", result["in_query"]["status"].Value.([]any)[0])
	})
}

func TestMCPSourceCollection_ByLabel(t *testing.T) {
	makeCollection := func(sources map[string]basecatalog.MCPSource) *MCPSourceCollection {
		coll := NewMCPSourceCollection()
		_ = coll.Merge("test", sources)
		return coll
	}

	t.Run("returns source with matching label", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"openai"}, Enabled: new(true)},
			"src2": {ID: "src2", Labels: []string{"github"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"openai"})
		require.Len(t, result, 1)
		assert.Equal(t, "src1", result[0].ID)
	})

	t.Run("returns multiple sources matching any label (OR semantics)", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"openai"}, Enabled: new(true)},
			"src2": {ID: "src2", Labels: []string{"github"}, Enabled: new(true)},
			"src3": {ID: "src3", Labels: []string{"other"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"openai", "github"})
		assert.Len(t, result, 2)
	})

	t.Run("matching is case insensitive", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"OpenAI"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"openai"})
		require.Len(t, result, 1)
		assert.Equal(t, "src1", result[0].ID)
	})

	t.Run("skips disabled sources", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"openai"}, Enabled: new(false)},
		})

		result := coll.ByLabel([]string{"openai"})
		assert.Empty(t, result)
	})

	t.Run("null label matches sources with no labels", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{}, Enabled: new(true)},
			"src2": {ID: "src2", Labels: []string{"openai"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"null"})
		require.Len(t, result, 1)
		assert.Equal(t, "src1", result[0].ID)
	})

	t.Run("non-matching label returns empty slice", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"openai"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"no-match"})
		assert.Empty(t, result)
	})

	t.Run("deduplicates by source ID", func(t *testing.T) {
		coll := makeCollection(map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Labels: []string{"openai", "ai"}, Enabled: new(true)},
		})

		result := coll.ByLabel([]string{"openai", "ai"})
		assert.Len(t, result, 1)
	})
}

func TestMCPSourceCollection_Merge_WithoutNamedQueries(t *testing.T) {
	// Verify the regular Merge path still works after refactoring.
	coll := NewMCPSourceCollection()
	sources := map[string]basecatalog.MCPSource{
		"src1": {ID: "src1", Name: "Source 1", Type: "yaml"},
	}

	err := coll.Merge("origin1", sources)
	require.NoError(t, err)

	all := coll.AllSources()
	assert.Contains(t, all, "src1")
	assert.Empty(t, coll.GetNamedQueries())
}

func TestMCPSourceCollection_NamedQueriesClearedOnMerge(t *testing.T) {
	t.Run("Merge clears named queries from same origin", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{
			"src1": {ID: "src1", Name: "Source 1", Type: "yaml"},
		}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"production_ready": {
				"verified": {Operator: "=", Value: true},
			},
		}

		// First, merge with named queries
		err := coll.MergeWithNamedQueries("origin1", sources, queries)
		require.NoError(t, err)
		require.Len(t, coll.GetNamedQueries(), 1)

		// Now re-merge the same origin without named queries (simulates config update)
		err = coll.Merge("origin1", sources)
		require.NoError(t, err)

		// Named queries from origin1 should be cleared
		assert.Empty(t, coll.GetNamedQueries())
	})

	t.Run("Merge only clears named queries from its own origin", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}

		// Origin A contributes a named query
		queriesA := map[string]map[string]basecatalog.FieldFilter{
			"queryA": {"field": {Operator: "=", Value: "a"}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, queriesA))

		// Origin B contributes a different named query
		queriesB := map[string]map[string]basecatalog.FieldFilter{
			"queryB": {"field": {Operator: "=", Value: "b"}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originB", sources, queriesB))

		require.Len(t, coll.GetNamedQueries(), 2)

		// Re-merge originA without named queries
		require.NoError(t, coll.Merge("originA", sources))

		// Only queryA should be removed; queryB should remain
		result := coll.GetNamedQueries()
		assert.Len(t, result, 1)
		assert.Contains(t, result, "queryB")
		assert.NotContains(t, result, "queryA")
	})

	t.Run("MergeWithNamedQueries replaces previous named queries from same origin", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}

		// Origin A contributes initial named queries
		queriesV1 := map[string]map[string]basecatalog.FieldFilter{
			"old_query": {"field": {Operator: "=", Value: "old"}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, queriesV1))
		require.Contains(t, coll.GetNamedQueries(), "old_query")

		// Origin A re-parses with different named queries
		queriesV2 := map[string]map[string]basecatalog.FieldFilter{
			"new_query": {"field": {Operator: "=", Value: "new"}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, queriesV2))

		// old_query should be gone, new_query should be present
		result := coll.GetNamedQueries()
		assert.Len(t, result, 1)
		assert.NotContains(t, result, "old_query")
		assert.Contains(t, result, "new_query")
	})

	t.Run("cross-origin field-level merge still works after clear", func(t *testing.T) {
		coll := NewMCPSourceCollection()
		sources := map[string]basecatalog.MCPSource{}

		// Origin A sets field "rating" on query "quality"
		queriesA := map[string]map[string]basecatalog.FieldFilter{
			"quality": {"rating": {Operator: ">=", Value: 3}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originA", sources, queriesA))

		// Origin B sets field "verified" on query "quality"
		queriesB := map[string]map[string]basecatalog.FieldFilter{
			"quality": {"verified": {Operator: "=", Value: true}},
		}
		require.NoError(t, coll.MergeWithNamedQueries("originB", sources, queriesB))

		// Both fields should be present
		result := coll.GetNamedQueries()
		assert.Len(t, result["quality"], 2)

		// Clear origin A's queries
		require.NoError(t, coll.Merge("originA", sources))

		// Only origin B's "verified" field should remain
		result = coll.GetNamedQueries()
		require.Len(t, result, 1)
		assert.Len(t, result["quality"], 1)
		assert.Contains(t, result["quality"], "verified")
		assert.NotContains(t, result["quality"], "rating")
	})
}

func TestMergeMCPSources_IncludedExcludedServers(t *testing.T) {
	tests := []struct {
		name         string
		base         basecatalog.MCPSource
		override     basecatalog.MCPSource
		wantIncluded []string
		wantExcluded []string
	}{
		{
			name: "override replaces includedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				IncludedServers: []string{"github*"},
			},
			override: basecatalog.MCPSource{
				ID:              "src",
				IncludedServers: []string{"slack*"},
			},
			wantIncluded: []string{"slack*"},
			wantExcluded: nil,
		},
		{
			name: "override replaces excludedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				ExcludedServers: []string{"internal*"},
			},
			override: basecatalog.MCPSource{
				ID:              "src",
				ExcludedServers: []string{"private*"},
			},
			wantIncluded: nil,
			wantExcluded: []string{"private*"},
		},
		{
			name: "nil override inherits base includedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				IncludedServers: []string{"github*"},
			},
			override: basecatalog.MCPSource{
				ID: "src",
				// IncludedServers nil -> inherit from base
			},
			wantIncluded: []string{"github*"},
			wantExcluded: nil,
		},
		{
			name: "nil override inherits base excludedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				ExcludedServers: []string{"internal*"},
			},
			override: basecatalog.MCPSource{
				ID: "src",
				// ExcludedServers nil -> inherit from base
			},
			wantIncluded: nil,
			wantExcluded: []string{"internal*"},
		},
		{
			name: "empty slice override clears includedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				IncludedServers: []string{"github*"},
			},
			override: basecatalog.MCPSource{
				ID:              "src",
				IncludedServers: []string{}, // explicitly clear
			},
			wantIncluded: []string{},
			wantExcluded: nil,
		},
		{
			name: "empty slice override clears excludedServers",
			base: basecatalog.MCPSource{
				ID:              "src",
				ExcludedServers: []string{"internal*"},
			},
			override: basecatalog.MCPSource{
				ID:              "src",
				ExcludedServers: []string{}, // explicitly clear
			},
			wantIncluded: nil,
			wantExcluded: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMCPSources(tt.base, tt.override)
			assert.Equal(t, tt.wantIncluded, result.IncludedServers)
			assert.Equal(t, tt.wantExcluded, result.ExcludedServers)
		})
	}
}
