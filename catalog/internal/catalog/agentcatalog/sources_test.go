package agentcatalog

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentSourceCollection_NamedQueries(t *testing.T) {
	sc := NewAgentSourceCollection()

	sources := map[string]basecatalog.PluginSource{
		"test1": {ID: "test1"},
	}
	namedQueries := map[string]map[string]basecatalog.FieldFilter{
		"validation-default": {
			"ttft_p90": {Operator: "<", Value: 70},
		},
	}

	err := sc.MergeWithNamedQueries("origin1", sources, namedQueries)
	if err != nil {
		t.Fatalf("MergeWithNamedQueries failed: %v", err)
	}

	queries := sc.GetNamedQueries()
	if len(queries) != 1 {
		t.Errorf("GetNamedQueries() returned %d queries, want 1", len(queries))
	}
	if queries["validation-default"]["ttft_p90"].Operator != "<" {
		t.Errorf("Expected operator '<', got '%s'", queries["validation-default"]["ttft_p90"].Operator)
	}
	if queries["validation-default"]["ttft_p90"].Value != 70 {
		t.Errorf("Expected value 70, got %v", queries["validation-default"]["ttft_p90"].Value)
	}
}

func TestAgentSourceCollection_NamedQueriesClearedOnMerge(t *testing.T) {
	t.Run("Merge clears named queries from same origin", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{
			"src1": {ID: "src1", Name: "Source 1"},
		}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"validation-default": {
				"ttft_p90": {Operator: "<", Value: 70},
			},
		}

		// First, merge with named queries
		err := sc.MergeWithNamedQueries("origin1", sources, queries)
		if err != nil {
			t.Fatalf("MergeWithNamedQueries failed: %v", err)
		}
		if len(sc.GetNamedQueries()) != 1 {
			t.Fatalf("Expected 1 named query, got %d", len(sc.GetNamedQueries()))
		}

		// Now re-merge the same origin without named queries (simulates config update)
		err = sc.Merge("origin1", sources)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		// Named queries from origin1 should be cleared
		result := sc.GetNamedQueries()
		if len(result) != 0 {
			t.Errorf("Expected 0 named queries after Merge, got %d", len(result))
		}
	})

	t.Run("Merge only clears named queries from its own origin", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{}

		// Origin A contributes a named query
		queriesA := map[string]map[string]basecatalog.FieldFilter{
			"queryA": {"field": {Operator: "=", Value: "a"}},
		}
		if err := sc.MergeWithNamedQueries("originA", sources, queriesA); err != nil {
			t.Fatalf("MergeWithNamedQueries(originA) failed: %v", err)
		}

		// Origin B contributes a different named query
		queriesB := map[string]map[string]basecatalog.FieldFilter{
			"queryB": {"field": {Operator: "=", Value: "b"}},
		}
		if err := sc.MergeWithNamedQueries("originB", sources, queriesB); err != nil {
			t.Fatalf("MergeWithNamedQueries(originB) failed: %v", err)
		}

		if len(sc.GetNamedQueries()) != 2 {
			t.Fatalf("Expected 2 named queries, got %d", len(sc.GetNamedQueries()))
		}

		// Re-merge originA without named queries
		if err := sc.Merge("originA", sources); err != nil {
			t.Fatalf("Merge(originA) failed: %v", err)
		}

		// Only queryA should be removed; queryB should remain
		result := sc.GetNamedQueries()
		if len(result) != 1 {
			t.Errorf("Expected 1 named query, got %d", len(result))
		}
		if _, ok := result["queryB"]; !ok {
			t.Error("queryB should still be present")
		}
		if _, ok := result["queryA"]; ok {
			t.Error("queryA should have been cleared")
		}
	})

	t.Run("MergeWithNamedQueries replaces previous named queries from same origin", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{}

		// Origin A contributes initial named queries
		queriesV1 := map[string]map[string]basecatalog.FieldFilter{
			"old_query": {"field": {Operator: "=", Value: "old"}},
		}
		if err := sc.MergeWithNamedQueries("originA", sources, queriesV1); err != nil {
			t.Fatalf("MergeWithNamedQueries failed: %v", err)
		}
		if _, ok := sc.GetNamedQueries()["old_query"]; !ok {
			t.Fatal("old_query should be present")
		}

		// Origin A re-parses with different named queries
		queriesV2 := map[string]map[string]basecatalog.FieldFilter{
			"new_query": {"field": {Operator: "=", Value: "new"}},
		}
		if err := sc.MergeWithNamedQueries("originA", sources, queriesV2); err != nil {
			t.Fatalf("MergeWithNamedQueries failed: %v", err)
		}

		// old_query should be gone, new_query should be present
		result := sc.GetNamedQueries()
		if len(result) != 1 {
			t.Errorf("Expected 1 named query, got %d", len(result))
		}
		if _, ok := result["old_query"]; ok {
			t.Error("old_query should have been replaced")
		}
		if _, ok := result["new_query"]; !ok {
			t.Error("new_query should be present")
		}
	})

	t.Run("cross-origin field-level merge still works after clear", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{}

		// Origin A sets field "rating" on query "quality"
		queriesA := map[string]map[string]basecatalog.FieldFilter{
			"quality": {"rating": {Operator: ">=", Value: 3}},
		}
		if err := sc.MergeWithNamedQueries("originA", sources, queriesA); err != nil {
			t.Fatalf("MergeWithNamedQueries(originA) failed: %v", err)
		}

		// Origin B sets field "verified" on query "quality"
		queriesB := map[string]map[string]basecatalog.FieldFilter{
			"quality": {"verified": {Operator: "=", Value: true}},
		}
		if err := sc.MergeWithNamedQueries("originB", sources, queriesB); err != nil {
			t.Fatalf("MergeWithNamedQueries(originB) failed: %v", err)
		}

		// Both fields should be present
		result := sc.GetNamedQueries()
		if len(result["quality"]) != 2 {
			t.Fatalf("Expected 2 fields in quality query, got %d", len(result["quality"]))
		}

		// Clear origin A's queries
		if err := sc.Merge("originA", sources); err != nil {
			t.Fatalf("Merge(originA) failed: %v", err)
		}

		// Only origin B's "verified" field should remain
		result = sc.GetNamedQueries()
		if len(result) != 1 {
			t.Errorf("Expected 1 named query, got %d", len(result))
		}
		if len(result["quality"]) != 1 {
			t.Errorf("Expected 1 field in quality query, got %d", len(result["quality"]))
		}
		if _, ok := result["quality"]["verified"]; !ok {
			t.Error("verified field should still be present")
		}
		if _, ok := result["quality"]["rating"]; ok {
			t.Error("rating field should have been cleared")
		}
	})
}

