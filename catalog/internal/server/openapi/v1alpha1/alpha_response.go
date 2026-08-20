package v1alpha1

import (
	"encoding/json"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
)

func renameKeysInVal(val any, keyMap map[string]string) any {
	switch v := val.(type) {
	case map[string]any:
		newMap := make(map[string]any, len(v))
		for k, item := range v {
			newKey := k
			if mappedKey, ok := keyMap[k]; ok {
				newKey = mappedKey
			}
			if slice, ok := item.([]any); ok {
				newMap[newKey] = renameKeysInVal(slice, keyMap)
			} else {
				newMap[newKey] = item
			}
		}
		return newMap
	case []any:
		newList := make([]any, len(v))
		for i, item := range v {
			newList[i] = renameKeysInVal(item, keyMap)
		}
		return newList
	default:
		return v
	}
}

func renameKeys(data []byte, keyMap map[string]string) ([]byte, error) {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	renamed := renameKeysInVal(val, keyMap)
	return json.Marshal(renamed)
}

// AlphaCatalogModel wraps model.CatalogModel for v1alpha1 JSON compatibility (source_id)
type AlphaCatalogModel struct {
	model.CatalogModel
}

func (m AlphaCatalogModel) MarshalJSON() ([]byte, error) {
	data, err := m.CatalogModel.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{"sourceId": "source_id"})
}

func wrapCatalogModel(m *model.CatalogModel) any {
	if m == nil {
		return nil
	}
	return AlphaCatalogModel{*m}
}

func unwrapCatalogModel(v any) (*model.CatalogModel, bool) {
	switch val := v.(type) {
	case *model.CatalogModel:
		return val, true
	case model.CatalogModel:
		return &val, true
	case AlphaCatalogModel:
		return &val.CatalogModel, true
	case *AlphaCatalogModel:
		return &val.CatalogModel, true
	default:
		return nil, false
	}
}

// AlphaCatalogModelList wraps model.CatalogModelList for v1alpha1 JSON compatibility (source_id)
type AlphaCatalogModelList struct {
	model.CatalogModelList
}

func (m AlphaCatalogModelList) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(m.CatalogModelList)
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{"sourceId": "source_id"})
}

func wrapCatalogModelList(l *model.CatalogModelList) any {
	if l == nil {
		return nil
	}
	return AlphaCatalogModelList{*l}
}

func unwrapCatalogModelList(v any) (model.CatalogModelList, bool) {
	switch val := v.(type) {
	case model.CatalogModelList:
		return val, true
	case *model.CatalogModelList:
		return *val, true
	case AlphaCatalogModelList:
		return val.CatalogModelList, true
	case *AlphaCatalogModelList:
		return val.CatalogModelList, true
	default:
		return model.CatalogModelList{}, false
	}
}

// AlphaMCPServer wraps model.MCPServer for v1alpha1 JSON compatibility (source_id, license_link)
type AlphaMCPServer struct {
	model.MCPServer
}

func (m AlphaMCPServer) MarshalJSON() ([]byte, error) {
	data, err := m.MCPServer.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{
		"sourceId":    "source_id",
		"licenseLink": "license_link",
	})
}

func wrapMCPServer(m *model.MCPServer) any {
	if m == nil {
		return nil
	}
	return AlphaMCPServer{*m}
}

func unwrapMCPServer(v any) (*model.MCPServer, bool) {
	switch val := v.(type) {
	case *model.MCPServer:
		return val, true
	case model.MCPServer:
		return &val, true
	case AlphaMCPServer:
		return &val.MCPServer, true
	case *AlphaMCPServer:
		return &val.MCPServer, true
	default:
		return nil, false
	}
}

// AlphaMCPServerList wraps model.MCPServerList for v1alpha1 JSON compatibility (source_id, license_link)
type AlphaMCPServerList struct {
	model.MCPServerList
}

func (m AlphaMCPServerList) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(m.MCPServerList)
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{
		"sourceId":    "source_id",
		"licenseLink": "license_link",
	})
}

func wrapMCPServerList(l *model.MCPServerList) any {
	if l == nil {
		return nil
	}
	return AlphaMCPServerList{*l}
}

func unwrapMCPServerList(v any) (model.MCPServerList, bool) {
	switch val := v.(type) {
	case model.MCPServerList:
		return val, true
	case *model.MCPServerList:
		return *val, true
	case AlphaMCPServerList:
		return val.MCPServerList, true
	case *AlphaMCPServerList:
		return val.MCPServerList, true
	default:
		return model.MCPServerList{}, false
	}
}

// AlphaAgent wraps model.Agent for v1alpha1 JSON compatibility (source_id)
type AlphaAgent struct {
	model.Agent
}

func (m AlphaAgent) MarshalJSON() ([]byte, error) {
	data, err := m.Agent.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{"sourceId": "source_id"})
}

func wrapAgent(m *model.Agent) any {
	if m == nil {
		return nil
	}
	return AlphaAgent{*m}
}

// AlphaAgentList wraps model.AgentList for v1alpha1 JSON compatibility (source_id)
type AlphaAgentList struct {
	model.AgentList
}

func (m AlphaAgentList) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(m.AgentList)
	if err != nil {
		return nil, err
	}
	return renameKeys(data, map[string]string{"sourceId": "source_id"})
}

func wrapAgentList(l *model.AgentList) any {
	if l == nil {
		return nil
	}
	return AlphaAgentList{*l}
}
