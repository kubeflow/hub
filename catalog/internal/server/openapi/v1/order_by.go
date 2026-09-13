package v1

import (
	"fmt"
	"strings"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

// propertyValueColumns are the typed value columns a custom-property orderBy
// may name, matching what the repositories accept.
var propertyValueColumns = []string{"int_value", "double_value", "string_value"}

// orderBySpec describes the orderBy values one list endpoint honours. Anything
// outside the spec is rejected with api.ErrBadRequest, as KEP-0004 requires
// for the v1 APIs, instead of silently sorting by the default column.
type orderBySpec struct {
	// fields are the enum values the endpoint sorts by.
	fields []model.OrderByField
	// customProperty allows "<property>.<value_column>".
	customProperty bool
	// artifactProperty allows "artifacts.<property>.<value_column>".
	artifactProperty bool
	// caseInsensitive is set for endpoints that upper-case the value before use.
	caseInsensitive bool
}

var (
	// enumOrderByFields are the values the catalog's shared OrderByField
	// enum declares. Endpoints that reference that parameter accept all of
	// them; narrowing the enum per plugin is a spec change, not a server one.
	enumOrderByFields = []model.OrderByField{
		model.ORDERBYFIELD_ID,
		model.ORDERBYFIELD_CREATE_TIME,
		model.ORDERBYFIELD_LAST_UPDATE_TIME,
		model.ORDERBYFIELD_NAME,
		model.ORDERBYFIELD_RECOMMENDED,
	}

	// modelOrderBy: FindModels also sorts by a model property, by an
	// artifact property, and by ACCURACY, which the model provider maps to
	// artifacts.overall_average.double_value.
	modelOrderBy = orderBySpec{
		fields:           append(append([]model.OrderByField{}, enumOrderByFields...), "ACCURACY"),
		customProperty:   true,
		artifactProperty: true,
	}

	// enumOrderBy: MCP servers and tools, and agents.
	enumOrderBy = orderBySpec{fields: enumOrderByFields}

	// caseInsensitiveOrderBy: skills and agent artifacts upper-case the value.
	caseInsensitiveOrderBy = orderBySpec{fields: enumOrderByFields, caseInsensitive: true}
)

// validate returns nil when orderBy is empty or honoured by the endpoint, and
// an error wrapping api.ErrBadRequest otherwise.
func (s orderBySpec) validate(orderBy string) error {
	if orderBy == "" {
		return nil
	}

	candidate := orderBy
	if s.caseInsensitive {
		candidate = strings.ToUpper(orderBy)
	}
	for _, f := range s.fields {
		if candidate == string(f) {
			return nil
		}
	}

	parts := strings.Split(orderBy, ".")
	switch {
	case s.customProperty && len(parts) == 2 && validPropertyOrder(parts[0], parts[1]):
		return nil
	case s.artifactProperty && len(parts) == 3 && parts[0] == "artifacts" && validPropertyOrder(parts[1], parts[2]):
		return nil
	}

	return fmt.Errorf("invalid orderBy %q: %s: %w", orderBy, s.describe(), api.ErrBadRequest)
}

// validPropertyOrder mirrors the repositories' checks: a non-empty property
// name of at most 255 characters and one of the typed value columns.
func validPropertyOrder(property, valueColumn string) bool {
	if property == "" || len(property) > 255 {
		return false
	}
	for _, c := range propertyValueColumns {
		if valueColumn == c {
			return true
		}
	}
	return false
}

func (s orderBySpec) describe() string {
	names := make([]string, len(s.fields))
	for i, f := range s.fields {
		names[i] = string(f)
	}
	desc := "must be one of " + strings.Join(names, ", ")
	if s.customProperty {
		desc += ", or <property>.<" + strings.Join(propertyValueColumns, "|") + ">"
	}
	if s.artifactProperty {
		desc += ", or artifacts.<property>.<" + strings.Join(propertyValueColumns, "|") + ">"
	}
	return desc
}
