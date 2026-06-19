package main

import (
	"strings"
	"unicode"
)

type PluginConfig struct {
	Name        string
	Description string
	Entities    []EntityConfig
	RootDir     string
	DryRun      bool
}

type EntityConfig struct {
	Name          string // PascalCase, e.g. CatalogDataset
	DatastoreType string // context, artifact, execution
	TypeName      string // e.g. kf.CatalogDataset
}

func (c *PluginConfig) PascalName() string {
	return toPascalCase(c.Name)
}

func (c *PluginConfig) CamelName() string {
	p := c.PascalName()
	if p == "" {
		return ""
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func (c *PluginConfig) UpperName() string {
	return strings.ToUpper(c.Name)
}

func (c *PluginConfig) CatalogPkg() string {
	return c.Name + "catalog"
}

func (c *PluginConfig) BasePath() string {
	return "/api/" + c.Name + "_catalog/v1alpha1"
}

func (c *PluginConfig) AssetTypeValue() string {
	return c.PrimaryEntity().SnakeName() + "s"
}

func (c *PluginConfig) AssetTypeConstSuffix() string {
	return strings.ToUpper(c.AssetTypeValue())
}

func (c *PluginConfig) AssetTypePascal() string {
	return toPascalCase(c.AssetTypeValue())
}

func (c *PluginConfig) SourceStructName() string {
	return c.PascalName() + "Source"
}

func (c *PluginConfig) SourceConfigField() string {
	return c.PascalName() + "Catalogs"
}

func (c *PluginConfig) SourceYAMLKey() string {
	return c.Name + "_catalogs"
}

func (c *PluginConfig) ContextEntities() []EntityConfig {
	var result []EntityConfig
	for _, e := range c.Entities {
		if e.DatastoreType == "context" {
			result = append(result, e)
		}
	}
	return result
}

func (c *PluginConfig) ArtifactEntities() []EntityConfig {
	var result []EntityConfig
	for _, e := range c.Entities {
		if e.DatastoreType == "artifact" {
			result = append(result, e)
		}
	}
	return result
}

func (c *PluginConfig) ExecutionEntities() []EntityConfig {
	var result []EntityConfig
	for _, e := range c.Entities {
		if e.DatastoreType == "execution" {
			result = append(result, e)
		}
	}
	return result
}

func (c *PluginConfig) PrimaryEntity() EntityConfig {
	for _, e := range c.Entities {
		if e.DatastoreType == "context" {
			return e
		}
	}
	return c.Entities[0]
}

func (e EntityConfig) SnakeName() string {
	return toSnakeCase(e.Name)
}

func (e EntityConfig) CamelName() string {
	if e.Name == "" {
		return ""
	}
	return strings.ToLower(e.Name[:1]) + e.Name[1:]
}

func (e EntityConfig) RepoInterface() string {
	return e.Name + "Repository"
}

func (e EntityConfig) RepoImpl() string {
	return e.Name + "RepositoryImpl"
}

func (e EntityConfig) ListOptions() string {
	return e.Name + "ListOptions"
}

func (e EntityConfig) Attributes() string {
	return e.Name + "Attributes"
}

func (e EntityConfig) TypeConstName() string {
	return e.Name + "TypeName"
}

func (e EntityConfig) SchemaType() string {
	switch e.DatastoreType {
	case "context":
		return "schema.Context"
	case "artifact":
		return "schema.Artifact"
	case "execution":
		return "schema.Execution"
	default:
		return "schema.Context"
	}
}

func (e EntityConfig) SchemaPropertyType() string {
	switch e.DatastoreType {
	case "context":
		return "schema.ContextProperty"
	case "artifact":
		return "schema.ArtifactProperty"
	case "execution":
		return "schema.ExecutionProperty"
	default:
		return "schema.ContextProperty"
	}
}

func (e EntityConfig) PropertyFieldName() string {
	switch e.DatastoreType {
	case "context":
		return "context_id"
	case "artifact":
		return "artifact_id"
	case "execution":
		return "execution_id"
	default:
		return "context_id"
	}
}

func (e EntityConfig) DatastoreAddMethod() string {
	switch e.DatastoreType {
	case "context":
		return "AddContext"
	case "artifact":
		return "AddArtifact"
	case "execution":
		return "AddExecution"
	default:
		return "AddContext"
	}
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// toSnakeCase converts PascalCase/camelCase to snake_case.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
