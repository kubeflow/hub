package plugin

import (
	"fmt"
	"reflect"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/kubeflow/hub/internal/platform/datastore"
)

// GetRepo extracts a typed repository from a RepoSet.
// Panics if the repository type is not found — this indicates
// a programming error in the datastore spec, not a runtime condition.
func GetRepo[T any](repoSet datastore.RepoSet) T {
	repo, err := repoSet.Repository(reflect.TypeFor[T]())
	if err != nil {
		panic(fmt.Sprintf("unable to get repository: %v", err))
	}
	return repo.(T)
}

// CollectAllSourceIDs queries all registered plugins that implement
// SourceIDProvider and returns the union of their known source IDs.
func CollectAllSourceIDs() mapset.Set[string] {
	combined := mapset.NewSet[string]()
	for _, p := range All() {
		if sp, ok := p.(SourceIDProvider); ok {
			combined = combined.Union(sp.KnownSourceIDs())
		}
	}
	return combined
}