func TestAgentSourceCollection_MergeWithNamedQueries_InputMutationIsolation(t *testing.T) {
	t.Run("mutating input map after MergeWithNamedQueries does not affect stored state", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"my_query": {
				"status": {Operator: "=", Value: "active"},
			},
		}

		require.NoError(t, sc.MergeWithNamedQueries("origin", sources, queries))

		// Mutate the input maps after merge
		queries["my_query"]["status"] = basecatalog.FieldFilter{Operator: "!=", Value: "mutated"}
		queries["injected"] = map[string]basecatalog.FieldFilter{
			"field": {Operator: "=", Value: "injected"},
		}

		// Internal state must be unchanged
		result := sc.GetNamedQueries()
		require.Len(t, result, 1)
		assert.NotContains(t, result, "injected")
		assert.Equal(t, "=", result["my_query"]["status"].Operator)
		assert.Equal(t, "active", result["my_query"]["status"].Value)
	})

	t.Run("mutating input slice Value after MergeWithNamedQueries does not affect stored state", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		sources := map[string]basecatalog.PluginSource{}
		sliceVal := []any{"active", "beta"}
		queries := map[string]map[string]basecatalog.FieldFilter{
			"in_query": {
				"status": {Operator: "IN", Value: sliceVal},
			},
		}

		require.NoError(t, sc.MergeWithNamedQueries("origin", sources, queries))

		// Mutate the original slice after merge
		sliceVal[0] = "mutated"

		// Internal state must be unchanged
		result := sc.GetNamedQueries()
		assert.Equal(t, "active", result["in_query"]["status"].Value.([]any)[0])
	})

	t.Run("GetNamedQueries returns deep copy including slice Values", func(t *testing.T) {
		sc := NewAgentSourceCollection()
		queries := map[string]map[string]basecatalog.FieldFilter{
			"in_query": {"status": {Operator: "IN", Value: []any{"active", "beta"}}},
		}
		require.NoError(t, sc.MergeWithNamedQueries("origin", map[string]basecatalog.PluginSource{}, queries))

		// Mutate the slice returned by GetNamedQueries
		result := sc.GetNamedQueries()
		sliceVal := result["in_query"]["status"].Value.([]any)
		sliceVal[0] = "mutated"

		// Internal state must be unchanged
		result2 := sc.GetNamedQueries()
		assert.Equal(t, "active", result2["in_query"]["status"].Value.([]any)[0])
	})
}
