package model

import "github.com/kubeflow/hub/catalog/internal/plugin"

func init() {
	plugin.Register(&Plugin{})
}
