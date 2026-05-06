package plugin

import (
	"fmt"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

// LoadConfig loads and parses a sources.yaml file using basecatalog's parser.
func LoadConfig(path string) (*basecatalog.SourceConfig, error) {
	return basecatalog.ReadSourceConfig(path)
}

// LoadConfigs loads multiple sources.yaml files and returns them as
// independent configs. Callers are responsible for any merge logic
// (e.g., basecatalog's SourceCollection handles field-level merging).
func LoadConfigs(paths []string) ([]*basecatalog.SourceConfig, error) {
	configs := make([]*basecatalog.SourceConfig, 0, len(paths))
	for _, path := range paths {
		cfg, err := LoadConfig(path)
		if err != nil {
			return nil, fmt.Errorf("loading config %s: %w", path, err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}
