package skill

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"

	"github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog"
)

// MarketplaceHandler renders the indexed skills as a Claude Code plugin
// marketplace document. The optional `audience` query parameter selects the clone
// URL base for git-subdir sources ("cluster" for in-cluster consumers); see
// MarketplaceConfig.Options.
func MarketplaceHandler(provider *skillcatalog.DBSkillCatalog, cfg skillcatalog.MarketplaceConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		skills, err := provider.ListAllSkills(r.Context())
		if err != nil {
			glog.Errorf("skill marketplace: listing skills: %v", err)
			http.Error(w, "failed to list skills", http.StatusInternalServerError)
			return
		}

		doc := skillcatalog.BuildMarketplace(skills, cfg.Options(r.URL.Query().Get("audience")))

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(doc); err != nil {
			// Headers/body may be partially written; log rather than double-write.
			glog.Errorf("skill marketplace: encoding response: %v", err)
		}
	}
}
